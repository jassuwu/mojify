package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	xterm "golang.org/x/term"

	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/player"
	"github.com/jass/mojify/packages/core/internal/render"
	"github.com/jass/mojify/packages/core/internal/terminal"
)

type PlayOptions struct {
	Stats   bool
	NoAudio bool
}

type playRunnerOptions struct {
	YTDLPPath string
	Probe     func(context.Context, string) (media.Info, error)
}

func RunPlay(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer, stderr io.Writer, options PlayOptions) error {
	return runPlayWithOptions(ctx, inputPath, stdin, stdout, stderr, options, playRunnerOptions{})
}

func runPlayWithOptions(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer, stderr io.Writer, options PlayOptions, runnerOptions playRunnerOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resolved, err := resolveSourceMediaWithOptions(ctx, inputPath, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: runnerOptions.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()
	if resolved.Kind == sourceKindStill {
		return fmt.Errorf("still image sources cannot be played; use mojify export <source> <output> instead")
	}
	inputPath = resolved.Path

	probe := runnerOptions.Probe
	if probe == nil {
		probe = media.ProbeContext
	}
	info, err := probe(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}

	cols, rows, err := xterm.GetSize(int(stdin.Fd()))
	if err != nil {
		cols, rows = 120, 40
	}
	grid := render.FitGrid(
		render.InputSize{Width: info.Width, Height: info.Height},
		render.TerminalSize{Cols: cols, Rows: rows - 1},
	)
	var metrics *playback.Metrics
	if options.Stats {
		metrics = playback.NewMetrics(grid.Cols, grid.Rows)
	}

	decodeWidth := min(info.Width, 640)
	decodeHeight := max(1, decodeWidth*info.Height/info.Width)
	cmd, pipe, err := media.StartDecoderContext(ctx, inputPath, decodeWidth, decodeHeight)
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	decoderCleaned := false
	defer func() {
		if !decoderCleaned {
			_ = cleanupDecoder(cmd, pipe)
		}
	}()

	frames := make(chan render.CharacterFrame, 12)
	renderErr := make(chan error, 1)
	renderDone := make(chan struct{})
	renderer := render.DefaultRenderer{}
	if metrics != nil {
		metrics.Start(time.Now())
	}
	go func() {
		defer close(renderDone)
		defer close(frames)
		for {
			rgb, err := media.ReadRawFrame(pipe, decodeWidth, decodeHeight)
			if err != nil {
				if err != io.EOF {
					renderErr <- fmt.Errorf("read decoded frame: %w", err)
				}
				return
			}
			start := time.Now()
			frame := renderer.Render(rgb, grid)
			if metrics != nil {
				metrics.RecordRendered(time.Since(start))
			}
			select {
			case <-ctx.Done():
				return
			case frames <- frame:
			}
		}
	}()

	presenter := terminal.Presenter{Out: stdout, Metrics: metrics}
	if err := presenter.Start(); err != nil {
		return err
	}
	presenterStopped := false
	defer func() {
		if !presenterStopped {
			_ = presenter.Stop()
		}
	}()

	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   !options.NoAudio,
		HasStream: info.HasAudio,
	})

	controls, stopControls, err := startPlaybackControls(ctx, cancel, stdin, audio)
	if err != nil {
		return err
	}
	controlsStopped := false
	defer func() {
		if !controlsStopped {
			stopControls()
		}
	}()

	audio.Start(ctx, inputPath)
	for _, warning := range audio.DrainWarnings() {
		fmt.Fprintln(stderr, warning)
	}
	defer audio.Stop()

	playErr := player.PlayWithControls(ctx, frames, &presenter, info.FPS, controls, metrics)
	audio.Stop()
	ctxErr := ctx.Err()
	if ctxErr != nil || playErr != nil {
		cancel()
		_ = pipe.Close()
	}
	<-renderDone
	if metrics != nil {
		metrics.Finish(time.Now())
	}

	var frameErr error
	select {
	case err := <-renderErr:
		frameErr = err
	default:
	}

	shouldKillDecoder := ctxErr != nil || playErr != nil || frameErr != nil
	var decoderErr error
	if shouldKillDecoder {
		cancel()
		decoderErr = cleanupDecoder(cmd, pipe)
	} else {
		decoderErr = waitDecoder(cmd)
	}
	decoderCleaned = true
	cancel()
	resultErr := playbackResult(ctxErr, playErr, frameErr, decoderErr)
	stopControls()
	controlsStopped = true
	if err := presenter.Stop(); err != nil && resultErr == nil {
		resultErr = err
	}
	presenterStopped = true
	for _, warning := range audio.DrainWarnings() {
		fmt.Fprintln(stderr, warning)
	}
	printStats(stderr, options, metrics, audio.Status())
	return resultErr
}

func cleanupDecoder(cmd *exec.Cmd, pipe io.Closer) error {
	_ = pipe.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return cmd.Wait()
}

func waitDecoder(cmd *exec.Cmd) error {
	return cmd.Wait()
}

func printStats(w io.Writer, options PlayOptions, metrics *playback.Metrics, audioStatus playbackAudioStatus) {
	if !options.Stats || metrics == nil {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, metrics.Summary())
	printAudioStats(w, audioStatus)
}

type audioPauseTarget interface {
	TogglePause()
}

func startPlaybackControls(ctx context.Context, cancel context.CancelFunc, stdin *os.File, audio audioPauseTarget) (<-chan player.Control, func(), error) {
	fd := int(stdin.Fd())
	if !xterm.IsTerminal(fd) {
		return nil, func() {}, nil
	}

	state, err := xterm.MakeRaw(fd)
	if err != nil {
		return nil, nil, fmt.Errorf("enable playback controls: %w", err)
	}

	controlCtx, stopControls := context.WithCancel(ctx)
	terminalControls := make(chan terminal.Control, 4)
	playerControls := make(chan player.Control, 4)

	go terminal.ReadControls(controlCtx, stdin, terminalControls)
	go bridgeTerminalControls(controlCtx, cancel, terminalControls, playerControls, audio)

	cleanup := func() {
		stopControls()
		_ = xterm.Restore(fd, state)
	}
	return playerControls, cleanup, nil
}

func bridgeTerminalControls(ctx context.Context, cancel context.CancelFunc, in <-chan terminal.Control, out chan<- player.Control, audio audioPauseTarget) {
	defer close(out)
	for {
		select {
		case <-ctx.Done():
			return
		case control, ok := <-in:
			if !ok {
				return
			}
			var playerControl player.Control
			switch control {
			case terminal.Quit:
				playerControl = player.Quit
				cancel()
			case terminal.TogglePause:
				if audio != nil {
					audio.TogglePause()
				}
				playerControl = player.TogglePause
			default:
				continue
			}
			select {
			case out <- playerControl:
			case <-ctx.Done():
				return
			}
		}
	}
}

func playbackResult(ctxErr error, playErr error, renderErr error, decoderErr error) error {
	wasCancelled := ctxErr != nil || errors.Is(playErr, player.ErrCancelled) || errors.Is(playErr, player.ErrQuit)
	if errors.Is(playErr, player.ErrCancelled) || errors.Is(playErr, player.ErrQuit) {
		playErr = nil
	}
	if playErr != nil {
		return playErr
	}
	if wasCancelled {
		return nil
	}
	if renderErr != nil {
		return renderErr
	}
	if decoderErr != nil && !wasCancelled {
		return fmt.Errorf("decoder failed: %w", decoderErr)
	}
	return nil
}
