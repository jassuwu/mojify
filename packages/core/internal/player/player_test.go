package player

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
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

func TestPlayerQuitControlStopsPlayback(t *testing.T) {
	frames := make(chan render.CharacterFrame, 1)
	frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: 'A'}}}

	controls := make(chan Control, 1)
	controls <- Quit

	presenter := &fakePresenter{}
	err := PlayWithControls(context.Background(), frames, presenter, 24, controls, nil)
	if !errors.Is(err, ErrQuit) {
		t.Fatalf("PlayWithControls returned %v, want %v", err, ErrQuit)
	}
	if presenter.count != 0 {
		t.Fatalf("presented %d frames, want 0", presenter.count)
	}
}

func TestPlayerPauseStopsFrameConsumptionUntilResumed(t *testing.T) {
	frames := make(chan render.CharacterFrame)
	controls := make(chan Control, 2)
	controls <- TogglePause

	presenter := &fakePresenter{}
	done := make(chan error, 1)
	go func() {
		done <- PlayWithControls(context.Background(), frames, presenter, 1000, controls, nil)
	}()

	sent := make(chan struct{})
	go func() {
		frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: 'A'}}}
		close(sent)
	}()

	select {
	case <-sent:
		t.Fatal("frame was consumed while playback was paused")
	case <-time.After(20 * time.Millisecond):
	}

	controls <- TogglePause
	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatal("frame was not consumed after resume")
	}
	close(frames)

	if err := <-done; err != nil {
		t.Fatalf("PlayWithControls returned error: %v", err)
	}
	if presenter.count != 1 {
		t.Fatalf("presented %d frames, want 1", presenter.count)
	}
}

func TestPlayerSkipsLateBufferedFrames(t *testing.T) {
	frames := make(chan render.CharacterFrame, 6)
	for _, ch := range []rune{'A', 'B', 'C', 'D', 'E', 'F'} {
		frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: ch}}}
	}
	close(frames)

	clock := &fakeClock{now: time.Unix(0, 0)}
	presenter := &slowPresenter{
		clock: clock,
		delay: 250 * time.Millisecond,
	}

	metrics := playback.NewMetrics(1, 1)
	err := playWithControls(context.Background(), frames, presenter, 10, nil, clock, metrics)
	if err != nil {
		t.Fatalf("playWithControls returned error: %v", err)
	}
	if metrics.Snapshot().SkippedFrames == 0 {
		t.Fatal("SkippedFrames = 0, want skipped frame count")
	}
	if len(presenter.presented) >= 6 {
		t.Fatalf("presented all frames %q, want late frames skipped", string(presenter.presented))
	}
	if len(presenter.presented) < 2 {
		t.Fatalf("presented too few frames %q", string(presenter.presented))
	}
	if presenter.presented[1] == 'B' {
		t.Fatalf("presented second sequential frame %q, want a later buffered frame", string(presenter.presented))
	}
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Sleep(context.Context, time.Duration) error {
	return nil
}

type slowPresenter struct {
	clock     *fakeClock
	delay     time.Duration
	presented []rune
}

func (p *slowPresenter) Present(frame render.CharacterFrame) error {
	p.presented = append(p.presented, frame.Cells[0].Ch)
	p.clock.now = p.clock.now.Add(p.delay)
	return nil
}
