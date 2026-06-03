package cli

import (
	"context"
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
	}
	if err := exporter.CheckOutputPath(outputPath, exportOptions); err != nil {
		return err
	}

	resolved, err := resolveSourceMediaWithOptions(ctx, inputPath, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: runnerOptions.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()

	exportFn := runnerOptions.Export
	if exportFn == nil {
		exportFn = exporter.Export
	}
	return exportFn(ctx, resolved.Path, outputPath, stderr, exportOptions)
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return xterm.IsTerminal(int(file.Fd()))
}
