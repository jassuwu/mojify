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
	if got, want := out.String(), BeginSynchronizedUpdate+SerializeFrame(frame)+EndSynchronizedUpdate; got != want {
		t.Fatalf("Present wrote %q, want %q", got, want)
	}

	out.Reset()
	if err := presenter.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if got, want := out.String(), EndSynchronizedUpdate+Reset+ShowCursor+ExitAltScreen; got != want {
		t.Fatalf("Stop wrote %q, want %q", got, want)
	}
}

func TestPresenterStartRestoresTerminalOnWriteError(t *testing.T) {
	writer := newFailOnceAfterBytesWriter(len(EnterAltScreen) + 1)
	presenter := Presenter{Out: writer}

	if err := presenter.Start(); err == nil {
		t.Fatal("Start returned nil error")
	}

	if got, want := writer.String(), EnterAltScreen+HideCursor[:1]+EndSynchronizedUpdate+Reset+ShowCursor+ExitAltScreen; got != want {
		t.Fatalf("Start wrote %q, want %q", got, want)
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

func TestWriteSynchronizedFrameReturnsFrameWriteError(t *testing.T) {
	output := "frame output"
	writer := newFailOnceAfterBytesWriter(len(BeginSynchronizedUpdate) + 2)

	n, err := writeSynchronizedFrame(writer, output)

	if err == nil {
		t.Fatal("writeSynchronizedFrame returned nil error")
	}
	if got, want := err.Error(), "write failed"; got != want {
		t.Fatalf("writeSynchronizedFrame error = %q, want %q", got, want)
	}
	if got, want := n, len(BeginSynchronizedUpdate)+2+len(EndSynchronizedUpdate); got != want {
		t.Fatalf("writeSynchronizedFrame wrote %d bytes, want %d", got, want)
	}
	if got, want := writer.String(), BeginSynchronizedUpdate+output[:2]+EndSynchronizedUpdate; got != want {
		t.Fatalf("writeSynchronizedFrame output = %q, want %q", got, want)
	}
}

func TestWriteSynchronizedFrameReturnsEndMarkerWriteError(t *testing.T) {
	output := "frame output"
	writer := newFailAfterBytesWriter(len(BeginSynchronizedUpdate) + len(output) + 2)

	n, err := writeSynchronizedFrame(writer, output)

	if err == nil {
		t.Fatal("writeSynchronizedFrame returned nil error")
	}
	if got, want := err.Error(), "write failed"; got != want {
		t.Fatalf("writeSynchronizedFrame error = %q, want %q", got, want)
	}
	if got, want := n, len(BeginSynchronizedUpdate)+len(output)+2; got != want {
		t.Fatalf("writeSynchronizedFrame wrote %d bytes, want %d", got, want)
	}
	if got, want := writer.String(), BeginSynchronizedUpdate+output+EndSynchronizedUpdate[:2]; got != want {
		t.Fatalf("writeSynchronizedFrame output = %q, want %q", got, want)
	}
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return len(p) / 2, errors.New("write failed")
}

type failAfterBytesWriter struct {
	limit int
	buf   bytes.Buffer
}

func newFailAfterBytesWriter(limit int) *failAfterBytesWriter {
	return &failAfterBytesWriter{limit: limit}
}

func (w *failAfterBytesWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.buf.Len()
	if remaining <= 0 {
		return 0, errors.New("write failed")
	}
	if len(p) > remaining {
		w.buf.Write(p[:remaining])
		return remaining, errors.New("write failed")
	}
	w.buf.Write(p)
	return len(p), nil
}

func (w *failAfterBytesWriter) String() string {
	return w.buf.String()
}

type failOnceAfterBytesWriter struct {
	limit  int
	failed bool
	buf    bytes.Buffer
}

func newFailOnceAfterBytesWriter(limit int) *failOnceAfterBytesWriter {
	return &failOnceAfterBytesWriter{limit: limit}
}

func (w *failOnceAfterBytesWriter) Write(p []byte) (int, error) {
	if w.failed {
		return w.buf.Write(p)
	}

	remaining := w.limit - w.buf.Len()
	if remaining <= 0 {
		w.failed = true
		return 0, errors.New("write failed")
	}
	if len(p) > remaining {
		w.buf.Write(p[:remaining])
		w.failed = true
		return remaining, errors.New("write failed")
	}
	return w.buf.Write(p)
}

func (w *failOnceAfterBytesWriter) String() string {
	return w.buf.String()
}
