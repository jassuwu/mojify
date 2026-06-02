package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jass/mojify/packages/core/internal/exporter/fonts"
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

	face, err := fonts.DefaultFace()
	if err != nil {
		return fmt.Errorf("load export font: %w", err)
	}
	rasterizer := NewRasterizer(face)
	renderer := render.DefaultRenderer{}
	renderedFrames := 0

	for {
		rgbFrame, err := media.ReadRawFrame(decodePipe, layout.Grid.Cols, layout.Grid.Rows)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read decoded frame: %w", err)
		}

		charFrame := renderer.Render(rgbFrame, layout.Grid)
		raw, err := rasterizer.Rasterize(charFrame, layout)
		if err != nil {
			return fmt.Errorf("rasterize frame: %w", err)
		}
		if _, err := encodePipe.Write(raw); err != nil {
			return fmt.Errorf("write encoder frame: %w", err)
		}
		renderedFrames++
		progress.Frame(renderedFrames)
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
	progress.Complete(outputPath)
	return nil
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
