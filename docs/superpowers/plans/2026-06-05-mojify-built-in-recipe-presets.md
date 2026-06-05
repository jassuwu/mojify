# Built-In Recipe Presets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add built-in renderer recipe presets selected by `--recipe` for `mojify play` and `mojify export`.

**Architecture:** Introduce a small recipe-definition model in the render package and route selected recipes from CLI parsing into playback and exporter rendering paths. Keep presets as named built-ins only, but model them as character ramp, color mode, and edge mode so a future `--recipe-file` can hydrate the same type. Add explicit cell color presence so no-color recipes affect terminal, text, ANSI, and raster exports consistently.

**Tech Stack:** Go, existing Mojify render/player/exporter packages, FFmpeg-backed export QA, Bun/Turbo scripts, shell QA scripts.

---

## Locked Product Contract

- `mojify play` accepts `--recipe default|mono|ascii|blocks`.
- `mojify export` accepts `--recipe default|mono|ascii|blocks`.
- `mojify probe` does not accept `--recipe`.
- No `--recipe` preserves today's default rendering.
- `--recipe default` is a valid explicit baseline.
- Invalid recipe names fail during CLI parsing before source resolution.
- Recipes do not affect grid sizing, FPS, export dimensions, source resolution, URL resolution, or audio.
- Presets:
  - `default`: existing Mojify ramp ` .;coPO?#@`, source color, default edge override.
  - `mono`: existing Mojify ramp ` .;coPO?#@`, no source/ANSI color, default edge override.
  - `ascii`: classic ramp ` .:-=+*#%@`, no source/ANSI color, no edge override.
  - `blocks`: Unicode shade/block ramp ` ░▒▓█`, source color, no edge override.
- No-color raster media/image exports render white glyphs on the existing black background.
- `.txt` and `.ansi` exports reflect the selected recipe; no-color recipes do not emit ANSI foreground color.
- `--recipe-file`, custom recipe files, emoji, Braille/subpixel rendering, composable style flags, and recipe-specific layout changes are out of scope.

## File Structure

- Modify: `CONTEXT.md`
  - Already updated during grilling with recipe preset vocabulary.
- Create: `docs/adr/0029-ship-built-in-recipe-presets-before-custom-recipes.md`
  - Records the built-in-presets-before-custom-recipes decision.
- Create: `docs/superpowers/specs/2026-06-05-mojify-built-in-recipe-presets-design.md`
  - Captures the approved design.
- Create: `docs/recipes.md`
  - User-facing recipe preset documentation and future custom recipe intent.
- Modify: `README.md`
  - Add one recipe example and a capability bullet.
- Modify: `docs/qa/export.md`
  - Add preset matrix expectations.
- Modify: `packages/core/internal/render/frame.go`
  - Add explicit color presence to `Cell`.
- Modify: `packages/core/internal/render/renderer.go`
  - Add recipe definition, preset lookup, ramp/color/edge modes, and recipe-aware rendering.
- Modify: `packages/core/internal/render/renderer_test.go`
  - Add golden tests for each preset and `HasColor`.
- Modify: `packages/core/internal/terminal/ansi.go`
  - Skip ANSI color changes for cells without color.
- Modify: `packages/core/internal/terminal/ansi_test.go`
  - Cover mixed color/no-color serialization.
- Modify: `packages/core/internal/exporter/text.go`
  - Skip ANSI foreground color for cells without color.
- Modify: `packages/core/internal/exporter/text_test.go`
  - Cover ANSI serialization with no-color cells.
- Modify: `packages/core/internal/exporter/raster.go`
  - Render no-color cells as white glyphs.
- Modify: `packages/core/internal/exporter/raster_test.go`
  - Cover white glyph pixels for no-color cells.
- Modify: `packages/core/internal/exporter/layout.go`
  - Add `Recipe render.Recipe` to exporter options.
- Modify: `packages/core/internal/exporter/export.go`
  - Use selected recipe for video, image, and text exports.
- Modify: `packages/core/internal/exporter/pipeline.go`
  - Pass recipe into export frame processors.
- Modify: `packages/core/internal/exporter/pipeline_test.go`
  - Cover recipe handoff through the parallel frame pipeline.
- Modify: `packages/core/internal/exporter/export_test.go`
  - Cover recipe handoff for single-frame text/image helper paths.
- Modify: `packages/core/internal/cli/cli.go`
  - Parse `--recipe` for `play` and `export`; reject it for `probe`; update help.
- Modify: `packages/core/internal/cli/cli_test.go`
  - Cover parser behavior and bad recipe errors.
- Modify: `packages/core/internal/cli/play.go`
  - Resolve and pass selected recipe to playback rendering.
- Modify: `packages/core/internal/cli/play_test.go`
  - Cover play options carrying recipe selection.
- Modify: `packages/core/internal/cli/export.go`
  - Resolve and pass selected recipe to exporter options.
- Modify: `packages/core/internal/cli/export_test.go`
  - Cover export handoff carrying recipe selection.
- Modify: `scripts/export-qa.sh`
  - Export still-source recipe matrix and verify outputs.

---

### Task 1: Add Recipe Definitions and Preset Rendering

**Files:**
- Modify: `packages/core/internal/render/frame.go`
- Modify: `packages/core/internal/render/renderer.go`
- Modify: `packages/core/internal/render/renderer_test.go`

- [ ] **Step 1: Add failing preset lookup and renderer tests**

Append these tests to `packages/core/internal/render/renderer_test.go`:

```go
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
```

Add imports if missing:

```go
import (
	"reflect"
	"strings"
	"testing"
)
```

- [ ] **Step 2: Run renderer tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/render
```

Expected: fail to compile because `RecipePresetNames`, `RecipeByName`, `MustRecipeByName`, and `NewRenderer` do not exist, and `Cell.HasColor` does not exist.

- [ ] **Step 3: Add color presence to rendered cells**

In `packages/core/internal/render/frame.go`, change `Cell` to:

```go
type Cell struct {
	Ch       rune
	HasColor bool
	R, G, B  uint8
}
```

- [ ] **Step 4: Add recipe types and preset lookup**

In `packages/core/internal/render/renderer.go`, replace the single hard-coded renderer shape with these definitions near the top:

```go
const (
	defaultDensityRamp = " .;coPO?#@"
	asciiDensityRamp   = " .:-=+*#%@"
	blocksDensityRamp  = " ░▒▓█"
)

type RampMode string

const (
	RampModeDefault RampMode = "default"
	RampModeASCII   RampMode = "ascii"
	RampModeBlocks  RampMode = "blocks"
)

type ColorMode string

const (
	ColorModeSource ColorMode = "source"
	ColorModeNone   ColorMode = "none"
)

type EdgeMode string

const (
	EdgeModeDefault EdgeMode = "default"
	EdgeModeNone    EdgeMode = "none"
)

type Recipe struct {
	Name      string
	RampMode  RampMode
	ColorMode ColorMode
	EdgeMode  EdgeMode
}

var recipePresets = []Recipe{
	{Name: "default", RampMode: RampModeDefault, ColorMode: ColorModeSource, EdgeMode: EdgeModeDefault},
	{Name: "mono", RampMode: RampModeDefault, ColorMode: ColorModeNone, EdgeMode: EdgeModeDefault},
	{Name: "ascii", RampMode: RampModeASCII, ColorMode: ColorModeNone, EdgeMode: EdgeModeNone},
	{Name: "blocks", RampMode: RampModeBlocks, ColorMode: ColorModeSource, EdgeMode: EdgeModeNone},
}
```

Add lookup helpers:

```go
func DefaultRecipe() Recipe {
	return recipePresets[0]
}

func RecipePresetNames() []string {
	names := make([]string, 0, len(recipePresets))
	for _, recipe := range recipePresets {
		names = append(names, recipe.Name)
	}
	return names
}

func RecipeByName(name string) (Recipe, error) {
	for _, recipe := range recipePresets {
		if recipe.Name == name {
			return recipe, nil
		}
	}
	return Recipe{}, fmt.Errorf("unsupported recipe %q; supported recipes: %s", name, strings.Join(RecipePresetNames(), ", "))
}

func MustRecipeByName(name string) Recipe {
	recipe, err := RecipeByName(name)
	if err != nil {
		panic(err)
	}
	return recipe
}
```

Add `fmt` and `strings` to the imports:

```go
import (
	"fmt"
	"math"
	"strings"
)
```

- [ ] **Step 5: Make the renderer recipe-aware**

In `packages/core/internal/render/renderer.go`, replace `DefaultRenderer` with a recipe-aware renderer while keeping `DefaultRenderer` as compatibility wrapper:

```go
type Renderer struct {
	Recipe Recipe
}

type DefaultRenderer struct{}

func NewRenderer(recipe Recipe) Renderer {
	if recipe.Name == "" {
		recipe = DefaultRecipe()
	}
	return Renderer{Recipe: recipe}
}

func (DefaultRenderer) Render(frame RGBFrame, grid Grid) CharacterFrame {
	return NewRenderer(DefaultRecipe()).Render(frame, grid)
}

func (r Renderer) Render(frame RGBFrame, grid Grid) CharacterFrame {
	cells := make([]Cell, grid.Cols*grid.Rows)
	for gy := 0; gy < grid.Rows; gy++ {
		for gx := 0; gx < grid.Cols; gx++ {
			sx := gx * frame.Width / grid.Cols
			sy := gy * frame.Height / grid.Rows
			red, green, blue := frame.RGBAt(sx, sy)
			luma := luminance(red, green, blue)
			ch := densityCharForRamp(luma, rampForMode(r.Recipe.RampMode))
			if r.Recipe.EdgeMode == EdgeModeDefault {
				if edge, ok := edgeGlyph(frame, sx, sy); ok {
					ch = edge
				}
			}
			cell := Cell{Ch: ch}
			if r.Recipe.ColorMode == ColorModeSource {
				cell.HasColor = true
				cell.R = red
				cell.G = green
				cell.B = blue
			}
			cells[gy*grid.Cols+gx] = cell
		}
	}
	return CharacterFrame{Width: grid.Cols, Height: grid.Rows, Cells: cells}
}
```

Replace `densityChar` with:

```go
func densityChar(luma float64) rune {
	return densityCharForRamp(luma, defaultDensityRamp)
}

func densityCharForRamp(luma float64, ramp string) rune {
	normalized := luma / 255.0
	if normalized < 0.5 {
		normalized = 0.5 * math.Pow(normalized*2, 4)
	} else {
		normalized = 1 - 0.5*math.Pow((1-normalized)*2, 4)
	}
	runes := []rune(ramp)
	index := int(math.Round(normalized * float64(len(runes)-1)))
	if index < 0 {
		index = 0
	}
	if index >= len(runes) {
		index = len(runes) - 1
	}
	return runes[index]
}

func rampForMode(mode RampMode) string {
	switch mode {
	case RampModeASCII:
		return asciiDensityRamp
	case RampModeBlocks:
		return blocksDensityRamp
	default:
		return defaultDensityRamp
	}
}
```

- [ ] **Step 6: Run renderer tests and verify they pass**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/render
```

Expected: pass.

- [ ] **Step 7: Commit renderer recipe model**

Commit:

```bash
git add packages/core/internal/render/frame.go packages/core/internal/render/renderer.go packages/core/internal/render/renderer_test.go
git commit --no-gpg-sign -m "feat: add renderer recipe presets"
```

---

### Task 2: Honor No-Color Cells in Terminal, Text, and Raster Output

**Files:**
- Modify: `packages/core/internal/terminal/ansi.go`
- Modify: `packages/core/internal/terminal/ansi_test.go`
- Modify: `packages/core/internal/exporter/text.go`
- Modify: `packages/core/internal/exporter/text_test.go`
- Modify: `packages/core/internal/exporter/raster.go`
- Modify: `packages/core/internal/exporter/raster_test.go`

- [ ] **Step 1: Add failing terminal ANSI serialization test**

Append to `packages/core/internal/terminal/ansi_test.go`:

```go
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
```

- [ ] **Step 2: Add failing text ANSI serialization test**

Append to `packages/core/internal/exporter/text_test.go`:

```go
func TestSerializeANSITextFrameSkipsNoColorCells(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: '@', HasColor: false},
			{Ch: '#', HasColor: true, R: 0, G: 255, B: 0},
		},
	}

	got := serializeANSITextFrame(frame)
	if strings.Contains(got, "\x1b[38;2;0;0;0m@") {
		t.Fatalf("serializeANSITextFrame emitted black color for no-color cell: %q", got)
	}
	if !strings.Contains(got, "@\x1b[38;2;0;255;0m#") {
		t.Fatalf("serializeANSITextFrame = %q, want uncolored @ followed by green #", got)
	}
}
```

- [ ] **Step 3: Add failing rasterizer no-color foreground test**

Append to `packages/core/internal/exporter/raster_test.go`:

```go
func TestRasterizerDrawsNoColorCellAsWhite(t *testing.T) {
	rasterizer := NewRasterizer(basicfont.Face7x13)
	frame := render.CharacterFrame{
		Width:  1,
		Height: 1,
		Cells: []render.Cell{{Ch: '@', HasColor: false}},
	}
	layout := Layout{
		OutputWidth:  ExportCellWidth,
		OutputHeight: ExportCellHeight,
		Grid:         render.Grid{Cols: 1, Rows: 1},
		FPS:          24,
	}

	raw, err := rasterizer.Rasterize(frame, layout)
	if err != nil {
		t.Fatalf("Rasterize returned error: %v", err)
	}
	if !cellContainsRGB(raw, layout.OutputWidth, 0, 0, 255, 255, 255) {
		t.Fatal("no-color cell does not contain white glyph pixels")
	}
}
```

If `rawHasBrightPixel` conflicts with an existing helper name, choose `rawContainsBrightPixel`.

- [ ] **Step 4: Run targeted tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/terminal ./packages/core/internal/exporter
```

Expected: fail because serializers/rasterizer ignore `HasColor`.

- [ ] **Step 5: Update existing colored cell test fixtures**

In test files that manually construct colored `render.Cell` values, add `HasColor: true` to every cell where `R`, `G`, or `B` is intended to affect output.

Use this search to find the fixtures:

```bash
rg -n 'render\.Cell\{|Cell\{[^\n]*(R:|G:|B:)' packages/core/internal -g '*_test.go'
```

At minimum update these files:

- `packages/core/internal/terminal/ansi_test.go`
- `packages/core/internal/terminal/presenter_test.go`
- `packages/core/internal/exporter/raster_test.go`
- `packages/core/internal/exporter/text_test.go`

Example replacement:

```go
render.Cell{Ch: 'A', R: 1, G: 2, B: 3}
```

becomes:

```go
render.Cell{Ch: 'A', HasColor: true, R: 1, G: 2, B: 3}
```

- [ ] **Step 6: Update terminal ANSI serialization**

In `packages/core/internal/terminal/ansi.go`, update both full-frame and patch serialization loops so color is emitted only when `cell.HasColor` is true. The full-frame run writer should follow this shape:

```go
func writeColoredRun(b *strings.Builder, cells []render.Cell) {
	hasColor := false
	var lastR, lastG, lastB uint8
	for _, cell := range cells {
		if cell.HasColor {
			if !hasColor || cell.R != lastR || cell.G != lastG || cell.B != lastB {
				fmt.Fprintf(b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
				hasColor = true
				lastR, lastG, lastB = cell.R, cell.G, cell.B
			}
		} else if hasColor {
			b.WriteString("\x1b[39m")
			hasColor = false
		}
		b.WriteRune(cell.Ch)
	}
	if hasColor {
		b.WriteString("\x1b[39m")
	}
}
```

Apply the same `HasColor` logic in any row/patch loop that currently compares only RGB fields.

- [ ] **Step 7: Update text ANSI serialization**

In `packages/core/internal/exporter/text.go`, update `serializeANSITextFrame` so it emits `\x1b[38;2...m` only for `cell.HasColor` and resets foreground with `\x1b[39m` when leaving a colored run.

- [ ] **Step 8: Update rasterizer no-color foreground**

In `packages/core/internal/exporter/raster.go`, choose a draw color per cell:

```go
glyphColor := color.RGBA{255, 255, 255, 255}
if cell.HasColor {
	glyphColor = color.RGBA{cell.R, cell.G, cell.B, 255}
}
```

Use `glyphColor` for the glyph draw operation. Keep the existing black background.

- [ ] **Step 9: Run targeted tests and verify they pass**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/terminal ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 10: Commit serializer and rasterizer color-presence support**

Commit:

```bash
git add packages/core/internal/terminal/ansi.go packages/core/internal/terminal/ansi_test.go packages/core/internal/terminal/presenter_test.go packages/core/internal/exporter/text.go packages/core/internal/exporter/text_test.go packages/core/internal/exporter/raster.go packages/core/internal/exporter/raster_test.go
git commit --no-gpg-sign -m "feat: support no-color rendered cells"
```

---

### Task 3: Route Recipes Through Exporter Rendering Paths

**Files:**
- Modify: `packages/core/internal/exporter/layout.go`
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/exporter/pipeline.go`
- Modify: `packages/core/internal/exporter/pipeline_test.go`
- Modify: `packages/core/internal/exporter/export_test.go`

- [ ] **Step 1: Add failing exporter recipe handoff tests**

In `packages/core/internal/exporter/export_test.go`, add:

```go
func TestExportSingleTextFrameForTestUsesSelectedRecipe(t *testing.T) {
	output := filepath.Join(t.TempDir(), "out.txt")
	frame := render.NewRGBFrame(1, 1, []byte{255, 255, 255})

	err := exportSingleTextFrameForTest(frame, output, Options{
		Format: OutputFormat{Extension: ".txt", Family: OutputFamilyText, Text: true, SingleFrame: true},
		Recipe: render.MustRecipeByName("blocks"),
	})
	if err != nil {
		t.Fatalf("exportSingleTextFrameForTest returned error: %v", err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), "█") {
		t.Fatalf("text output = %q, want block glyph", string(content))
	}
}
```

Ensure imports include `os`, `path/filepath`, `strings`, and render if not already present.

- [ ] **Step 2: Add failing pipeline processor recipe test**

In `packages/core/internal/exporter/pipeline_test.go`, add:

```go
func TestExportFrameProcessorFactoryUsesSelectedRecipe(t *testing.T) {
	layout := Layout{
		OutputWidth:  ExportCellWidth,
		OutputHeight: ExportCellHeight,
		Grid:         render.Grid{Cols: 1, Rows: 1},
		FPS:          24,
	}
	processor, err := newExportFrameProcessorFactory(layout, nil, fakeExportClock(time.Unix(0, 0)), render.MustRecipeByName("blocks"))()
	if err != nil {
		t.Fatalf("processor factory returned error: %v", err)
	}
	raw, err := processor(0, render.NewRGBFrame(1, 1, []byte{255, 255, 255}))
	if err != nil {
		t.Fatalf("processor returned error: %v", err)
	}
	if !cellContainsRGB(raw, layout.OutputWidth, 0, 0, 255, 255, 255) {
		t.Fatal("processor did not rasterize selected recipe output as white block pixels")
	}
}
```

- [ ] **Step 3: Run exporter tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/exporter
```

Expected: fail to compile because `Options.Recipe` and the new processor factory signature do not exist.

- [ ] **Step 4: Add recipe to exporter options**

In `packages/core/internal/exporter/layout.go`, the render package is already imported for `Layout.Grid`. Add `Recipe` to `Options` without removing existing fields:

```go
type Options struct {
	Width               int
	FPS                 float64
	Bitrate             string
	Overwrite           bool
	ProgressInteractive bool
	ProgressClock       func() time.Time
	Stats               bool
	Workers             int
	MetricsClock        func() time.Time
	InputLabel          string
	HasAt              bool
	AtSeconds          float64
	HasDuration        bool
	DurationSeconds    float64
	Format             OutputFormat
	Recipe             render.Recipe
}
```

Keep existing fields in their current order as much as possible; only add `Recipe render.Recipe`.

- [ ] **Step 5: Use selected recipe in exporter rendering**

In `packages/core/internal/exporter/export.go`, add a helper:

```go
func recipeOrDefault(recipe render.Recipe) render.Recipe {
	if recipe.Name == "" {
		return render.DefaultRecipe()
	}
	return recipe
}
```

Update render sites:

```go
NewProcessor: newExportFrameProcessorFactory(layout, metrics, metricsClock, recipeOrDefault(options.Recipe)),
```

```go
charFrame := render.NewRenderer(recipeOrDefault(options.Recipe)).Render(frame, layout.Grid)
```

```go
return render.NewRenderer(recipeOrDefault(options.Recipe)).Render(rgbFrame, layout.Grid), nil
```

- [ ] **Step 6: Update export frame processor factory signature**

In `packages/core/internal/exporter/pipeline.go`, change:

```go
func newExportFrameProcessorFactory(layout Layout, metrics *exportMetrics, clock exportClock, recipe render.Recipe) func() (exportFrameProcessor, error) {
	if clock == nil {
		clock = realExportClock{}
	}
	recipe = recipeOrDefault(recipe)
	return func() (exportFrameProcessor, error) {
		face, err := fonts.DefaultFace()
		if err != nil {
			return nil, fmt.Errorf("load export font: %w", err)
		}
		rasterizer := NewRasterizer(face)
		renderer := render.NewRenderer(recipe)

		return func(_ int, rgbFrame render.RGBFrame) ([]byte, error) {
			renderStart := clock.Now()
			charFrame := renderer.Render(rgbFrame, layout.Grid)
			if metrics != nil {
				metrics.RecordRender(clock.Now().Sub(renderStart))
			}

			rasterizeStart := clock.Now()
			raw, err := rasterizer.Rasterize(charFrame, layout)
			if err != nil {
				return nil, fmt.Errorf("rasterize frame: %w", err)
			}
			if metrics != nil {
				metrics.RecordRasterize(clock.Now().Sub(rasterizeStart))
			}
			return raw, nil
		}, nil
	}
}
```

Update all call sites to pass `recipeOrDefault(options.Recipe)` or `render.DefaultRecipe()`.

- [ ] **Step 7: Run exporter tests and verify they pass**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 8: Commit exporter recipe routing**

Commit:

```bash
git add packages/core/internal/exporter/layout.go packages/core/internal/exporter/export.go packages/core/internal/exporter/pipeline.go packages/core/internal/exporter/pipeline_test.go packages/core/internal/exporter/export_test.go
git commit --no-gpg-sign -m "feat: route recipes through exports"
```

---

### Task 4: Parse and Route `--recipe` Through CLI Commands

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`
- Modify: `packages/core/internal/cli/play.go`
- Modify: `packages/core/internal/cli/play_test.go`
- Modify: `packages/core/internal/cli/export.go`
- Modify: `packages/core/internal/cli/export_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Add failing CLI parser tests**

Append to `packages/core/internal/cli/cli_test.go`:

```go
func TestParsePlayRecipe(t *testing.T) {
	cmd, err := Parse([]string{"play", "--recipe", "blocks", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Play.Recipe.Name != "blocks" {
		t.Fatalf("play recipe = %q, want blocks", cmd.Play.Recipe.Name)
	}
}

func TestParseExportRecipe(t *testing.T) {
	cmd, err := Parse([]string{"export", "--recipe", "mono", "clip.mp4", "out.ansi"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Export.Recipe.Name != "mono" {
		t.Fatalf("export recipe = %q, want mono", cmd.Export.Recipe.Name)
	}
}

func TestParseRecipeRejectsUnknownBeforeCommandExecution(t *testing.T) {
	_, err := Parse([]string{"play", "--recipe", "banana", "https://example.com/watch?v=demo"})
	if err == nil {
		t.Fatal("Parse returned nil error for unknown recipe")
	}
	if !strings.Contains(err.Error(), `unsupported recipe "banana"`) {
		t.Fatalf("error = %v, want unsupported recipe", err)
	}
	if !strings.Contains(err.Error(), "default, mono, ascii, blocks") {
		t.Fatalf("error = %v, want supported recipe list", err)
	}
}

func TestParseProbeRejectsRecipe(t *testing.T) {
	_, err := Parse([]string{"probe", "--recipe", "blocks", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for probe --recipe")
	}
	if !strings.Contains(err.Error(), "probe does not accept --recipe") {
		t.Fatalf("error = %v, want probe --recipe rejection", err)
	}
}

func TestParseRecipeRequiresValue(t *testing.T) {
	_, err := Parse([]string{"play", "--recipe"})
	if err == nil {
		t.Fatal("Parse returned nil error for missing recipe value")
	}
	if !strings.Contains(err.Error(), "play requires a value for --recipe") {
		t.Fatalf("error = %v, want missing recipe value", err)
	}
}
```

- [ ] **Step 2: Run CLI parser tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestParse.*Recipe'
```

Expected: fail to compile because `cmd.Play` and recipe option fields do not exist, or fail because `--recipe` is unknown.

- [ ] **Step 3: Add play options to command model**

In `packages/core/internal/cli/cli.go`, import render:

```go
import (
	...
	"github.com/jass/mojify/packages/core/internal/render"
)
```

Add `Recipe` fields:

```go
type PlayOptions struct {
	Stats   bool
	NoAudio bool
	Recipe  render.Recipe
}

type ExportOptions struct {
	Width           int
	FPS             float64
	Bitrate         string
	Overwrite       bool
	Stats           bool
	Workers         int
	HasAt           bool
	AtSeconds       float64
	HasDuration     bool
	DurationSeconds float64
	Recipe          render.Recipe
}
```

Change `Command` to carry `Play PlayOptions` instead of only top-level `Stats`/`NoAudio`:

```go
type Command struct {
	Kind       CommandKind
	InputPath  string
	OutputPath string
	Play       PlayOptions
	Export     ExportOptions
}
```

- [ ] **Step 4: Parse `--recipe` for play and reject for probe**

In `parseInputCommand`, track recipe:

```go
recipe := render.DefaultRecipe()
seenRecipe := false
```

Add a switch case:

```go
case "--recipe":
	if kind != PlayCommand {
		return Command{}, fmt.Errorf("%s does not accept --recipe", args[0])
	}
	if seenRecipe {
		return Command{}, fmt.Errorf("%s accepts --recipe only once", args[0])
	}
	if i+1 >= len(args) {
		return Command{}, fmt.Errorf("%s requires a value for --recipe", args[0])
	}
	i++
	selected, err := render.RecipeByName(args[i])
	if err != nil {
		return Command{}, err
	}
	recipe = selected
	seenRecipe = true
```

This requires changing the loop from `for _, arg := range args[1:]` to an index loop:

```go
for i := 1; i < len(args); i++ {
	arg := args[i]
	...
}
```

Return:

```go
return Command{
	Kind:      kind,
	InputPath: inputPath,
	Play: PlayOptions{
		Stats:   stats,
		NoAudio: noAudio,
		Recipe:  recipe,
	},
}, nil
```

- [ ] **Step 5: Parse `--recipe` for export**

In `parseExportCommand`, initialize:

```go
options := ExportOptions{Recipe: render.DefaultRecipe()}
seenRecipe := false
```

Add case:

```go
case "--recipe":
	if seenRecipe {
		return Command{}, fmt.Errorf("export accepts --recipe only once")
	}
	if i+1 >= len(args) {
		return Command{}, fmt.Errorf("export requires a value for --recipe")
	}
	i++
	recipe, err := render.RecipeByName(args[i])
	if err != nil {
		return Command{}, err
	}
	options.Recipe = recipe
	seenRecipe = true
```

- [ ] **Step 6: Update CLI help**

In `HelpText`, add:

```text
  --recipe <name>     Built-in recipe preset: default, mono, ascii, blocks
```

under Play options and Export options.

- [ ] **Step 7: Update main command dispatch**

In `packages/core/cmd/mojify/main.go`, update play dispatch to pass `cmd.Play`:

```go
if err := cli.RunPlay(ctx, cmd.InputPath, os.Stdin, os.Stdout, os.Stderr, cmd.Play); err != nil {
	fmt.Fprintf(os.Stderr, "play failed: %v\n", err)
	os.Exit(1)
}
```

Update export dispatch to continue passing `cmd.Export`.

- [ ] **Step 8: Route recipe through export options**

In `packages/core/internal/cli/export.go`, add:

```go
Recipe: options.Recipe,
```

to `exporter.Options`.

- [ ] **Step 9: Route recipe through play rendering**

In `packages/core/internal/cli/play.go`, replace:

```go
renderer := render.DefaultRenderer{}
```

with:

```go
renderer := render.NewRenderer(recipeOrDefault(options.Recipe))
```

Add this CLI-local helper:

```go
func recipeOrDefault(recipe render.Recipe) render.Recipe {
	if recipe.Name == "" {
		return render.DefaultRecipe()
	}
	return recipe
}
```

If exporter already has an unexported helper with the same name, keep the CLI helper local to the `cli` package.

- [ ] **Step 10: Add CLI handoff tests**

In `packages/core/internal/cli/export_test.go`, add:

```go
func TestRunExportPassesRecipeToExporter(t *testing.T) {
	exportErr := errors.New("stop after export handoff")
	var gotOptions exporter.Options

	err := runExportWithOptions(context.Background(), "clip.mov", "out.png", io.Discard, ExportOptions{
		Recipe: render.MustRecipeByName("blocks"),
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotOptions = options
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if gotOptions.Recipe.Name != "blocks" {
		t.Fatalf("recipe = %q, want blocks", gotOptions.Recipe.Name)
	}
}
```

In `packages/core/internal/cli/play_test.go`, add a test that uses `runPlayWithOptions` with `PlayOptions{Recipe: render.MustRecipeByName("blocks")}` and a probe sentinel error after source resolution. This verifies the option compiles and flows to the play runner without changing source resolution. If direct renderer inspection is not practical in the current play tests, parser coverage plus successful compile is acceptable for play routing in this stage.

- [ ] **Step 11: Run CLI tests and verify they pass**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli
```

Expected: pass.

- [ ] **Step 12: Commit CLI recipe surface**

Commit:

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go packages/core/cmd/mojify/main.go
git commit --no-gpg-sign -m "feat: add recipe option to play and export"
```

---

### Task 5: Add Recipe Documentation and QA Matrix

**Files:**
- Create: `docs/recipes.md`
- Modify: `README.md`
- Modify: `docs/qa/export.md`
- Modify: `scripts/export-qa.sh`

- [ ] **Step 1: Create recipe docs**

Create `docs/recipes.md`:

````md
# Recipes

Mojify recipes control how source pixels become character frames. The current release supports built-in recipe presets selected with `--recipe <name>` on commands that render frames.

```bash
mojify play --recipe blocks ./demo.mp4
mojify export --recipe mono ./poster.png ./dist/poster-mono.ansi
```

`probe` does not accept recipes because recipes do not change source metadata or derived layout.

## Built-In Presets

| Preset | Character mapping | Color | Edges |
| --- | --- | --- | --- |
| `default` | Mojify's default ramp, ` .;coPO?#@` | source color | default edge glyphs |
| `mono` | Mojify's default ramp, ` .;coPO?#@` | none | default edge glyphs |
| `ascii` | classic ASCII ramp, ` .:-=+*#%@` | none | none |
| `blocks` | Unicode shade/block ramp, ` ░▒▓█` | source color | none |

No `--recipe` flag is the same as `--recipe default`.

For text output, no-color recipes write plain characters. For ANSI output, no-color recipes do not emit foreground color escapes. For raster image, animation, and video exports, no-color recipes render white glyphs on black.

## Custom Recipes

Custom recipe files are planned as a future stage. They should use a separate explicit surface, likely `--recipe-file <path>`, rather than overloading `--recipe` with both preset names and file paths.

Mojify does not currently publish or support a custom recipe file schema.
````

- [ ] **Step 2: Update README usage**

In `README.md`, add one compact recipe example near the export examples:

````md
Choose a built-in recipe preset:

```bash
mojify export --overwrite --recipe blocks --width 320 ./poster.png ./dist/poster-blocks.png
```
````

In current capabilities, add:

```md
- Built-in recipe presets: `default`, `mono`, `ascii`, and `blocks`
```

In the renderer section, link the docs:

```md
See [Recipes](docs/recipes.md) for built-in preset behavior and the future custom recipe direction.
```

- [ ] **Step 3: Update export QA docs**

In `docs/qa/export.md`, add recipe matrix outputs to the expected generated output list:

```md
- `dist/qa/export/recipe-default.png`
- `dist/qa/export/recipe-default.ansi`
- `dist/qa/export/recipe-mono.png`
- `dist/qa/export/recipe-mono.ansi`
- `dist/qa/export/recipe-ascii.png`
- `dist/qa/export/recipe-ascii.ansi`
- `dist/qa/export/recipe-blocks.png`
- `dist/qa/export/recipe-blocks.ansi`
```

Add checklist entries:

```md
- Recipe preset matrix exports still-source `.png` and `.ansi` outputs for `default`, `mono`, `ascii`, and `blocks`.
- Recipe preset image outputs have width `320`.
- Recipe preset ANSI outputs are non-empty.
- `mono` and `ascii` ANSI outputs do not emit foreground color escapes.
```

Add rows to the QA matrix for the recipe preset matrix.

- [ ] **Step 4: Update export QA script**

In `scripts/export-qa.sh`, add after still-source checks:

```bash
printf '\nExporting still-source recipe preset matrix...\n'
for recipe in default mono ascii blocks; do
  recipe_png="${export_dir}/recipe-${recipe}.png"
  recipe_ansi="${export_dir}/recipe-${recipe}.ansi"

  ./bin/mojify export --overwrite --recipe "${recipe}" --width 320 "${still_source}" "${recipe_png}"
  ./bin/mojify export --overwrite --recipe "${recipe}" --width 80 "${still_source}" "${recipe_ansi}"

  check_video_width "${recipe_png}" "320"
  require_nonempty_file "${recipe_ansi}"

  if [[ "${recipe}" == "mono" || "${recipe}" == "ascii" ]]; then
    if LC_ALL=C grep -q "$(printf '\033\\[38;2;')" "${recipe_ansi}"; then
      printf 'Expected %s ANSI recipe output to avoid foreground color escapes.\n' "${recipe}" >&2
      exit 1
    fi
  fi
done
```

- [ ] **Step 5: Run docs/QA syntax checks**

Run:

```bash
bash -n scripts/export-qa.sh
```

Expected: pass.

- [ ] **Step 6: Commit docs and QA matrix**

Commit:

```bash
git add docs/recipes.md README.md docs/qa/export.md scripts/export-qa.sh
git commit --no-gpg-sign -m "docs: document recipe presets"
```

---

### Task 6: End-to-End Verification and Cleanup

**Files:**
- All modified files from previous tasks.
- Modified: `docs/superpowers/plans/2026-06-05-mojify-built-in-recipe-presets.md`
  - Check off completed task boxes if the implementation workflow tracks progress in the plan.

- [ ] **Step 1: Run formatting**

Run:

```bash
bun run fmt:check
```

Expected: pass. If it fails, run:

```bash
bun run fmt
```

Then rerun `bun run fmt:check`.

- [ ] **Step 2: Run Go tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...
```

Expected: pass.

- [ ] **Step 3: Run repo tests**

Run:

```bash
bun run test
```

Expected: pass.

- [ ] **Step 4: Run build**

Run:

```bash
bun run build
```

Expected: pass. A non-fatal Go stat-cache warning from the sandboxed module cache is acceptable only if the command exits with code 0.

- [ ] **Step 5: Run export QA**

Run:

```bash
bun run qa:clips
bun run qa:export
```

Expected:

- Existing export QA still passes.
- Recipe preset matrix exports `.png` and `.ansi` outputs for `default`, `mono`, `ascii`, and `blocks`.
- `mono` and `ascii` ANSI outputs contain no truecolor foreground escape sequences.

- [ ] **Step 6: Manual smoke selected CLI errors**

Run:

```bash
./bin/mojify play --recipe banana dist/qa/low-motion-bars.mp4
```

Expected exit code: non-zero.

Expected stderr contains:

```text
unsupported recipe "banana"; supported recipes: default, mono, ascii, blocks
```

Run:

```bash
./bin/mojify probe --recipe blocks dist/qa/low-motion-bars.mp4
```

Expected exit code: non-zero.

Expected stderr contains:

```text
probe does not accept --recipe
```

- [ ] **Step 7: Check diff cleanliness**

Run:

```bash
git diff --check
git status -sb
```

Expected: no whitespace errors. Status should show only intended files.

- [ ] **Step 8: Commit final plan progress updates when present**

When plan checkbox updates remain after implementation commits, commit them:

```bash
git add docs/superpowers/plans/2026-06-05-mojify-built-in-recipe-presets.md
git commit --no-gpg-sign -m "docs: finalize recipe preset plan"
```

If no files remain unstaged, skip this commit.

## Self-Review Checklist

- Spec coverage:
  - Built-in preset names are implemented in Task 1.
  - `HasColor` and no-color output behavior are implemented in Task 2.
  - Export routing is implemented in Task 3.
  - CLI parse and playback/export routing are implemented in Task 4.
  - README, `docs/recipes.md`, export QA docs, and QA script are implemented in Task 5.
  - Verification is covered in Task 6.
- Placeholder scan:
  - The plan contains no unresolved placeholder markers.
  - Out-of-scope custom recipe files are documented as future product surface only.
- Type consistency:
  - `render.Recipe`, `render.RecipeByName`, `render.MustRecipeByName`, `render.NewRenderer`, `render.DefaultRecipe`, and `render.Cell.HasColor` are introduced before later tasks use them.
  - `exporter.Options.Recipe` is introduced before CLI export handoff uses it.
  - `PlayOptions.Recipe` and `ExportOptions.Recipe` are introduced before `main.go` dispatch uses them.
