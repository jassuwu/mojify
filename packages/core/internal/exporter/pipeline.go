package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/jass/mojify/packages/core/internal/exporter/fonts"
	"github.com/jass/mojify/packages/core/internal/render"
)

type exportProgress interface {
	Frame(renderedFrames int)
}

type exportFramePipelineOptions struct {
	Workers      int
	ReadFrame    func() (render.RGBFrame, error)
	NewProcessor func() (exportFrameProcessor, error)
	WriteFrame   func([]byte) error
	Progress     exportProgress
	Metrics      *exportMetrics
	Clock        exportClock
}

type exportFrameProcessor func(index int, frame render.RGBFrame) ([]byte, error)

type exportFrameJob struct {
	Index int
	Frame render.RGBFrame
}

type exportFrameResult struct {
	Index   int
	RGB     []byte
	Err     error
	Release bool
}

func resolveExportWorkers(requested int) int {
	if requested > 0 {
		return requested
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		return 1
	}
	return min(workers, 8)
}

func runExportFramePipeline(ctx context.Context, options exportFramePipelineOptions) (int, error) {
	if options.ReadFrame == nil {
		return 0, fmt.Errorf("read frame function is required")
	}
	if options.NewProcessor == nil {
		return 0, fmt.Errorf("frame processor factory is required")
	}
	if options.WriteFrame == nil {
		return 0, fmt.Errorf("write frame function is required")
	}

	workers := resolveExportWorkers(options.Workers)
	clock := options.Clock
	if clock == nil {
		clock = realExportClock{}
	}

	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxInFlight := max(workers*2, 1)
	permits := make(chan struct{}, maxInFlight)
	for range maxInFlight {
		permits <- struct{}{}
	}

	jobs := make(chan exportFrameJob, workers)
	results := make(chan exportFrameResult, workers)
	readErr := make(chan error, 1)

	go readExportFrames(localCtx, jobs, readErr, options.ReadFrame, options.Metrics, clock, permits)

	var workerGroup sync.WaitGroup
	workerGroup.Add(workers)
	for range workers {
		go func() {
			defer workerGroup.Done()
			processor, err := options.NewProcessor()
			if err != nil {
				sendExportFrameResult(localCtx, results, exportFrameResult{Err: fmt.Errorf("create export frame processor: %w", err)})
				return
			}
			for job := range jobs {
				rgb, err := processor(job.Index, job.Frame)
				sendExportFrameResult(localCtx, results, exportFrameResult{Index: job.Index, RGB: rgb, Err: err, Release: true})
				if err != nil {
					return
				}
			}
		}()
	}

	go func() {
		workerGroup.Wait()
		close(results)
	}()

	written, writeErr := writeOrderedFrameResults(localCtx, results, options.WriteFrame, options.Progress, options.Metrics, clock, func() {
		select {
		case permits <- struct{}{}:
		case <-localCtx.Done():
		}
	})
	if writeErr != nil {
		cancel()
		return written, writeErr
	}
	if err := <-readErr; err != nil {
		cancel()
		return written, err
	}
	return written, nil
}

func newExportFrameProcessorFactory(layout Layout, metrics *exportMetrics, clock exportClock, recipe render.Recipe) func() (exportFrameProcessor, error) {
	if clock == nil {
		clock = realExportClock{}
	}
	recipe = recipeOrDefault(recipe)
	return func() (exportFrameProcessor, error) {
		face, err := fonts.DefaultFace()
		if err != nil {
			return nil, fmt.Errorf("load export font: %w", err)
		}
		rasterizer := NewRasterizer(face)
		renderer := render.NewRenderer(recipe)

		return func(_ int, rgbFrame render.RGBFrame) ([]byte, error) {
			renderStart := clock.Now()
			charFrame := renderer.Render(rgbFrame, layout.Grid)
			if metrics != nil {
				metrics.RecordRender(clock.Now().Sub(renderStart))
			}

			rasterizeStart := clock.Now()
			raw, err := rasterizer.Rasterize(charFrame, layout)
			if err != nil {
				return nil, fmt.Errorf("rasterize frame: %w", err)
			}
			if metrics != nil {
				metrics.RecordRasterize(clock.Now().Sub(rasterizeStart))
			}
			return raw, nil
		}, nil
	}
}

func sendExportFrameResult(ctx context.Context, results chan<- exportFrameResult, result exportFrameResult) {
	select {
	case <-ctx.Done():
	case results <- result:
	}
}

func readExportFrames(
	ctx context.Context,
	jobs chan<- exportFrameJob,
	readErr chan<- error,
	readFrame func() (render.RGBFrame, error),
	metrics *exportMetrics,
	clock exportClock,
	permits <-chan struct{},
) {
	defer close(jobs)

	index := 0
	for {
		select {
		case <-ctx.Done():
			readErr <- ctx.Err()
			return
		case <-permits:
		}

		start := clock.Now()
		frame, err := readFrame()
		if errors.Is(err, io.EOF) {
			readErr <- nil
			return
		}
		if err != nil {
			readErr <- fmt.Errorf("read decoded frame: %w", err)
			return
		}
		if metrics != nil {
			metrics.RecordRead(clock.Now().Sub(start))
		}

		select {
		case <-ctx.Done():
			readErr <- ctx.Err()
			return
		case jobs <- exportFrameJob{Index: index, Frame: frame}:
			index++
		}
	}
}

func writeOrderedFrameResults(
	ctx context.Context,
	results <-chan exportFrameResult,
	writeFrame func([]byte) error,
	progress exportProgress,
	metrics *exportMetrics,
	clock exportClock,
	releaseFrame func(),
) (int, error) {
	next := 0
	written := 0
	pending := map[int]exportFrameResult{}
	if clock == nil {
		clock = realExportClock{}
	}

	for {
		result, ok := pending[next]
		if ok {
			delete(pending, next)
		} else {
			var open bool
			select {
			case <-ctx.Done():
				return written, ctx.Err()
			case result, open = <-results:
				if !open {
					if len(pending) != 0 {
						return written, fmt.Errorf("missing frame result %d", next)
					}
					return written, nil
				}
			}
			if result.Err != nil {
				releaseExportFrame(result, releaseFrame)
				return written, result.Err
			}
			if result.Index != next {
				if _, exists := pending[result.Index]; exists {
					releaseExportFrame(result, releaseFrame)
					return written, fmt.Errorf("duplicate frame result %d", result.Index)
				}
				pending[result.Index] = result
				continue
			}
		}

		if result.Err != nil {
			releaseExportFrame(result, releaseFrame)
			return written, result.Err
		}

		start := clock.Now()
		if err := writeFrame(result.RGB); err != nil {
			releaseExportFrame(result, releaseFrame)
			return written, fmt.Errorf("write encoder frame: %w", err)
		}
		if metrics != nil {
			metrics.RecordWrite(clock.Now().Sub(start))
		}
		releaseExportFrame(result, releaseFrame)

		written++
		if progress != nil {
			progress.Frame(written)
		}
		next++
	}
}

func releaseExportFrame(result exportFrameResult, releaseFrame func()) {
	if result.Release && releaseFrame != nil {
		releaseFrame()
	}
}
