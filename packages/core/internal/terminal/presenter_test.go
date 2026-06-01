package terminal

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)

func TestPresenterLifecycleWritesTerminalSequences(t *testing.T) {
	var out bytes.Buffer
	presenter := Presenter{Out: &out}

	if err := presenter.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if got, want := out.String(), EnterAltScreen+HideCursor+CursorHome+ClearToEnd; got != want {
		t.Fatalf("Start wrote %q, want %q", got, want)
	}

	out.Reset()
	frame := render.CharacterFrame{
		Width:  1,
		Height: 1,
		Cells:  []render.Cell{{Ch: 'A', R: 1, G: 2, B: 3}},
	}
	if err := presenter.Present(frame); err != nil {
		t.Fatalf("Present returned error: %v", err)
	}
	if got, want := out.String(), SerializeFrame(frame); got != want {
		t.Fatalf("Present wrote %q, want %q", got, want)
	}

	out.Reset()
	if err := presenter.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if got, want := out.String(), Reset+ShowCursor+ExitAltScreen; got != want {
		t.Fatalf("Stop wrote %q, want %q", got, want)
	}
}

func TestPresenterRecordsPlaybackMetrics(t *testing.T) {
	metrics := playback.NewMetrics(2, 1)
	var out bytes.Buffer
	presenter := Presenter{Out: &out, Metrics: metrics}
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 255, G: 0, B: 0},
			{Ch: 'B', R: 0, G: 255, B: 0},
		},
	}

	if err := presenter.Present(frame); err != nil {
		t.Fatalf("Present returned error: %v", err)
	}

	snapshot := metrics.Snapshot()
	if snapshot.PresentedFrames != 1 {
		t.Fatalf("PresentedFrames = %d, want 1", snapshot.PresentedFrames)
	}
	if snapshot.AverageBytesPerFrame != out.Len() {
		t.Fatalf("AverageBytesPerFrame = %d, want %d", snapshot.AverageBytesPerFrame, out.Len())
	}
}

func TestPresenterDoesNotRecordPresentedFrameOnWriteError(t *testing.T) {
	metrics := playback.NewMetrics(1, 1)
	presenter := Presenter{Out: failingWriter{}, Metrics: metrics}
	frame := render.CharacterFrame{
		Width:  1,
		Height: 1,
		Cells:  []render.Cell{{Ch: 'A', R: 255, G: 0, B: 0}},
	}

	if err := presenter.Present(frame); err == nil {
		t.Fatal("Present returned nil error")
	}

	snapshot := metrics.Snapshot()
	if snapshot.PresentedFrames != 0 {
		t.Fatalf("PresentedFrames = %d, want 0", snapshot.PresentedFrames)
	}
	if snapshot.AverageBytesPerFrame != 0 {
		t.Fatalf("AverageBytesPerFrame = %d, want 0", snapshot.AverageBytesPerFrame)
	}
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return len(p) / 2, errors.New("write failed")
}
