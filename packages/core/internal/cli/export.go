package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jass/mojify/packages/core/internal/exporter"
	xterm "golang.org/x/term"
)

type exportRunnerOptions struct {
	YTDLPPath string
	Export    func(context.Context, string, string, io.Writer, exporter.Options) error
}

func RunExport(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions) error {
	return runExportWithOptions(ctx, inputPath, outputPath, stderr, options, exportRunnerOptions{})
}

func runExportWithOptions(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions, runnerOptions exportRunnerOptions) error {
	exportOptions := exporter.Options{
		Width:               options.Width,
		FPS:                 options.FPS,
		Bitrate:             options.Bitrate,
		Overwrite:           options.Overwrite,
		ProgressInteractive: isTerminalWriter(stderr),
		Stats:               options.Stats,
		Workers:             options.Workers,
		InputLabel:          inputPath,
		HasAt:               options.HasAt,
		AtSeconds:           options.AtSeconds,
		HasDuration:         options.HasDuration,
		DurationSeconds:     options.DurationSeconds,
		Recipe:              options.Recipe,
	}

	resolved, err := resolveSourceMediaWithOptions(ctx, inputPath, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: runnerOptions.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()

	if err := validateStillSourceExport(resolved, outputPath, options); err != nil {
		return err
	}

	exportFn := runnerOptions.Export
	if exportFn == nil {
		exportFn = exporter.Export
	}
	return exportFn(ctx, resolved.Path, outputPath, stderr, exportOptions)
}

func validateStillSourceExport(resolved resolvedSourceMedia, outputPath string, options ExportOptions) error {
	if resolved.Kind != sourceKindStill {
		return nil
	}
	if options.HasAt {
		return fmt.Errorf("export --at is not valid for still image sources")
	}
	if options.HasDuration {
		return fmt.Errorf("export --duration is not valid for still image sources")
	}
	format, err := exporter.ResolveOutputFormat(outputPath)
	if err != nil {
		return err
	}
	if !format.SingleFrame {
		return fmt.Errorf("still image sources can only export single-frame outputs: .png, .jpg, .jpeg, .txt, .ansi")
	}
	return nil
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return xterm.IsTerminal(int(file.Fd()))
}
