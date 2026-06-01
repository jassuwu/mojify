package player

import (
	"context"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

type fakePresenter struct {
	count int
}

func (p *fakePresenter) Present(render.CharacterFrame) error {
	p.count++
	return nil
}

func TestPlayerPresentsFrames(t *testing.T) {
	frames := make(chan render.CharacterFrame, 3)
	frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: 'A'}}}
	frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: 'B'}}}
	close(frames)

	presenter := &fakePresenter{}
	err := Play(context.Background(), frames, presenter, 1000)
	if err != nil {
		t.Fatalf("Play returned error: %v", err)
	}
	if presenter.count != 2 {
		t.Fatalf("presented %d frames, want 2", presenter.count)
	}
}

func TestPlayerHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	frames := make(chan render.CharacterFrame)
	cancel()
	start := time.Now()
	err := Play(ctx, frames, &fakePresenter{}, 24)
	if err == nil {
		t.Fatal("Play returned nil error after cancellation")
	}
	if time.Since(start) > time.Second {
		t.Fatal("Play did not return promptly after cancellation")
	}
}
