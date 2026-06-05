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
			{Ch: 'A', HasColor: true, R: 1, G: 2, B: 3},
			{Ch: 'B', HasColor: true, R: 4, G: 5, B: 6},
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
			{Ch: 'A', HasColor: true, R: 1, G: 2, B: 3},
			{Ch: 'B', HasColor: true, R: 1, G: 2, B: 3},
			{Ch: 'C', HasColor: true, R: 4, G: 5, B: 6},
			{Ch: 'D', HasColor: true, R: 4, G: 5, B: 6},
		},
	}
	got := SerializeFrame(frame)
	want := "\x1b[H\x1b[J\x1b[38;2;1;2;3mAB\r\n\x1b[38;2;4;5;6mCD\x1b[0m"
	if got != want {
		t.Fatalf("SerializeFrame() = %q, want %q", got, want)
	}
}

func TestCursorPositionUsesOneBasedCoordinates(t *testing.T) {
	if got := CursorPosition(2, 3); got != "\x1b[2;3H" {
		t.Fatalf("CursorPosition(2, 3) = %q, want %q", got, "\x1b[2;3H")
	}
}

func TestSerializeFramePatchWritesChangedRuns(t *testing.T) {
	previous := characterFrame(4, 2,
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
	)
	current := characterFrame(4, 2,
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'B', HasColor: true, R: 9, G: 9, B: 9},
		render.Cell{Ch: 'C', HasColor: true, R: 9, G: 9, B: 9},
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', HasColor: true, R: 2, G: 2, B: 2},
		render.Cell{Ch: 'Z', HasColor: true, R: 3, G: 4, B: 5},
	)

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch() unexpected error: %v", err)
	}

	want := CursorPosition(1, 2) + "\x1b[38;2;9;9;9mBC" +
		CursorPosition(2, 4) + "\x1b[38;2;3;4;5mZ" + Reset
	if got != want {
		t.Fatalf("SerializeFramePatch() = %q, want %q", got, want)
	}
	if strings.Contains(got, ClearToEnd) {
		t.Fatalf("SerializeFramePatch() included ClearToEnd in %q", got)
	}
}

func TestSerializeFramePatchReturnsEmptyForIdenticalFrames(t *testing.T) {
	previous := characterFrame(2, 1,
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'B', HasColor: true, R: 2, G: 2, B: 2},
	)
	current := characterFrame(2, 1,
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'B', HasColor: true, R: 2, G: 2, B: 2},
	)

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch() unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("SerializeFramePatch() = %q, want empty string", got)
	}
}

func TestSerializeFramePatchTreatsColorOnlyChangeAsChanged(t *testing.T) {
	previous := characterFrame(1, 1, render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1})
	current := characterFrame(1, 1, render.Cell{Ch: 'A', HasColor: true, R: 2, G: 3, B: 4})

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch() unexpected error: %v", err)
	}

	want := CursorPosition(1, 1) + "\x1b[38;2;2;3;4mA" + Reset
	if got != want {
		t.Fatalf("SerializeFramePatch() = %q, want %q", got, want)
	}
}

func TestSerializeFramePatchResetsColorBeforeLaterNoColorRun(t *testing.T) {
	previous := characterFrame(3, 1,
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'B', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'C', HasColor: true, R: 1, G: 1, B: 1},
	)
	current := characterFrame(3, 1,
		render.Cell{Ch: 'X', HasColor: true, R: 255, G: 0, B: 0},
		render.Cell{Ch: 'B', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'Y', HasColor: false},
	)

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch() unexpected error: %v", err)
	}

	want := CursorPosition(1, 1) + "\x1b[38;2;255;0;0mX" +
		CursorPosition(1, 3) + "\x1b[39mY" + Reset
	if got != want {
		t.Fatalf("SerializeFramePatch() = %q, want %q", got, want)
	}
}

func TestSerializeFramePatchIgnoresStaleRGBForNoColorCells(t *testing.T) {
	previous := characterFrame(1, 1, render.Cell{Ch: 'A', HasColor: false, R: 1, G: 2, B: 3})
	current := characterFrame(1, 1, render.Cell{Ch: 'A', HasColor: false, R: 9, G: 8, B: 7})

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch() unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("SerializeFramePatch() = %q, want empty string", got)
	}
}

func TestSerializeFramePatchRejectsMismatchedDimensions(t *testing.T) {
	previous := characterFrame(1, 1, render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1})
	current := characterFrame(2, 1,
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', HasColor: true, R: 1, G: 1, B: 1},
	)

	_, err := SerializeFramePatch(previous, current)
	if err == nil {
		t.Fatal("SerializeFramePatch() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "frame dimensions differ") {
		t.Fatalf("SerializeFramePatch() error = %q, want frame dimensions differ", err)
	}
}

func TestSerializeFrameSkipsColorForNoColorCells(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: '@', HasColor: false},
			{Ch: '#', HasColor: true, R: 255, G: 0, B: 0},
		},
	}

	got := SerializeFrame(frame)
	if strings.Contains(got, "\x1b[38;2;0;0;0m@") {
		t.Fatalf("SerializeFrame emitted black color for no-color cell: %q", got)
	}
	if !strings.Contains(got, "@\x1b[38;2;255;0;0m#") {
		t.Fatalf("SerializeFrame = %q, want uncolored @ followed by red #", got)
	}
}

func TestSynchronizedUpdateSequencesAreStable(t *testing.T) {
	if BeginSynchronizedUpdate != "\x1b[?2026h" {
		t.Fatalf("BeginSynchronizedUpdate = %q, want CSI ? 2026 h", BeginSynchronizedUpdate)
	}
	if EndSynchronizedUpdate != "\x1b[?2026l" {
		t.Fatalf("EndSynchronizedUpdate = %q, want CSI ? 2026 l", EndSynchronizedUpdate)
	}
}

func characterFrame(width int, height int, cells ...render.Cell) render.CharacterFrame {
	return render.CharacterFrame{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}
