package render

import (
	"reflect"
	"strings"
	"testing"
)

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

func TestRecipePresetNames(t *testing.T) {
	names := RecipePresetNames()
	want := []string{"default", "mono", "ascii", "blocks"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("RecipePresetNames() = %#v, want %#v", names, want)
	}
}

func TestRecipeByNameRejectsUnknownPreset(t *testing.T) {
	_, err := RecipeByName("banana")
	if err == nil {
		t.Fatal("RecipeByName returned nil error for unknown preset")
	}
	if !strings.Contains(err.Error(), `unsupported recipe "banana"`) {
		t.Fatalf("error = %v, want unsupported recipe name", err)
	}
	if !strings.Contains(err.Error(), "default, mono, ascii, blocks") {
		t.Fatalf("error = %v, want supported recipe list", err)
	}
}

func TestDefaultRecipeMatchesExistingRendererBehavior(t *testing.T) {
	frame := NewRGBFrame(3, 3, []byte{
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
	})
	out := NewRenderer(MustRecipeByName("default")).Render(frame, Grid{Cols: 3, Rows: 3})
	center := out.Cells[4]
	if center.Ch != '|' {
		t.Fatalf("center char = %q, want edge glyph |", center.Ch)
	}
	if !center.HasColor || center.R != 255 || center.G != 255 || center.B != 255 {
		t.Fatalf("center color = (%v,%d,%d,%d), want source white", center.HasColor, center.R, center.G, center.B)
	}
}

func TestMonoRecipeKeepsEdgesAndDisablesColor(t *testing.T) {
	frame := NewRGBFrame(3, 3, []byte{
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
	})
	out := NewRenderer(MustRecipeByName("mono")).Render(frame, Grid{Cols: 3, Rows: 3})
	center := out.Cells[4]
	if center.Ch != '|' {
		t.Fatalf("center char = %q, want edge glyph |", center.Ch)
	}
	if center.HasColor {
		t.Fatalf("center HasColor = true, want false")
	}
}

func TestASCIIRecipeUsesClassicRampWithoutEdgesOrColor(t *testing.T) {
	frame := NewRGBFrame(3, 3, []byte{
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
	})
	out := NewRenderer(MustRecipeByName("ascii")).Render(frame, Grid{Cols: 3, Rows: 3})
	center := out.Cells[4]
	if center.Ch != '@' {
		t.Fatalf("center char = %q, want classic ramp max @", center.Ch)
	}
	if center.HasColor {
		t.Fatalf("center HasColor = true, want false")
	}
}

func TestBlocksRecipeUsesShadeRampWithColorAndNoEdges(t *testing.T) {
	frame := NewRGBFrame(3, 3, []byte{
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
		0, 0, 0, 255, 255, 255, 255, 255, 255,
	})
	out := NewRenderer(MustRecipeByName("blocks")).Render(frame, Grid{Cols: 3, Rows: 3})
	center := out.Cells[4]
	if center.Ch != '█' {
		t.Fatalf("center char = %q, want block ramp max █", center.Ch)
	}
	if !center.HasColor || center.R != 255 || center.G != 255 || center.B != 255 {
		t.Fatalf("center color = (%v,%d,%d,%d), want source white", center.HasColor, center.R, center.G, center.B)
	}
}
