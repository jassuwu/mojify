package cli

import (
	"context"
	"io"

	"github.com/jass/mojify/packages/core/internal/exporter"
)

func RunExport(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions) error {
	return exporter.ExportMP4(ctx, inputPath, outputPath, stderr, exporter.Options{
		Width:     options.Width,
		FPS:       options.FPS,
		Bitrate:   options.Bitrate,
		Overwrite: options.Overwrite,
	})
}
