package exporter

import "testing"

func TestResolveLayoutDefaultsWidthFPSAndAlignsToCells(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 1920, Height: 1080, FPS: 30}, Options{})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}

	if layout.OutputWidth != 1280 {
		t.Fatalf("OutputWidth = %d, want 1280", layout.OutputWidth)
	}
	if layout.OutputHeight != 720 {
		t.Fatalf("OutputHeight = %d, want 720", layout.OutputHeight)
	}
	if layout.Grid.Cols != 160 || layout.Grid.Rows != 90 {
		t.Fatalf("Grid = %dx%d, want 160x90", layout.Grid.Cols, layout.Grid.Rows)
	}
	if layout.FPS != 30 {
		t.Fatalf("FPS = %v, want 30", layout.FPS)
	}
}

func TestResolveLayoutUsesRequestedWidthAndFPSOverride(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 640, Height: 480}, Options{Width: 321, FPS: 12.5})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}

	if layout.OutputWidth != 328 {
		t.Fatalf("OutputWidth = %d, want 328", layout.OutputWidth)
	}
	if layout.OutputHeight != 248 {
		t.Fatalf("OutputHeight = %d, want 248", layout.OutputHeight)
	}
	if layout.Grid.Cols != 41 || layout.Grid.Rows != 31 {
		t.Fatalf("Grid = %dx%d, want 41x31", layout.Grid.Cols, layout.Grid.Rows)
	}
	if layout.FPS != 12.5 {
		t.Fatalf("FPS = %v, want 12.5", layout.FPS)
	}
}

func TestResolveLayoutDefaultsFPSWhenSourceFPSMissing(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 320, Height: 240}, Options{})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}

	if layout.FPS != 24 {
		t.Fatalf("FPS = %v, want 24", layout.FPS)
	}
}

func TestResolveLayoutRejectsInvalidInputDimensions(t *testing.T) {
	if _, err := ResolveLayout(InputInfo{Width: 0, Height: 240}, Options{}); err == nil {
		t.Fatal("ResolveLayout returned nil error for zero input width")
	}
	if _, err := ResolveLayout(InputInfo{Width: 320, Height: -1}, Options{}); err == nil {
		t.Fatal("ResolveLayout returned nil error for negative input height")
	}
}

func TestResolveLayoutRejectsInvalidOverrides(t *testing.T) {
	if _, err := ResolveLayout(InputInfo{Width: 320, Height: 240}, Options{Width: -1}); err == nil {
		t.Fatal("ResolveLayout returned nil error for negative requested width")
	}
	if _, err := ResolveLayout(InputInfo{Width: 320, Height: 240}, Options{FPS: -1}); err == nil {
		t.Fatal("ResolveLayout returned nil error for negative requested FPS")
	}
}

func TestResolveLayoutUsesWidthAsColumnsForTextOutput(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 1920, Height: 1080, FPS: 24}, Options{
		Width: 80,
		Format: OutputFormat{
			Extension:   ".txt",
			Family:      OutputFamilyText,
			Text:        true,
			SingleFrame: true,
		},
	})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}
	if layout.Grid.Cols != 80 {
		t.Fatalf("Grid.Cols = %d, want 80", layout.Grid.Cols)
	}
	if layout.OutputWidth != 80*ExportCellWidth {
		t.Fatalf("OutputWidth = %d, want %d", layout.OutputWidth, 80*ExportCellWidth)
	}
}
