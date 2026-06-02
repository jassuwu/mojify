package cli

import (
	"context"
	"io"
	"os"

	"github.com/jass/mojify/packages/core/internal/exporter"
	xterm "golang.org/x/term"
)

func RunExport(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions) error {
	return exporter.ExportMP4(ctx, inputPath, outputPath, stderr, exporter.Options{
		Width:               options.Width,
		FPS:                 options.FPS,
		Bitrate:             options.Bitrate,
		Overwrite:           options.Overwrite,
		ProgressInteractive: isTerminalWriter(stderr),
	})
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return xterm.IsTerminal(int(file.Fd()))
}
