package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jass/mojify/packages/core/internal/exporter/fonts"
	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/render"
)

var (
	exportTimeBasedMediaFunc   = exportTimeBasedMedia
	exportSingleFrameMediaFunc = exportSingleFrameMedia
	exportTextFunc             = exportText
)

func ExportMP4(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) error {
	format := OutputFormat{Extension: ".mp4", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true}
	options.Format = format
	return exportTimeBasedMedia(ctx, inputPath, outputPath, stderr, options, format)
}

func Export(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) error {
	if err := CheckOutputPath(outputPath, options); err != nil {
		return err
	}
	format, err := resolveOptionsFormat(outputPath, options)
	if err != nil {
		return err
	}
	options.Format = format
	switch {
	case format.TimeBased:
		return exportTimeBasedMediaFunc(ctx, inputPath, outputPath, stderr, options, format)
	case format.Text:
		return exportTextFunc(ctx, inputPath, outputPath, stderr, options, format)
	case format.SingleFrame:
		return exportSingleFrameMediaFunc(ctx, inputPath, outputPath, stderr, options, format)
	default:
		return fmt.Errorf("unsupported export format family %q", format.Family)
	}
}

func exportTimeBasedMedia(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) (err error) {
	options.Format = format

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
			DurationSeconds: progressDuration(info.DurationSeconds, options),
		}, layout, options),
		Now:             options.ProgressClock,
		Label:           progressLabelForFormat(format),
		FinalizingLabel: finalizingLabelForFormat(format),
	})
	inputLabel := options.InputLabel
	if inputLabel == "" {
		inputLabel = inputPath
	}
	progress.Start(inputLabel, outputPath, layout)
	defer func() {
		if err != nil {
			progress.ErrorLine()
		}
	}()

	decodeCmd, decodePipe, err := media.StartExportDecoderWithOptions(ctx, media.ExportDecodeOptions{
		Path:            inputPath,
		Width:           layout.Grid.Cols,
		Height:          layout.Grid.Rows,
		FPS:             options.FPS,
		HasAt:           options.HasAt,
		AtSeconds:       options.AtSeconds,
		HasDuration:     options.HasDuration,
		DurationSeconds: options.DurationSeconds,
	})
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	decodeCleaned := false
	defer func() {
		if !decodeCleaned {
			_ = cleanupProcess(decodeCmd, decodePipe)
		}
	}()

	encodeFormat, err := encodeFormatForOutput(format)
	if err != nil {
		return err
	}
	encodeCmd, encodePipe, err := media.StartRawVideoEncoderContext(ctx, media.RawVideoEncodeOptions{
		Format:          encodeFormat,
		InputPath:       inputPath,
		OutputPath:      outputPath,
		Width:           layout.OutputWidth,
		Height:          layout.OutputHeight,
		FPS:             layout.FPS,
		Bitrate:         options.Bitrate,
		Overwrite:       options.Overwrite,
		IncludeAudio:    format.SupportsAudio,
		HasAt:           options.HasAt,
		AtSeconds:       options.AtSeconds,
		HasDuration:     options.HasDuration,
		DurationSeconds: options.DurationSeconds,
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
		NewProcessor: newExportFrameProcessorFactory(layout, metrics, metricsClock, recipeOrDefault(options.Recipe)),
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

func exportSingleFrameMedia(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
	options.Format = format
	info, err := media.ProbeContext(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	layout, err := ResolveLayout(InputInfo{Width: info.Width, Height: info.Height, FPS: 1}, options)
	if err != nil {
		return err
	}
	raw, err := renderSingleRawFrame(ctx, inputPath, layout, options)
	if err != nil {
		return err
	}
	encodeFormat, err := encodeFormatForOutput(format)
	if err != nil {
		return err
	}
	encodeCmd, encodePipe, err := media.StartRawVideoEncoderContext(ctx, media.RawVideoEncodeOptions{
		Format:     encodeFormat,
		OutputPath: outputPath,
		Width:      layout.OutputWidth,
		Height:     layout.OutputHeight,
		FPS:        1,
		Overwrite:  options.Overwrite,
	}, stderr)
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
	if _, err := encodePipe.Write(raw); err != nil {
		return fmt.Errorf("write encoder frame: %w", err)
	}
	if err := encodePipe.Close(); err != nil {
		return fmt.Errorf("close encoder pipe: %w", err)
	}
	encodeClosed = true
	if err := encodeCmd.Wait(); err != nil {
		return fmt.Errorf("encoder failed: %w", err)
	}
	fmt.Fprintf(stderr, "export complete: %s\n", outputPath)
	return nil
}

func exportText(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
	options.Format = format
	info, err := media.ProbeContext(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	layout, err := ResolveLayout(InputInfo{Width: info.Width, Height: info.Height, FPS: 1}, options)
	if err != nil {
		return err
	}
	charFrame, err := renderSingleCharacterFrame(ctx, inputPath, layout, options)
	if err != nil {
		return err
	}
	text, err := SerializeTextFrame(charFrame, format)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write text output: %w", err)
	}
	fmt.Fprintf(stderr, "export complete: %s\n", outputPath)
	return nil
}

func exportSingleTextFrameForTest(frame render.RGBFrame, outputPath string, options Options) error {
	layout, err := ResolveLayout(InputInfo{Width: frame.Width, Height: frame.Height, FPS: 1}, options)
	if err != nil {
		return err
	}
	charFrame := render.NewRenderer(recipeOrDefault(options.Recipe)).Render(frame, layout.Grid)
	text, err := SerializeTextFrame(charFrame, options.Format)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(text), 0o644)
}

func renderSingleRawFrame(ctx context.Context, inputPath string, layout Layout, options Options) ([]byte, error) {
	charFrame, err := renderSingleCharacterFrame(ctx, inputPath, layout, options)
	if err != nil {
		return nil, err
	}
	face, err := fonts.DefaultFace()
	if err != nil {
		return nil, fmt.Errorf("load export font: %w", err)
	}
	raw, err := NewRasterizer(face).Rasterize(charFrame, layout)
	if err != nil {
		return nil, fmt.Errorf("rasterize frame: %w", err)
	}
	return raw, nil
}

func renderSingleCharacterFrame(ctx context.Context, inputPath string, layout Layout, options Options) (render.CharacterFrame, error) {
	decodeCmd, decodePipe, err := media.StartExportDecoderWithOptions(ctx, media.ExportDecodeOptions{
		Path:      inputPath,
		Width:     layout.Grid.Cols,
		Height:    layout.Grid.Rows,
		HasAt:     options.HasAt,
		AtSeconds: options.AtSeconds,
	})
	if err != nil {
		return render.CharacterFrame{}, fmt.Errorf("start decoder: %w", err)
	}
	defer func() {
		_ = cleanupProcess(decodeCmd, decodePipe)
	}()
	rgbFrame, err := media.ReadRawFrame(decodePipe, layout.Grid.Cols, layout.Grid.Rows)
	if err != nil {
		return render.CharacterFrame{}, fmt.Errorf("read decoded frame: %w", err)
	}
	return render.NewRenderer(recipeOrDefault(options.Recipe)).Render(rgbFrame, layout.Grid), nil
}

func recipeOrDefault(recipe render.Recipe) render.Recipe {
	if recipe.Name == "" {
		return render.DefaultRecipe()
	}
	return recipe
}

func encodeFormatForOutput(format OutputFormat) (media.EncodeFormat, error) {
	switch format.Extension {
	case ".mp4":
		return media.EncodeFormatMP4, nil
	case ".webm":
		return media.EncodeFormatWebM, nil
	case ".mov":
		return media.EncodeFormatMOV, nil
	case ".gif":
		return media.EncodeFormatGIF, nil
	case ".apng":
		return media.EncodeFormatAPNG, nil
	case ".png":
		return media.EncodeFormatPNG, nil
	case ".jpg", ".jpeg":
		return media.EncodeFormatJPEG, nil
	default:
		return "", fmt.Errorf("unsupported media export format %q", format.Extension)
	}
}

func progressDuration(sourceDuration float64, options Options) float64 {
	if options.HasAt && sourceDuration > 0 {
		remaining := sourceDuration - options.AtSeconds
		if remaining <= 0 {
			return 0
		}
		if options.HasDuration && options.DurationSeconds < remaining {
			return options.DurationSeconds
		}
		return remaining
	}
	if options.HasDuration {
		return options.DurationSeconds
	}
	return sourceDuration
}

func progressLabelForFormat(format OutputFormat) string {
	switch format.Family {
	case OutputFamilyAnimated:
		return "animated export"
	case OutputFamilyVideo:
		return "video"
	default:
		return "export"
	}
}

func finalizingLabelForFormat(format OutputFormat) string {
	if format.Extension == "" {
		return "finalizing output..."
	}
	return "finalizing " + strings.TrimPrefix(format.Extension, ".") + "..."
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

func CheckOutputPath(outputPath string, options Options) error {
	if outputPath == "" {
		return fmt.Errorf("missing output path")
	}
	format, err := resolveOptionsFormat(outputPath, options)
	if err != nil {
		return err
	}
	if options.HasDuration && format.SingleFrame {
		return fmt.Errorf("export --duration is valid only for video and animated outputs")
	}
	_, err = os.Stat(outputPath)
	if err == nil && !options.Overwrite {
		return fmt.Errorf("output exists: %s; pass --overwrite to replace it", outputPath)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat output: %w", err)
	}
	return nil
}

func resolveOptionsFormat(outputPath string, options Options) (OutputFormat, error) {
	if options.Format.Extension != "" {
		return options.Format, nil
	}
	return ResolveOutputFormat(outputPath)
}

func checkOutputPath(outputPath string, options Options) error {
	return CheckOutputPath(outputPath, options)
}

func cleanupProcess(cmd *exec.Cmd, pipe io.Closer) error {
	_ = pipe.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return cmd.Wait()
}
