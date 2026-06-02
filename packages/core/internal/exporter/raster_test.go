package exporter

import (
	"testing"

	"github.com/jass/mojify/packages/core/internal/render"
	"golang.org/x/image/font/basicfont"
)

func TestRasterizerDrawsCellForegroundIntoRGBBuffer(t *testing.T) {
	layout := Layout{
		OutputWidth:  ExportCellWidth * 2,
		OutputHeight: ExportCellHeight,
		Grid:         render.Grid{Cols: 2, Rows: 1},
		FPS:          24,
	}
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: '@', R: 255, G: 0, B: 0},
			{Ch: '@', R: 0, G: 255, B: 0},
		},
	}

	rgb, err := NewRasterizer(basicfont.Face7x13).Rasterize(frame, layout)
	if err != nil {
		t.Fatalf("Rasterize returned error: %v", err)
	}

	wantLen := layout.OutputWidth * layout.OutputHeight * 3
	if len(rgb) != wantLen {
		t.Fatalf("len(rgb) = %d, want %d", len(rgb), wantLen)
	}
	if !cellContainsRGB(rgb, layout.OutputWidth, 0, 0, 255, 0, 0) {
		t.Fatal("first cell does not contain red glyph pixels")
	}
	if !cellContainsRGB(rgb, layout.OutputWidth, 1, 0, 0, 255, 0) {
		t.Fatal("second cell does not contain green glyph pixels")
	}
	if r, g, b := pixelAt(rgb, layout.OutputWidth, 7, 0); r != defaultBackground.R || g != defaultBackground.G || b != defaultBackground.B {
		t.Fatalf("background pixel = rgb(%d,%d,%d), want rgb(%d,%d,%d)", r, g, b, defaultBackground.R, defaultBackground.G, defaultBackground.B)
	}
}

func TestRasterizerRejectsFrameThatDoesNotMatchLayoutGrid(t *testing.T) {
	layout := Layout{
		OutputWidth:  ExportCellWidth,
		OutputHeight: ExportCellHeight,
		Grid:         render.Grid{Cols: 1, Rows: 1},
		FPS:          24,
	}
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells:  []render.Cell{{Ch: '@'}, {Ch: '@'}},
	}

	if _, err := NewRasterizer(basicfont.Face7x13).Rasterize(frame, layout); err == nil {
		t.Fatal("Rasterize returned nil error for mismatched frame dimensions")
	}
}

func TestRasterizerRejectsInvalidFrameCellCount(t *testing.T) {
	layout := Layout{
		OutputWidth:  ExportCellWidth * 2,
		OutputHeight: ExportCellHeight,
		Grid:         render.Grid{Cols: 2, Rows: 1},
		FPS:          24,
	}
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells:  []render.Cell{{Ch: '@'}},
	}

	if _, err := NewRasterizer(basicfont.Face7x13).Rasterize(frame, layout); err == nil {
		t.Fatal("Rasterize returned nil error for invalid frame cell count")
	}
}

func TestRasterizerRejectsInvalidCellSize(t *testing.T) {
	rasterizer := NewRasterizer(basicfont.Face7x13)
	rasterizer.CellWidth = 0

	layout := Layout{
		OutputWidth:  ExportCellWidth,
		OutputHeight: ExportCellHeight,
		Grid:         render.Grid{Cols: 1, Rows: 1},
		FPS:          24,
	}
	frame := render.CharacterFrame{
		Width:  1,
		Height: 1,
		Cells:  []render.Cell{{Ch: '@'}},
	}

	if _, err := rasterizer.Rasterize(frame, layout); err == nil {
		t.Fatal("Rasterize returned nil error for invalid cell size")
	}
}

func cellContainsRGB(rgb []byte, imageWidth int, cellX int, cellY int, r uint8, g uint8, b uint8) bool {
	startX := cellX * ExportCellWidth
	startY := cellY * ExportCellHeight
	for y := startY; y < startY+ExportCellHeight; y++ {
		for x := startX; x < startX+ExportCellWidth; x++ {
			pr, pg, pb := pixelAt(rgb, imageWidth, x, y)
			if pr == r && pg == g && pb == b {
				return true
			}
		}
	}
	return false
}

func pixelAt(rgb []byte, imageWidth int, x int, y int) (uint8, uint8, uint8) {
	offset := (y*imageWidth + x) * 3
	return rgb[offset], rgb[offset+1], rgb[offset+2]
}
