package exporter

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestResolveExportWorkersUsesExplicitValue(t *testing.T) {
	if got := resolveExportWorkers(3); got != 3 {
		t.Fatalf("resolveExportWorkers(3) = %d, want 3", got)
	}
}

func TestResolveExportWorkersDefaultsToPositiveBound(t *testing.T) {
	got := resolveExportWorkers(0)
	if got < 1 {
		t.Fatalf("resolveExportWorkers(0) = %d, want positive", got)
	}
	if got > 8 {
		t.Fatalf("resolveExportWorkers(0) = %d, want at most 8", got)
	}
}

func TestWriteOrderedFrameResultsWritesInFrameOrder(t *testing.T) {
	results := make(chan exportFrameResult, 3)
	results <- exportFrameResult{Index: 1, RGB: []byte{1}}
	results <- exportFrameResult{Index: 0, RGB: []byte{0}}
	results <- exportFrameResult{Index: 2, RGB: []byte{2}}
	close(results)

	writes := make([][]byte, 0, 3)
	written, err := writeOrderedFrameResults(context.Background(), results, func(data []byte) error {
		writes = append(writes, append([]byte(nil), data...))
		return nil
	}, nil, nil, fakeExportClock(time.Unix(0, 0)), nil)
	if err != nil {
		t.Fatalf("writeOrderedFrameResults returned error: %v", err)
	}
	if written != 3 {
		t.Fatalf("written = %d, want 3", written)
	}
	if !reflect.DeepEqual(writes, [][]byte{{0}, {1}, {2}}) {
		t.Fatalf("writes = %#v, want ordered frame bytes", writes)
	}
}

func TestWriteOrderedFrameResultsReturnsResultError(t *testing.T) {
	wantErr := errors.New("rasterize failed")
	results := make(chan exportFrameResult, 1)
	results <- exportFrameResult{Index: 0, Err: wantErr}
	close(results)

	_, err := writeOrderedFrameResults(context.Background(), results, func([]byte) error {
		t.Fatal("writer should not be called for errored result")
		return nil
	}, nil, nil, fakeExportClock(time.Unix(0, 0)), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestWriteOrderedFrameResultsReturnsWriterError(t *testing.T) {
	wantErr := errors.New("encoder write failed")
	results := make(chan exportFrameResult, 1)
	results <- exportFrameResult{Index: 0, RGB: []byte{0}}
	close(results)

	_, err := writeOrderedFrameResults(context.Background(), results, func([]byte) error {
		return wantErr
	}, nil, nil, fakeExportClock(time.Unix(0, 0)), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestWriteOrderedFrameResultsUsesInjectedClockForMetrics(t *testing.T) {
	results := make(chan exportFrameResult, 1)
	results <- exportFrameResult{Index: 0, RGB: []byte{0}}
	close(results)
	clock := &advancingExportClock{current: time.Unix(0, 0), step: 5 * time.Millisecond}
	metrics := newExportMetrics(1, clock)
	metrics.Start()

	_, err := writeOrderedFrameResults(context.Background(), results, func([]byte) error {
		return nil
	}, nil, metrics, clock, nil)
	if err != nil {
		t.Fatalf("writeOrderedFrameResults returned error: %v", err)
	}
	metrics.Finish()
	if want := "avg encoder write time: 5ms"; !strings.Contains(metrics.Summary(), want) {
		t.Fatalf("Summary missing %q in:\n%s", want, metrics.Summary())
	}
}

func TestRunExportFramePipelineWritesFramesAndUpdatesProgress(t *testing.T) {
	frames := []render.RGBFrame{
		render.NewRGBFrame(1, 1, []byte{1, 0, 0}),
		render.NewRGBFrame(1, 1, []byte{2, 0, 0}),
		render.NewRGBFrame(1, 1, []byte{3, 0, 0}),
	}
	next := 0
	writes := make([][]byte, 0, len(frames))
	progress := &countingProgress{}

	written, err := runExportFramePipeline(context.Background(), exportFramePipelineOptions{
		Workers: 2,
		ReadFrame: func() (render.RGBFrame, error) {
			if next == len(frames) {
				return render.RGBFrame{}, io.EOF
			}
			frame := frames[next]
			next++
			return frame, nil
		},
		NewProcessor: processorFactory(func(index int, frame render.RGBFrame) ([]byte, error) {
			return []byte{byte(index)}, nil
		}),
		WriteFrame: func(data []byte) error {
			writes = append(writes, append([]byte(nil), data...))
			return nil
		},
		Progress: progress,
	})
	if err != nil {
		t.Fatalf("runExportFramePipeline returned error: %v", err)
	}
	if written != 3 {
		t.Fatalf("written = %d, want 3", written)
	}
	if !reflect.DeepEqual(writes, [][]byte{{0}, {1}, {2}}) {
		t.Fatalf("writes = %#v, want ordered frame bytes", writes)
	}
	if !reflect.DeepEqual(progress.frames, []int{1, 2, 3}) {
		t.Fatalf("progress frames = %#v, want [1 2 3]", progress.frames)
	}
}

func TestRunExportFramePipelineReturnsProcessErrorWithoutHanging(t *testing.T) {
	wantErr := errors.New("process failed")

	_, err := runPipelineWithTimeout(t, exportFramePipelineOptions{
		Workers:   2,
		ReadFrame: finiteFrameReader(3),
		NewProcessor: processorFactory(func(index int, frame render.RGBFrame) ([]byte, error) {
			if index == 1 {
				return nil, wantErr
			}
			return []byte{byte(index)}, nil
		}),
		WriteFrame: func([]byte) error {
			return nil
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestRunExportFramePipelineReturnsReadErrorWithoutHanging(t *testing.T) {
	wantErr := errors.New("decode failed")
	reads := 0

	_, err := runPipelineWithTimeout(t, exportFramePipelineOptions{
		Workers: 2,
		ReadFrame: func() (render.RGBFrame, error) {
			if reads == 1 {
				return render.RGBFrame{}, wantErr
			}
			reads++
			return render.NewRGBFrame(1, 1, []byte{byte(reads), 0, 0}), nil
		},
		NewProcessor: processorFactory(func(index int, frame render.RGBFrame) ([]byte, error) {
			return []byte{byte(index)}, nil
		}),
		WriteFrame: func([]byte) error {
			return nil
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestRunExportFramePipelineReturnsWriteErrorWithoutHanging(t *testing.T) {
	wantErr := errors.New("encoder failed")

	_, err := runPipelineWithTimeout(t, exportFramePipelineOptions{
		Workers:      2,
		ReadFrame:    finiteFrameReader(3),
		NewProcessor: processorFactory(func(index int, frame render.RGBFrame) ([]byte, error) { return []byte{byte(index)}, nil }),
		WriteFrame: func([]byte) error {
			return wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestRunExportFramePipelineCreatesProcessorPerWorker(t *testing.T) {
	var created atomic.Int32

	written, err := runPipelineWithTimeout(t, exportFramePipelineOptions{
		Workers:   3,
		ReadFrame: finiteFrameReader(3),
		NewProcessor: func() (exportFrameProcessor, error) {
			created.Add(1)
			return func(index int, frame render.RGBFrame) ([]byte, error) {
				return []byte{byte(index)}, nil
			}, nil
		},
		WriteFrame: func([]byte) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runExportFramePipeline returned error: %v", err)
	}
	if written != 3 {
		t.Fatalf("written = %d, want 3", written)
	}
	if got := created.Load(); got != 3 {
		t.Fatalf("created processors = %d, want 3", got)
	}
}

func TestReadExportFramesWaitsForInFlightPermit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan exportFrameJob, 10)
	readErr := make(chan error, 1)
	permits := make(chan struct{}, 2)
	permits <- struct{}{}
	permits <- struct{}{}
	var reads atomic.Int32

	go readExportFrames(ctx, jobs, readErr, func() (render.RGBFrame, error) {
		reads.Add(1)
		return render.NewRGBFrame(1, 1, []byte{0, 0, 0}), nil
	}, nil, fakeExportClock(time.Unix(0, 0)), permits)

	waitForReads(t, &reads, 2)
	time.Sleep(25 * time.Millisecond)
	if got := reads.Load(); got != 2 {
		t.Fatalf("reads = %d, want blocked at 2 without another permit", got)
	}

	permits <- struct{}{}
	waitForReads(t, &reads, 3)
	cancel()
	select {
	case <-readErr:
	case <-time.After(2 * time.Second):
		t.Fatal("readExportFrames did not stop after cancellation")
	}
}

type countingProgress struct {
	frames []int
}

func (p *countingProgress) Frame(renderedFrames int) {
	p.frames = append(p.frames, renderedFrames)
}

func finiteFrameReader(total int) func() (render.RGBFrame, error) {
	next := 0
	return func() (render.RGBFrame, error) {
		if next == total {
			return render.RGBFrame{}, io.EOF
		}
		next++
		return render.NewRGBFrame(1, 1, []byte{byte(next), 0, 0}), nil
	}
}

func processorFactory(processor exportFrameProcessor) func() (exportFrameProcessor, error) {
	return func() (exportFrameProcessor, error) {
		return processor, nil
	}
}

func waitForReads(t *testing.T, reads *atomic.Int32, want int32) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if reads.Load() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("reads = %d, want at least %d", reads.Load(), want)
}

func runPipelineWithTimeout(t *testing.T, options exportFramePipelineOptions) (int, error) {
	t.Helper()

	type result struct {
		written int
		err     error
	}
	done := make(chan result, 1)
	go func() {
		written, err := runExportFramePipeline(context.Background(), options)
		done <- result{written: written, err: err}
	}()

	select {
	case result := <-done:
		return result.written, result.err
	case <-time.After(2 * time.Second):
		t.Fatal("runExportFramePipeline did not return")
		return 0, nil
	}
}

type advancingExportClock struct {
	current time.Time
	step    time.Duration
}

func (c *advancingExportClock) Now() time.Time {
	now := c.current
	c.current = c.current.Add(c.step)
	return now
}
