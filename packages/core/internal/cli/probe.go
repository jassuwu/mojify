package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/render"
)

type probeOutput struct {
	OriginalSource      string
	ResolvedDisplayName string
	Width               int
	Height              int
	FPS                 float64
	FrameCount          int
	DurationSeconds     float64
	HasAudio            bool
	RenderCols          int
	RenderRows          int
}

type probeRunnerOptions struct {
	YTDLPPath string
	Probe     func(context.Context, string) (media.Info, error)
}

func RunProbe(ctx context.Context, source string, stdout io.Writer, stderr io.Writer) error {
	return runProbeWithOptions(ctx, source, stdout, stderr, probeRunnerOptions{})
}

func runProbeWithOptions(ctx context.Context, source string, stdout io.Writer, stderr io.Writer, options probeRunnerOptions) error {
	resolved, err := resolveSourceMediaWithOptions(ctx, source, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: options.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()

	probe := options.Probe
	if probe == nil {
		probe = media.ProbeContext
	}
	info, err := probe(ctx, resolved.Path)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}

	grid := render.FitGrid(
		render.InputSize{Width: info.Width, Height: info.Height},
		render.TerminalSize{Cols: 120, Rows: 40},
	)
	printProbeInfo(stdout, probeOutput{
		OriginalSource:      resolved.Original,
		ResolvedDisplayName: resolvedDisplayName(resolved),
		Width:               info.Width,
		Height:              info.Height,
		FPS:                 info.FPS,
		FrameCount:          info.FrameCount,
		DurationSeconds:     info.DurationSeconds,
		HasAudio:            info.HasAudio,
		RenderCols:          grid.Cols,
		RenderRows:          grid.Rows,
	})
	return nil
}

func resolvedDisplayName(source resolvedSourceMedia) string {
	if !source.Temporary {
		return ""
	}
	return source.DisplayName
}

func printProbeInfo(w io.Writer, output probeOutput) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "input: %s\n", output.OriginalSource)
	if output.ResolvedDisplayName != "" {
		fmt.Fprintf(w, "resolved-source: %s\n", output.ResolvedDisplayName)
	}
	fmt.Fprintf(w, "video: %dx%d\n", output.Width, output.Height)
	fmt.Fprintf(w, "fps: %.3f\n", output.FPS)
	fmt.Fprintf(w, "frames: %d\n", output.FrameCount)
	fmt.Fprintf(w, "duration: %.3fs\n", output.DurationSeconds)
	if output.HasAudio {
		fmt.Fprintln(w, "audio: yes")
	} else {
		fmt.Fprintln(w, "audio: no")
	}
	fmt.Fprintf(w, "render-grid: %dx%d (sample terminal 120x40)\n", output.RenderCols, output.RenderRows)
}
