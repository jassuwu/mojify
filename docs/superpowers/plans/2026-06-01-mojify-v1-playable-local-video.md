# Mojify V1 Playable Local Video Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first source-build `mojify` milestone: `mojify play <local-video>` renders truecolor, edge-aware character frames smoothly in the terminal, and `mojify probe <local-video>` reports media/render metadata.

**Architecture:** The repo is a Bun/Turbo monorepo with a Go-first product. The Go core shells out to `ffprobe`/`ffmpeg`, normalizes decoded RGB frames, renders them into character frames, buffers them, and presents them in the terminal with minimal controls.

**Tech Stack:** Go 1.23+, FFmpeg CLI, Bun workspaces, Turbo, MIT license, `golang.org/x/term` for terminal size/raw mode.

---

## File Structure

- `package.json`: private monorepo root, Bun workspace definitions, shared scripts.
- `turbo.json`: task graph for build/test/dev.
- `go.mod`: root Go module for the v1 native core.
- `packages/core/package.json`: Turbo facade for the Go core package.
- `packages/core/cmd/mojify/main.go`: binary entrypoint.
- `packages/core/internal/cli`: command parsing, help text, CLI dispatch.
- `packages/core/internal/media`: `ffprobe` metadata and `ffmpeg` raw RGB frame decoding.
- `packages/core/internal/render`: render grid calculation and default renderer.
- `packages/core/internal/terminal`: ANSI frame serialization, terminal lifecycle, input controls.
- `packages/core/internal/player`: bounded-buffer playback scheduler.
- `.github/workflows/ci.yml`: first CI gate for source-build v1.
- `docs/superpowers/plans/2026-06-01-mojify-v1-playable-local-video.md`: this plan.

## Task 1: Monorepo Skeleton

**Files:**
- Create: `package.json`
- Create: `turbo.json`
- Create: `.gitignore`
- Create: `LICENSE`
- Create: `README.md`
- Create: `go.mod`
- Create: `packages/core/package.json`

- [ ] **Step 1: Create root package manifest**

Create `package.json`:

```json
{
  "name": "mojify-monorepo",
  "private": true,
  "license": "MIT",
  "type": "module",
  "workspaces": {
    "packages": [
      "apps/*",
      "packages/*",
      "scripts"
    ],
    "catalog": {
      "turbo": "^2.5.4"
    }
  },
  "scripts": {
    "build": "turbo run build",
    "test": "turbo run test",
    "typecheck": "turbo run typecheck",
    "dev:core": "turbo run dev --filter=@mojify/core",
    "fmt": "gofmt -w packages/core",
    "fmt:check": "test -z \"$(gofmt -l packages/core)\""
  },
  "devDependencies": {
    "turbo": "catalog:"
  },
  "engines": {
    "bun": ">=1.3.0",
    "node": ">=20"
  },
  "packageManager": "bun@1.3.11"
}
```

- [ ] **Step 2: Create Turbo task graph**

Create `turbo.json`:

```json
{
  "$schema": "https://turbo.build/schema.json",
  "tasks": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["bin/**", "dist/**"]
    },
    "dev": {
      "cache": false,
      "persistent": true
    },
    "test": {
      "cache": false,
      "outputs": []
    },
    "typecheck": {
      "cache": false,
      "outputs": []
    }
  }
}
```

- [ ] **Step 3: Create ignore file**

Create `.gitignore`:

```gitignore
node_modules
.turbo
bin
dist
coverage
.DS_Store
*.testbin
```

- [ ] **Step 4: Create MIT license**

Create `LICENSE`:

```text
MIT License

Copyright (c) 2026 mojify contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 5: Create first README**

Create `README.md`:

````markdown
# mojify

Terminal-first video playback with colored, edge-aware character frames.

## V1 Scope

- Local video files only.
- Visual playback only, no audio.
- FFmpeg CLI required.
- Source-build distribution first.

## Development

```bash
bun install
bun run build
bun run test
go run ./packages/core/cmd/mojify --help
```
````

- [ ] **Step 6: Create Go module**

Create `go.mod`:

```go
module github.com/jass/mojify

go 1.23

require golang.org/x/term v0.31.0
```

- [ ] **Step 7: Create Go core workspace facade**

Create `packages/core/package.json`:

```json
{
  "name": "@mojify/core",
  "private": true,
  "license": "MIT",
  "type": "module",
  "scripts": {
    "build": "go build -o ../../bin/mojify ./cmd/mojify",
    "dev": "go run ./cmd/mojify --help",
    "test": "go test ./...",
    "typecheck": "go test ./...",
    "fmt": "gofmt -w .",
    "fmt:check": "test -z \"$(gofmt -l .)\""
  }
}
```

- [ ] **Step 8: Verify skeleton commands**

Run:

```bash
bun install
bun run test
bun run build
```

Expected:

```text
turbo ... no package tasks fail
```

- [ ] **Step 9: Commit**

```bash
git add package.json turbo.json .gitignore LICENSE README.md go.mod packages/core/package.json CONTEXT.md docs/adr
git commit -m "chore: initialize mojify monorepo"
```

## Task 2: CLI Help And Command Parsing

**Files:**
- Create: `packages/core/cmd/mojify/main.go`
- Create: `packages/core/internal/cli/cli.go`
- Create: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Write CLI parser tests**

Create `packages/core/internal/cli/cli_test.go`:

```go
package cli

import "testing"

func TestParseBareCommandShowsHelp(t *testing.T) {
	cmd, err := Parse([]string{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != HelpCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, HelpCommand)
	}
}

func TestParsePlayCommand(t *testing.T) {
	cmd, err := Parse([]string{"play", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != PlayCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, PlayCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
}

func TestParseProbeCommand(t *testing.T) {
	cmd, err := Parse([]string{"probe", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != ProbeCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, ProbeCommand)
	}
}

func TestParseMissingInput(t *testing.T) {
	_, err := Parse([]string{"play"})
	if err == nil {
		t.Fatal("Parse returned nil error for missing input")
	}
}

func TestHelpTextMentionsCommands(t *testing.T) {
	help := HelpText()
	for _, want := range []string{"mojify play <video>", "mojify probe <video>", "FFmpeg"} {
		if !contains(help, want) {
			t.Fatalf("HelpText() missing %q in:\n%s", want, help)
		}
	}
}

func contains(s string, needle string) bool {
	for i := 0; i+len(needle) <= len(s); i++ {
		if s[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected:

```text
FAIL ... undefined: Parse
```

- [ ] **Step 3: Implement parser**

Create `packages/core/internal/cli/cli.go`:

```go
package cli

import (
	"errors"
	"fmt"
)

type CommandKind int

const (
	HelpCommand CommandKind = iota
	PlayCommand
	ProbeCommand
)

type Command struct {
	Kind      CommandKind
	InputPath string
}

func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{Kind: HelpCommand}, nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		return Command{Kind: HelpCommand}, nil
	case "play":
		if len(args) < 2 {
			return Command{}, errors.New("play requires a local video path")
		}
		return Command{Kind: PlayCommand, InputPath: args[1]}, nil
	case "probe":
		if len(args) < 2 {
			return Command{}, errors.New("probe requires a local video path")
		}
		return Command{Kind: ProbeCommand, InputPath: args[1]}, nil
	default:
		return Command{}, fmt.Errorf("unknown command %q", args[0])
	}
}

func HelpText() string {
	return `mojify

Terminal-first video playback with colored, edge-aware character frames.

Usage:
  mojify play <video>    Play a local video file in the terminal
  mojify probe <video>   Print media and render metadata
  mojify --help          Show this help

Requirements:
  FFmpeg and ffprobe must be available on PATH for v1.
`
}
```

- [ ] **Step 4: Create entrypoint**

Create `packages/core/cmd/mojify/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/jass/mojify/packages/core/internal/cli"
)

func main() {
	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, cli.HelpText())
		os.Exit(2)
	}

	switch cmd.Kind {
	case cli.HelpCommand:
		fmt.Print(cli.HelpText())
	case cli.PlayCommand:
		fmt.Fprintf(os.Stderr, "play is not implemented yet: %s\n", cmd.InputPath)
		os.Exit(1)
	case cli.ProbeCommand:
		fmt.Fprintf(os.Stderr, "probe is not implemented yet: %s\n", cmd.InputPath)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Run tests and help**

Run:

```bash
go test ./packages/core/internal/cli
go run ./packages/core/cmd/mojify --help
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/cli
mojify
```

- [ ] **Step 6: Commit**

```bash
git add packages/core/cmd/mojify packages/core/internal/cli
git commit -m "feat: add mojify cli skeleton"
```

## Task 3: Media Probe Via ffprobe

**Files:**
- Create: `packages/core/internal/media/probe.go`
- Create: `packages/core/internal/media/probe_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Write probe JSON parser tests**

Create `packages/core/internal/media/probe_test.go`:

```go
package media

import "testing"

func TestParseProbeJSON(t *testing.T) {
	const input = `{
	  "streams": [
	    {
	      "codec_type": "video",
	      "width": 1920,
	      "height": 1080,
	      "avg_frame_rate": "30000/1001",
	      "nb_frames": "240"
	    }
	  ],
	  "format": { "duration": "8.008000" }
	}`

	info, err := ParseProbeJSON([]byte(input))
	if err != nil {
		t.Fatalf("ParseProbeJSON returned error: %v", err)
	}
	if info.Width != 1920 || info.Height != 1080 {
		t.Fatalf("size = %dx%d, want 1920x1080", info.Width, info.Height)
	}
	if info.FPS < 29.96 || info.FPS > 29.98 {
		t.Fatalf("FPS = %f, want about 29.97", info.FPS)
	}
	if info.FrameCount != 240 {
		t.Fatalf("FrameCount = %d, want 240", info.FrameCount)
	}
	if info.DurationSeconds != 8.008 {
		t.Fatalf("DurationSeconds = %f, want 8.008", info.DurationSeconds)
	}
}

func TestParseProbeJSONRejectsMissingVideo(t *testing.T) {
	_, err := ParseProbeJSON([]byte(`{"streams":[]}`))
	if err == nil {
		t.Fatal("ParseProbeJSON returned nil error for missing video stream")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/media
```

Expected:

```text
FAIL ... undefined: ParseProbeJSON
```

- [ ] **Step 3: Implement probe parser and runner**

Create `packages/core/internal/media/probe.go`:

```go
package media

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Width           int
	Height          int
	FPS             float64
	FrameCount      int
	DurationSeconds float64
}

type probePayload struct {
	Streams []probeStream `json:"streams"`
	Format  probeFormat   `json:"format"`
}

type probeStream struct {
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
	NBFrames     string `json:"nb_frames"`
}

type probeFormat struct {
	Duration string `json:"duration"`
}

func Probe(path string) (Info, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		path,
	).Output()
	if err != nil {
		return Info{}, fmt.Errorf("run ffprobe: %w", err)
	}
	return ParseProbeJSON(out)
}

func ParseProbeJSON(data []byte) (Info, error) {
	var payload probePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return Info{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	for _, stream := range payload.Streams {
		if stream.CodecType != "video" {
			continue
		}

		fps, err := parseRate(stream.AvgFrameRate)
		if err != nil {
			return Info{}, err
		}
		frameCount, _ := strconv.Atoi(stream.NBFrames)
		duration, _ := strconv.ParseFloat(payload.Format.Duration, 64)

		return Info{
			Width:           stream.Width,
			Height:          stream.Height,
			FPS:             fps,
			FrameCount:      frameCount,
			DurationSeconds: duration,
		}, nil
	}

	return Info{}, errors.New("no video stream found")
}

func parseRate(rate string) (float64, error) {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid frame rate %q", rate)
	}
	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid frame rate numerator %q: %w", parts[0], err)
	}
	den, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid frame rate denominator %q: %w", parts[1], err)
	}
	if den == 0 {
		return 0, fmt.Errorf("invalid frame rate denominator 0")
	}
	return num / den, nil
}
```

- [ ] **Step 4: Wire `mojify probe`**

Modify `packages/core/cmd/mojify/main.go` so the `ProbeCommand` case becomes:

```go
	case cli.ProbeCommand:
		info, err := media.Probe(cmd.InputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "probe failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("input: %s\n", cmd.InputPath)
		fmt.Printf("video: %dx%d\n", info.Width, info.Height)
		fmt.Printf("fps: %.3f\n", info.FPS)
		fmt.Printf("frames: %d\n", info.FrameCount)
		fmt.Printf("duration: %.3fs\n", info.DurationSeconds)
```

Also add the import:

```go
	"github.com/jass/mojify/packages/core/internal/media"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./packages/core/internal/media
go test ./...
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/media
```

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/media packages/core/cmd/mojify/main.go
git commit -m "feat: add ffprobe media probe"
```

## Task 4: Render Grid Auto-Fit

**Files:**
- Create: `packages/core/internal/render/grid.go`
- Create: `packages/core/internal/render/grid_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Write grid tests**

Create `packages/core/internal/render/grid_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/render
```

Expected:

```text
FAIL ... undefined: FitGrid
```

- [ ] **Step 3: Implement grid fitting**

Create `packages/core/internal/render/grid.go`:

```go
package render

import "math"

const CellAspect = 2.0

type InputSize struct {
	Width  int
	Height int
}

type TerminalSize struct {
	Cols int
	Rows int
}

type Grid struct {
	Cols int
	Rows int
}

func FitGrid(input InputSize, term TerminalSize) Grid {
	maxCols := max(term.Cols, 10)
	maxRows := max(term.Rows, 5)
	if input.Width <= 0 || input.Height <= 0 {
		return Grid{Cols: maxCols, Rows: maxRows}
	}

	aspect := float64(input.Width) / float64(input.Height)
	rowsFromCols := int(math.Floor(float64(maxCols) / (aspect * CellAspect)))
	if rowsFromCols >= 5 && rowsFromCols <= maxRows {
		return Grid{Cols: maxCols, Rows: rowsFromCols}
	}

	colsFromRows := int(math.Floor(float64(maxRows) * aspect * CellAspect))
	return Grid{Cols: max(10, min(maxCols, colsFromRows)), Rows: maxRows}
}
```

- [ ] **Step 4: Add render grid to probe output**

Modify `packages/core/cmd/mojify/main.go` in the `ProbeCommand` case after printing duration:

```go
		grid := render.FitGrid(
			render.InputSize{Width: info.Width, Height: info.Height},
			render.TerminalSize{Cols: 120, Rows: 40},
		)
		fmt.Printf("render-grid: %dx%d (sample terminal 120x40)\n", grid.Cols, grid.Rows)
```

Add import:

```go
	"github.com/jass/mojify/packages/core/internal/render"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./packages/core/internal/render
go test ./...
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/render
```

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/render packages/core/cmd/mojify/main.go
git commit -m "feat: add render grid auto-fit"
```

## Task 5: Default Renderer

**Files:**
- Create: `packages/core/internal/render/frame.go`
- Create: `packages/core/internal/render/renderer.go`
- Create: `packages/core/internal/render/renderer_test.go`

- [ ] **Step 1: Write golden renderer tests**

Create `packages/core/internal/render/renderer_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/render
```

Expected:

```text
FAIL ... undefined: NewRGBFrame
```

- [ ] **Step 3: Implement frame types**

Create `packages/core/internal/render/frame.go`:

```go
package render

type RGBFrame struct {
	Width  int
	Height int
	Data   []byte
}

type Cell struct {
	Ch    rune
	R, G, B uint8
}

type CharacterFrame struct {
	Width  int
	Height int
	Cells  []Cell
}

func NewRGBFrame(width int, height int, data []byte) RGBFrame {
	return RGBFrame{Width: width, Height: height, Data: data}
}

func (f RGBFrame) RGBAt(x int, y int) (uint8, uint8, uint8) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= f.Width {
		x = f.Width - 1
	}
	if y >= f.Height {
		y = f.Height - 1
	}
	offset := (y*f.Width + x) * 3
	return f.Data[offset], f.Data[offset+1], f.Data[offset+2]
}
```

- [ ] **Step 4: Implement renderer**

Create `packages/core/internal/render/renderer.go`:

```go
package render

import "math"

const densityRamp = " .;coPO?#@"

type DefaultRenderer struct{}

func (DefaultRenderer) Render(frame RGBFrame, grid Grid) CharacterFrame {
	cells := make([]Cell, grid.Cols*grid.Rows)
	for gy := 0; gy < grid.Rows; gy++ {
		for gx := 0; gx < grid.Cols; gx++ {
			sx := gx * frame.Width / grid.Cols
			sy := gy * frame.Height / grid.Rows
			r, g, b := frame.RGBAt(sx, sy)
			luma := luminance(r, g, b)
			ch := densityChar(luma)
			if edge, ok := edgeGlyph(frame, sx, sy); ok {
				ch = edge
			}
			cells[gy*grid.Cols+gx] = Cell{Ch: ch, R: r, G: g, B: b}
		}
	}
	return CharacterFrame{Width: grid.Cols, Height: grid.Rows, Cells: cells}
}

func luminance(r uint8, g uint8, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

func densityChar(luma float64) rune {
	index := int(math.Round((luma / 255.0) * float64(len([]rune(densityRamp))-1)))
	if index < 0 {
		index = 0
	}
	runes := []rune(densityRamp)
	if index >= len(runes) {
		index = len(runes) - 1
	}
	return runes[index]
}

func edgeGlyph(frame RGBFrame, x int, y int) (rune, bool) {
	if frame.Width < 3 || frame.Height < 3 {
		return 0, false
	}
	gx := sobelX(frame, x, y)
	gy := sobelY(frame, x, y)
	mag := math.Sqrt(gx*gx + gy*gy)
	if mag < 180 {
		return 0, false
	}
	angle := math.Atan2(gy, gx) * 180 / math.Pi
	if angle < 0 {
		angle += 180
	}
	switch {
	case angle < 22.5 || angle >= 157.5:
		return '|', true
	case angle < 67.5:
		return '/', true
	case angle < 112.5:
		return '-', true
	default:
		return '\\', true
	}
}

func grayAt(frame RGBFrame, x int, y int) float64 {
	r, g, b := frame.RGBAt(x, y)
	return luminance(r, g, b)
}

func sobelX(frame RGBFrame, x int, y int) float64 {
	return -grayAt(frame, x-1, y-1) + grayAt(frame, x+1, y-1) -
		2*grayAt(frame, x-1, y) + 2*grayAt(frame, x+1, y) -
		grayAt(frame, x-1, y+1) + grayAt(frame, x+1, y+1)
}

func sobelY(frame RGBFrame, x int, y int) float64 {
	return -grayAt(frame, x-1, y-1) - 2*grayAt(frame, x, y-1) - grayAt(frame, x+1, y-1) +
		grayAt(frame, x-1, y+1) + 2*grayAt(frame, x, y+1) + grayAt(frame, x+1, y+1)
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./packages/core/internal/render
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/render
```

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/render
git commit -m "feat: add edge-aware truecolor renderer"
```

## Task 6: ANSI Terminal Frame Serialization

**Files:**
- Create: `packages/core/internal/terminal/ansi.go`
- Create: `packages/core/internal/terminal/ansi_test.go`

- [ ] **Step 1: Write ANSI tests**

Create `packages/core/internal/terminal/ansi_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected:

```text
FAIL ... undefined: SerializeFrame
```

- [ ] **Step 3: Implement ANSI serializer**

Create `packages/core/internal/terminal/ansi.go`:

```go
package terminal

import (
	"fmt"
	"strings"

	"github.com/jass/mojify/packages/core/internal/render"
)

const (
	EnterAltScreen = "\x1b[?1049h"
	ExitAltScreen  = "\x1b[?1049l"
	HideCursor     = "\x1b[?25l"
	ShowCursor     = "\x1b[?25h"
	CursorHome     = "\x1b[H"
	ClearToEnd     = "\x1b[J"
	Reset          = "\x1b[0m"
)

func SerializeFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	b.WriteString(CursorHome)
	b.WriteString(ClearToEnd)
	var lastR, lastG, lastB uint8
	hasColor := false

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			cell := frame.Cells[y*frame.Width+x]
			if !hasColor || cell.R != lastR || cell.G != lastG || cell.B != lastB {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
				lastR, lastG, lastB = cell.R, cell.G, cell.B
				hasColor = true
			}
			b.WriteRune(cell.Ch)
		}
		if y != frame.Height-1 {
			b.WriteByte('\n')
		}
	}

	b.WriteString(Reset)
	return b.String()
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/terminal
```

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/terminal
git commit -m "feat: add ansi frame serializer"
```

## Task 7: FFmpeg Raw Frame Decoder

**Files:**
- Create: `packages/core/internal/media/decode.go`
- Create: `packages/core/internal/media/decode_test.go`

- [ ] **Step 1: Write decoder command and raw reader tests**

Create `packages/core/internal/media/decode_test.go`:

```go
package media

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestDecodeArgs(t *testing.T) {
	args := DecodeArgs("clip.mp4", 320, 180)
	want := []string{
		"-v", "error",
		"-i", "clip.mp4",
		"-vf", "scale=320:180:force_original_aspect_ratio=decrease",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestReadRawFrame(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	frame, err := ReadRawFrame(bytes.NewReader(data), 1, 2)
	if err != nil {
		t.Fatalf("ReadRawFrame returned error: %v", err)
	}
	if frame.Width != 1 || frame.Height != 2 {
		t.Fatalf("size = %dx%d, want 1x2", frame.Width, frame.Height)
	}
	if !reflect.DeepEqual(frame.Data, data) {
		t.Fatalf("data = %#v, want %#v", frame.Data, data)
	}
}

func TestReadRawFrameEOF(t *testing.T) {
	_, err := ReadRawFrame(bytes.NewReader(nil), 1, 1)
	if err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/media
```

Expected:

```text
FAIL ... undefined: DecodeArgs
```

- [ ] **Step 3: Implement raw decoder helpers**

Create `packages/core/internal/media/decode.go`:

```go
package media

import (
	"io"
	"os/exec"

	"github.com/jass/mojify/packages/core/internal/render"
)

func DecodeArgs(path string, width int, height int) []string {
	return []string{
		"-v", "error",
		"-i", path,
		"-vf", "scale=" + itoa(width) + ":" + itoa(height) + ":force_original_aspect_ratio=decrease",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
}

func StartDecoder(path string, width int, height int) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command("ffmpeg", DecodeArgs(path, width, height)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdout, nil
}

func ReadRawFrame(r io.Reader, width int, height int) (render.RGBFrame, error) {
	size := width * height * 3
	buf := make([]byte, size)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return render.RGBFrame{}, err
	}
	return render.NewRGBFrame(width, height, buf), nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./packages/core/internal/media
go test ./...
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/media
```

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/media/decode.go packages/core/internal/media/decode_test.go
git commit -m "feat: add ffmpeg raw frame decoder"
```

## Task 8: Bounded Playback Scheduler

**Files:**
- Create: `packages/core/internal/player/player.go`
- Create: `packages/core/internal/player/player_test.go`

- [ ] **Step 1: Write scheduler tests**

Create `packages/core/internal/player/player_test.go`:

```go
package player

import (
	"context"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

type fakePresenter struct {
	count int
}

func (p *fakePresenter) Present(render.CharacterFrame) error {
	p.count++
	return nil
}

func TestPlayerPresentsFrames(t *testing.T) {
	frames := make(chan render.CharacterFrame, 3)
	frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: 'A'}}}
	frames <- render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: 'B'}}}
	close(frames)

	presenter := &fakePresenter{}
	err := Play(context.Background(), frames, presenter, 1000)
	if err != nil {
		t.Fatalf("Play returned error: %v", err)
	}
	if presenter.count != 2 {
		t.Fatalf("presented %d frames, want 2", presenter.count)
	}
}

func TestPlayerHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	frames := make(chan render.CharacterFrame)
	cancel()
	start := time.Now()
	err := Play(ctx, frames, &fakePresenter{}, 24)
	if err == nil {
		t.Fatal("Play returned nil error after cancellation")
	}
	if time.Since(start) > time.Second {
		t.Fatal("Play did not return promptly after cancellation")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/player
```

Expected:

```text
FAIL ... undefined: Play
```

- [ ] **Step 3: Implement scheduler**

Create `packages/core/internal/player/player.go`:

```go
package player

import (
	"context"
	"errors"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter interface {
	Present(render.CharacterFrame) error
}

var ErrCancelled = errors.New("playback cancelled")

func Play(ctx context.Context, frames <-chan render.CharacterFrame, presenter Presenter, fps float64) error {
	if fps <= 0 {
		fps = 24
	}
	frameDuration := time.Duration(float64(time.Second) / fps)
	nextDeadline := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ErrCancelled
		case frame, ok := <-frames:
			if !ok {
				return nil
			}
			now := time.Now()
			if now.Before(nextDeadline) {
				timer := time.NewTimer(nextDeadline.Sub(now))
				select {
				case <-ctx.Done():
					timer.Stop()
					return ErrCancelled
				case <-timer.C:
				}
			}
			if err := presenter.Present(frame); err != nil {
				return err
			}
			nextDeadline = nextDeadline.Add(frameDuration)
		}
	}
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./packages/core/internal/player
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/player
```

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/player
git commit -m "feat: add bounded playback scheduler"
```

## Task 9: Terminal Presenter And Minimal Controls

**Files:**
- Create: `packages/core/internal/terminal/presenter.go`
- Create: `packages/core/internal/terminal/input.go`

- [ ] **Step 1: Implement terminal presenter**

Create `packages/core/internal/terminal/presenter.go`:

```go
package terminal

import (
	"fmt"
	"io"

	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter struct {
	Out io.Writer
}

func (p Presenter) Start() error {
	_, err := fmt.Fprint(p.Out, EnterAltScreen, HideCursor, CursorHome, ClearToEnd)
	return err
}

func (p Presenter) Present(frame render.CharacterFrame) error {
	_, err := fmt.Fprint(p.Out, SerializeFrame(frame))
	return err
}

func (p Presenter) Stop() error {
	_, err := fmt.Fprint(p.Out, Reset, ShowCursor, ExitAltScreen)
	return err
}
```

- [ ] **Step 2: Implement input control reader**

Create `packages/core/internal/terminal/input.go`:

```go
package terminal

import (
	"context"
	"io"
)

type Control int

const (
	Quit Control = iota
	TogglePause
)

func ReadControls(ctx context.Context, r io.Reader, out chan<- Control) {
	defer close(out)
	buf := make([]byte, 1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := r.Read(buf)
		if err != nil || n == 0 {
			return
		}
		switch buf[0] {
		case 'q', 'Q':
			out <- Quit
			return
		case ' ':
			out <- TogglePause
		}
	}
}
```

- [ ] **Step 3: Run tests**

Run:

```bash
go test ./packages/core/internal/terminal
go test ./...
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/terminal
```

- [ ] **Step 4: Commit**

```bash
git add packages/core/internal/terminal/presenter.go packages/core/internal/terminal/input.go
git commit -m "feat: add terminal presenter controls"
```

## Task 10: `mojify play` Integration

**Files:**
- Modify: `packages/core/cmd/mojify/main.go`
- Create: `packages/core/internal/cli/play.go`

- [ ] **Step 1: Implement play command orchestration**

Create `packages/core/internal/cli/play.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	xterm "golang.org/x/term"

	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/player"
	"github.com/jass/mojify/packages/core/internal/render"
	"github.com/jass/mojify/packages/core/internal/terminal"
)

func RunPlay(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer) error {
	info, err := media.Probe(inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}

	cols, rows, err := xterm.GetSize(int(stdin.Fd()))
	if err != nil {
		cols, rows = 120, 40
	}
	grid := render.FitGrid(
		render.InputSize{Width: info.Width, Height: info.Height},
		render.TerminalSize{Cols: cols, Rows: rows - 1},
	)

	decodeWidth := min(info.Width, 640)
	decodeHeight := max(1, decodeWidth*info.Height/info.Width)
	cmd, pipe, err := media.StartDecoder(inputPath, decodeWidth, decodeHeight)
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	defer pipe.Close()
	defer cmd.Process.Kill()

	frames := make(chan render.CharacterFrame, 12)
	renderer := render.DefaultRenderer{}
	go func() {
		defer close(frames)
		for {
			rgb, err := media.ReadRawFrame(pipe, decodeWidth, decodeHeight)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case frames <- renderer.Render(rgb, grid):
			}
		}
	}()

	presenter := terminal.Presenter{Out: stdout}
	if err := presenter.Start(); err != nil {
		return err
	}
	defer presenter.Stop()

	return player.Play(ctx, frames, presenter, info.FPS)
}
```

- [ ] **Step 2: Wire play command**

Modify `packages/core/cmd/mojify/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jass/mojify/packages/core/internal/cli"
	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/render"
)
```

Replace the `PlayCommand` case:

```go
	case cli.PlayCommand:
		if err := cli.RunPlay(context.Background(), cmd.InputPath, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "play failed: %v\n", err)
			os.Exit(1)
		}
```

- [ ] **Step 3: Run build and tests**

Run:

```bash
go test ./...
go build -o bin/mojify ./packages/core/cmd/mojify
```

Expected:

```text
all tests pass
bin/mojify exists
```

- [ ] **Step 4: Manual smoke test**

Run with any small local MP4:

```bash
./bin/mojify probe /absolute/path/to/small.mp4
./bin/mojify play /absolute/path/to/small.mp4
```

Expected:

```text
probe prints video metadata
play enters alternate screen and renders moving character frames
Ctrl-C restores cursor and terminal
```

- [ ] **Step 5: Commit**

```bash
git add packages/core/cmd/mojify/main.go packages/core/internal/cli/play.go
git commit -m "feat: play local video in terminal"
```

## Task 11: First CI Gate

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  pull_request:
  push:
    branches:
      - main

jobs:
  quality:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4

      - uses: oven-sh/setup-bun@v2
        with:
          bun-version-file: package.json

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Install FFmpeg
        run: sudo apt-get update && sudo apt-get install -y ffmpeg

      - name: Install JS dependencies
        run: bun install --frozen-lockfile

      - name: Format check
        run: bun run fmt:check

      - name: Test
        run: bun run test

      - name: Build
        run: bun run build
```

- [ ] **Step 2: Run local equivalents**

Run:

```bash
bun run fmt:check
bun run test
bun run build
```

Expected:

```text
all commands pass
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add first quality gate"
```

## Task 12: Milestone README Update

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace README with v1 instructions**

Replace `README.md`:

````markdown
# mojify

Mojify is a terminal-first video player that transforms local video files into colored, edge-aware character frames.

## Status

V1 is source-build only. The first milestone is local visual playback in the terminal.

## Requirements

- Go 1.23+
- Bun 1.3+
- FFmpeg and ffprobe on `PATH`

## Run

```bash
bun install
bun run build
./bin/mojify --help
./bin/mojify probe ./demo.mp4
./bin/mojify play ./demo.mp4
```

## Scope

Included in v1:

- Local video files
- Visual terminal playback
- Truecolor ANSI output
- Edge-aware character rendering
- `play` and `probe` commands

Deferred:

- YouTube/URL input
- Audio
- Export to GIF/MP4/PNG
- npm/npx distribution
- Plugins and custom recipes
````

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: describe v1 source-build usage"
```

## Self-Review Checklist

- Scope coverage:
  - Terminal-first v1: Tasks 2, 10, 12.
  - Local video files only: Tasks 3, 7, 10.
  - Visual-only playback: Tasks 8, 9, 10.
  - Edge-aware truecolor renderer: Tasks 5, 6.
  - Bounded buffering and stable timing: Task 8.
  - Explicit CLI subcommands: Task 2.
  - `play` and `probe`: Tasks 3, 10.
  - Go core in monorepo: Task 1.
  - FFmpeg CLI boundary: Tasks 3, 7.
  - Golden renderer tests first: Task 5.
- Placeholder scan:
  - No `TBD`, `TODO`, or unspecified file paths remain.
- Type consistency:
  - `media.Info`, `render.Grid`, `render.RGBFrame`, `render.CharacterFrame`, `terminal.Presenter`, and `player.Play` are introduced before being consumed.
