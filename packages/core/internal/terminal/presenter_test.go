package terminal

import (
	"bytes"
	"testing"

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
