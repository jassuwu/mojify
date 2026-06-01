package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	xterm "golang.org/x/term"

	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/player"
	"github.com/jass/mojify/packages/core/internal/render"
	"github.com/jass/mojify/packages/core/internal/terminal"
)

func RunPlay(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	info, err := media.ProbeContext(ctx, inputPath)
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

	decodeWidth := min(info.Width, 640)
	decodeHeight := max(1, decodeWidth*info.Height/info.Width)
	cmd, pipe, err := media.StartDecoderContext(ctx, inputPath, decodeWidth, decodeHeight)
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	decoderDone := make(chan error, 1)
	go func() {
		decoderDone <- cmd.Wait()
	}()
	decoderCleaned := false
	defer func() {
		if !decoderCleaned {
			_ = cleanupDecoder(cmd, pipe, decoderDone)
		}
	}()

	frames := make(chan render.CharacterFrame, 12)
	renderErr := make(chan error, 1)
	renderDone := make(chan struct{})
	renderer := render.DefaultRenderer{}
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
			select {
			case <-ctx.Done():
				return
			case frames <- renderer.Render(rgb, grid):
			}
		}
	}()

	presenter := terminal.Presenter{Out: stdout}
	if err := presenter.Start(); err != nil {
		return err
	}
	defer presenter.Stop()

	controls, stopControls, err := startPlaybackControls(ctx, cancel, stdin)
	if err != nil {
		return err
	}
	defer stopControls()

	playErr := player.PlayWithControls(ctx, frames, presenter, info.FPS, controls)
	ctxErr := ctx.Err()
	if ctxErr != nil || playErr != nil {
		cancel()
	}
	_ = pipe.Close()
	<-renderDone

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
		decoderErr = cleanupDecoder(cmd, pipe, decoderDone)
	} else {
		decoderErr = waitDecoder(pipe, decoderDone)
	}
	decoderCleaned = true
	cancel()
	return playbackResult(ctxErr, playErr, frameErr, decoderErr)
}

func cleanupDecoder(cmd *exec.Cmd, pipe io.Closer, decoderDone <-chan error) error {
	_ = pipe.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return <-decoderDone
}

func waitDecoder(pipe io.Closer, decoderDone <-chan error) error {
	_ = pipe.Close()
	return <-decoderDone
}

func startPlaybackControls(ctx context.Context, cancel context.CancelFunc, stdin *os.File) (<-chan player.Control, func(), error) {
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
	go bridgeTerminalControls(controlCtx, cancel, terminalControls, playerControls)

	cleanup := func() {
		stopControls()
		_ = xterm.Restore(fd, state)
	}
	return playerControls, cleanup, nil
}

func bridgeTerminalControls(ctx context.Context, cancel context.CancelFunc, in <-chan terminal.Control, out chan<- player.Control) {
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
