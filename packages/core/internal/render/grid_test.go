package render

import "testing"

func TestFitGridUsesTerminalWidthForWideVideo(t *testing.T) {
	grid := FitGrid(InputSize{Width: 1920, Height: 1080}, TerminalSize{Cols: 120, Rows: 40})
	if grid.Cols != 120 {
		t.Fatalf("Cols = %d, want 120", grid.Cols)
	}
	if grid.Rows != 33 {
		t.Fatalf("Rows = %d, want 33", grid.Rows)
	}
}

func TestFitGridUsesTerminalHeightWhenNeeded(t *testing.T) {
	grid := FitGrid(InputSize{Width: 1080, Height: 1920}, TerminalSize{Cols: 120, Rows: 40})
	if grid.Rows != 40 {
		t.Fatalf("Rows = %d, want 40", grid.Rows)
	}
	if grid.Cols != 45 {
		t.Fatalf("Cols = %d, want 45", grid.Cols)
	}
}

func TestFitGridKeepsMinimums(t *testing.T) {
	grid := FitGrid(InputSize{Width: 1920, Height: 1080}, TerminalSize{Cols: 5, Rows: 2})
	if grid.Cols != 10 || grid.Rows != 5 {
		t.Fatalf("grid = %dx%d, want 10x5", grid.Cols, grid.Rows)
	}
}

func TestFitGridClampsUltraWideVideoToMinimumRows(t *testing.T) {
	grid := FitGrid(InputSize{Width: 1200, Height: 50}, TerminalSize{Cols: 120, Rows: 40})
	if grid.Cols != 120 || grid.Rows != 5 {
		t.Fatalf("grid = %dx%d, want 120x5", grid.Cols, grid.Rows)
	}
}
