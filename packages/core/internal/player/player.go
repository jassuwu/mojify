package player

import (
	"context"
	"errors"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter interface {
	Present(render.CharacterFrame) error
}

var ErrCancelled = errors.New("playback cancelled")

func Play(ctx context.Context, frames <-chan render.CharacterFrame, presenter Presenter, fps float64) error {
	if fps <= 0 {
		fps = 24
	}
	frameDuration := time.Duration(float64(time.Second) / fps)
	nextDeadline := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ErrCancelled
		case frame, ok := <-frames:
			if !ok {
				return nil
			}
			now := time.Now()
			if now.Before(nextDeadline) {
				timer := time.NewTimer(nextDeadline.Sub(now))
				select {
				case <-ctx.Done():
					timer.Stop()
					return ErrCancelled
				case <-timer.C:
				}
			}
			if err := presenter.Present(frame); err != nil {
				return err
			}
			nextDeadline = nextDeadline.Add(frameDuration)
		}
	}
}
