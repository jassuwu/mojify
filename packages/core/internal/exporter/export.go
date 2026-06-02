package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/render"
)

func ExportMP4(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) (err error) {
	if err := checkOutputPath(outputPath, options); err != nil {
		return err
	}

	info, err := media.ProbeContext(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	layout, err := ResolveLayout(InputInfo{Width: info.Width, Height: info.Height, FPS: info.FPS}, options)
	if err != nil {
		return err
	}
	progress := newProgressReporter(stderr, progressReporterOptions{
		Interactive: options.ProgressInteractive,
		TotalFrames: estimateExportFrameTotal(InputProgressInfo{
			SourceFPS:       info.FPS,
			FrameCount:      info.FrameCount,
			DurationSeconds: info.DurationSeconds,
		}, layout, options),
		Now: options.ProgressClock,
	})
	progress.Start(inputPath, outputPath, layout)
	defer func() {
		if err != nil {
			progress.ErrorLine()
		}
	}()

	decodeCmd, decodePipe, err := media.StartExportDecoderContext(ctx, inputPath, layout.Grid.Cols, layout.Grid.Rows, options.FPS)
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	decodeCleaned := false
	defer func() {
		if !decodeCleaned {
			_ = cleanupProcess(decodeCmd, decodePipe)
		}
	}()

	encodeCmd, encodePipe, err := media.StartMP4EncoderContext(ctx, media.MP4EncodeOptions{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Width:      layout.OutputWidth,
		Height:     layout.OutputHeight,
		FPS:        layout.FPS,
		Bitrate:    options.Bitrate,
		Overwrite:  options.Overwrite,
	}, progress.lineSafeWriter(stderr))
	if err != nil {
		return fmt.Errorf("start encoder: %w", err)
	}
	encodeClosed := false
	defer func() {
		if !encodeClosed {
			_ = encodePipe.Close()
			_ = encodeCmd.Wait()
		}
	}()

	workers := resolveExportWorkers(options.Workers)
	metricsClock := exportMetricsClock(options)
	var metrics *exportMetrics
	if options.Stats {
		metrics = newExportMetrics(workers, metricsClock)
		metrics.Start()
	}

	renderedFrames, err := runExportFramePipeline(ctx, exportFramePipelineOptions{
		Workers: workers,
		ReadFrame: func() (render.RGBFrame, error) {
			return media.ReadRawFrame(decodePipe, layout.Grid.Cols, layout.Grid.Rows)
		},
		NewProcessor: newExportFrameProcessorFactory(layout, metrics, metricsClock),
		WriteFrame: func(raw []byte) error {
			_, err := encodePipe.Write(raw)
			return err
		},
		Progress: progress,
		Metrics:  metrics,
		Clock:    metricsClock,
	})
	if err != nil {
		return err
	}

	if err := decodePipe.Close(); err != nil {
		return fmt.Errorf("close decoder pipe: %w", err)
	}
	decodeErr := decodeCmd.Wait()
	decodeCleaned = true

	if decodeErr != nil {
		return fmt.Errorf("decoder failed: %w", decodeErr)
	}

	if err := encodePipe.Close(); err != nil {
		return fmt.Errorf("close encoder pipe: %w", err)
	}
	encodeClosed = true

	progress.AllFramesWritten(renderedFrames)
	progress.Finalizing()

	encodeErr := encodeCmd.Wait()

	if encodeErr != nil {
		return fmt.Errorf("encoder failed: %w", encodeErr)
	}
	if metrics != nil {
		metrics.Finish()
	}
	progress.Complete(outputPath)
	printExportStats(stderr, options, metrics)
	return nil
}

func exportMetricsClock(options Options) exportClock {
	if options.MetricsClock != nil {
		return exportClockFunc(options.MetricsClock)
	}
	return realExportClock{}
}

func printExportStats(w io.Writer, options Options, metrics *exportMetrics) {
	if !options.Stats || metrics == nil || w == nil {
		return
	}
	_, _ = fmt.Fprint(w, metrics.Summary())
}

func checkOutputPath(outputPath string, options Options) error {
	if outputPath == "" {
		return fmt.Errorf("missing output path")
	}
	_, err := os.Stat(outputPath)
	if err == nil && !options.Overwrite {
		return fmt.Errorf("output exists: %s; pass --overwrite to replace it", outputPath)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat output: %w", err)
	}
	return nil
}

func cleanupProcess(cmd *exec.Cmd, pipe io.Closer) error {
	_ = pipe.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return cmd.Wait()
}
