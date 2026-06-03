package exporter

import (
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestSerializeTextFrameWritesPlainRows(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 2,
		Cells: []render.Cell{
			{Ch: 'A'},
			{},
			{Ch: 'B'},
			{Ch: '!'},
		},
	}

	got, err := SerializeTextFrame(frame, OutputFormat{Extension: ".txt"})
	if err != nil {
		t.Fatalf("SerializeTextFrame returned error: %v", err)
	}

	want := "A \nB!\n"
	if got != want {
		t.Fatalf("SerializeTextFrame() = %q, want %q", got, want)
	}
}

func TestSerializeTextFrameWritesANSIForegroundEscapesAndFinalReset(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'R', R: 255, G: 0, B: 0},
			{Ch: 0, R: 0, G: 128, B: 255},
		},
	}

	got, err := SerializeTextFrame(frame, OutputFormat{Extension: ".ansi"})
	if err != nil {
		t.Fatalf("SerializeTextFrame returned error: %v", err)
	}

	want := "\x1b[38;2;255;0;0mR\x1b[38;2;0;128;255m \x1b[0m\n"
	if got != want {
		t.Fatalf("SerializeTextFrame() = %q, want %q", got, want)
	}
	if !strings.HasSuffix(got, "\x1b[0m\n") {
		t.Fatalf("SerializeTextFrame() does not end with reset/newline: %q", got)
	}
}

func TestSerializeTextFrameRejectsInvalidFrames(t *testing.T) {
	tests := []struct {
		name  string
		frame render.CharacterFrame
	}{
		{
			name: "zero width",
			frame: render.CharacterFrame{
				Width:  0,
				Height: 1,
				Cells:  []render.Cell{{Ch: 'A'}},
			},
		},
		{
			name: "zero height",
			frame: render.CharacterFrame{
				Width:  1,
				Height: 0,
				Cells:  []render.Cell{{Ch: 'A'}},
			},
		},
		{
			name: "wrong cell count",
			frame: render.CharacterFrame{
				Width:  2,
				Height: 1,
				Cells:  []render.Cell{{Ch: 'A'}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := SerializeTextFrame(tc.frame, OutputFormat{Extension: ".txt"}); err == nil {
				t.Fatal("SerializeTextFrame returned nil error")
			}
		})
	}
}
