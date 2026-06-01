package player

import (
	"context"
	"errors"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter interface {
	Present(render.CharacterFrame) error
}

type Control int

const (
	Quit Control = iota
	TogglePause
)

var ErrCancelled = errors.New("playback cancelled")
var ErrQuit = errors.New("playback quit")

func Play(ctx context.Context, frames <-chan render.CharacterFrame, presenter Presenter, fps float64) error {
	return PlayWithControls(ctx, frames, presenter, fps, nil, nil)
}

func PlayWithControls(
	ctx context.Context,
	frames <-chan render.CharacterFrame,
	presenter Presenter,
	fps float64,
	controls <-chan Control,
	metrics *playback.Metrics,
) error {
	return playWithControls(ctx, frames, presenter, fps, controls, realClock{}, metrics)
}

type playbackClock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ErrCancelled
	case <-timer.C:
		return nil
	}
}

func playWithControls(
	ctx context.Context,
	frames <-chan render.CharacterFrame,
	presenter Presenter,
	fps float64,
	controls <-chan Control,
	clock playbackClock,
	metrics *playback.Metrics,
) error {
	if fps <= 0 {
		fps = 24
	}
	frameDuration := time.Duration(float64(time.Second) / fps)
	nextDeadline := clock.Now()
	paused := false

	for {
		if paused {
			control, ok, err := waitForControl(ctx, controls)
			if err != nil {
				return err
			}
			if !ok {
				controls = nil
				paused = false
				nextDeadline = clock.Now()
				continue
			}
			switch control {
			case Quit:
				return ErrQuit
			case TogglePause:
				paused = false
				nextDeadline = clock.Now()
			}
			continue
		}

		if control, ok, polled := pollControl(controls); polled {
			if !ok {
				controls = nil
				continue
			}
			switch control {
			case Quit:
				return ErrQuit
			case TogglePause:
				paused = true
			}
			continue
		}

		select {
		case <-ctx.Done():
			return ErrCancelled
		case control, ok := <-controls:
			if !ok {
				controls = nil
				continue
			}
			switch control {
			case Quit:
				return ErrQuit
			case TogglePause:
				paused = true
			}
		case frame, ok := <-frames:
			if !ok {
				return nil
			}
			now := clock.Now()
			if now.Before(nextDeadline) {
				if err := clock.Sleep(ctx, nextDeadline.Sub(now)); err != nil {
					return err
				}
			} else {
				var skipped int
				frame, nextDeadline, skipped = skipLateBufferedFrames(frames, frame, nextDeadline, frameDuration, now)
				if metrics != nil {
					metrics.RecordSkipped(skipped)
				}
			}
			if err := presenter.Present(frame); err != nil {
				return err
			}
			nextDeadline = nextDeadline.Add(frameDuration)
		}
	}
}

func pollControl(controls <-chan Control) (Control, bool, bool) {
	if controls == nil {
		return 0, true, false
	}
	select {
	case control, ok := <-controls:
		return control, ok, true
	default:
		return 0, true, false
	}
}

func waitForControl(ctx context.Context, controls <-chan Control) (Control, bool, error) {
	select {
	case <-ctx.Done():
		return 0, false, ErrCancelled
	case control, ok := <-controls:
		return control, ok, nil
	}
}

func skipLateBufferedFrames(
	frames <-chan render.CharacterFrame,
	frame render.CharacterFrame,
	nextDeadline time.Time,
	frameDuration time.Duration,
	now time.Time,
) (render.CharacterFrame, time.Time, int) {
	skipped := 0
	for now.Sub(nextDeadline) >= frameDuration {
		select {
		case nextFrame, ok := <-frames:
			if !ok {
				return frame, nextDeadline, skipped
			}
			frame = nextFrame
			nextDeadline = nextDeadline.Add(frameDuration)
			skipped++
		default:
			return frame, nextDeadline, skipped
		}
	}
	return frame, nextDeadline, skipped
}
