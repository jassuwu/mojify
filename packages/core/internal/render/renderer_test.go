package render

import "testing"

func TestRendererMapsLuminanceToDensity(t *testing.T) {
	frame := NewRGBFrame(4, 1, []byte{
		0, 0, 0,
		85, 85, 85,
		170, 170, 170,
		255, 255, 255,
	})
	out := DefaultRenderer{}.Render(frame, Grid{Cols: 4, Rows: 1})
	got := string([]rune{out.Cells[0].Ch, out.Cells[1].Ch, out.Cells[2].Ch, out.Cells[3].Ch})
	want := " .#@"
	if got != want {
		t.Fatalf("chars = %q, want %q", got, want)
	}
}

func TestRendererPreservesColor(t *testing.T) {
	frame := NewRGBFrame(1, 1, []byte{10, 20, 30})
	out := DefaultRenderer{}.Render(frame, Grid{Cols: 1, Rows: 1})
	cell := out.Cells[0]
	if cell.R != 10 || cell.G != 20 || cell.B != 30 {
		t.Fatalf("color = (%d,%d,%d), want (10,20,30)", cell.R, cell.G, cell.B)
	}
}

func TestRendererOverridesStrongVerticalEdge(t *testing.T) {
	frame := NewRGBFrame(3, 3, []byte{
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
	})
	out := DefaultRenderer{}.Render(frame, Grid{Cols: 3, Rows: 3})
	center := out.Cells[4].Ch
	if center != '|' {
		t.Fatalf("center edge char = %q, want |", center)
	}
}
