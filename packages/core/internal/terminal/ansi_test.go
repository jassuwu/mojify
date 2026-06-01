package terminal

import (
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestSerializeFrameUsesCursorHomeAndTruecolor(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 4, G: 5, B: 6},
		},
	}
	out := SerializeFrame(frame)
	for _, want := range []string{"\x1b[H", "\x1b[38;2;1;2;3mA", "\x1b[38;2;4;5;6mB", "\x1b[0m"} {
		if !strings.Contains(out, want) {
			t.Fatalf("SerializeFrame missing %q in %q", want, out)
		}
	}
}

func TestSerializeFrameUsesDeterministicRowsAndSuppressesRepeatedColor(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 2,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 1, G: 2, B: 3},
			{Ch: 'C', R: 4, G: 5, B: 6},
			{Ch: 'D', R: 4, G: 5, B: 6},
		},
	}
	got := SerializeFrame(frame)
	want := "\x1b[H\x1b[J\x1b[38;2;1;2;3mAB\r\n\x1b[38;2;4;5;6mCD\x1b[0m"
	if got != want {
		t.Fatalf("SerializeFrame() = %q, want %q", got, want)
	}
}
