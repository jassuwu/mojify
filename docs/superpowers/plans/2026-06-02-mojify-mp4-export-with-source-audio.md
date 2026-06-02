# MP4 Export With Source Audio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `mojify export <input-video> <output.mp4>` to render Mojify visuals into an ordinary MP4 while preserving source audio content when available.

**Architecture:** Keep Mojify's current FFmpeg CLI boundary. Decode local video into raw RGB frames, render Mojify `CharacterFrame`s, rasterize those cells with the bundled `Mx437_IBM_BIOS` font into raw RGB video frames, then stream those frames into an FFmpeg encoder process over stdin. FFmpeg maps source audio optionally and transcodes audio to AAC for MP4 compatibility.

**Tech Stack:** Go 1.23, FFmpeg CLI, ffprobe CLI, Bun/Turbo scripts, `golang.org/x/image/font`, bundled `Mx437_IBM_BIOS.ttf`.

---

## Scope

In scope:

- `mojify export <input-video> <output.mp4>`
- `--width <px>`, `--fps <n>`, `--bitrate <value>`, `--overwrite`
- local file input only
- output path must be `.mp4`
- H.264 video with `yuv420p`
- AAC audio when source audio exists
- silent MP4 when source audio is absent
- dark background with per-cell truecolor glyphs
- bundled `Mx437_IBM_BIOS` export font with explicit attribution
- generated-clip smoke export QA
- optional real `dist/` sample export QA when local samples exist

Out of scope:

- terminal live audio playback
- URL input
- image export
- text export
- GIF/WebM export
- font picker
- live resize
- video zoom/pan
- package distribution or release polish

## File Structure

- Modify `CONTEXT.md`: already records MP4 export, export font, live terminal audio, and product utility expansion language.
- Modify `docs/adr/0021-export-mp4-with-source-audio-before-live-terminal-audio.md`: already records MP4 export before live terminal audio and AAC-compatible audio preservation.
- Modify `docs/adr/0022-use-ffmpeg-cli-for-export-encoding.md`: already records FFmpeg CLI export encoding.
- Modify `docs/adr/0023-use-mx437-ibm-bios-as-default-export-font.md`: already records the default export font.
- Modify `packages/core/internal/cli/cli.go`: parse the `export` subcommand and export flags.
- Modify `packages/core/internal/cli/cli_test.go`: command parser tests for export.
- Create `packages/core/internal/cli/export.go`: CLI-facing export runner.
- Modify `packages/core/cmd/mojify/main.go`: dispatch `ExportCommand`.
- Modify `packages/core/internal/media/decode.go`: add export decoder args with optional FPS filter.
- Modify `packages/core/internal/media/decode_test.go`: export decoder arg tests.
- Create `packages/core/internal/media/encode.go`: FFmpeg rawvideo-to-MP4 encoder args and process launcher.
- Create `packages/core/internal/media/encode_test.go`: encoder arg tests.
- Create `packages/core/internal/exporter/layout.go`: export output dimensions, grid, and FPS resolution.
- Create `packages/core/internal/exporter/layout_test.go`: layout tests.
- Create `packages/core/internal/exporter/raster.go`: cell-level truecolor rasterization.
- Create `packages/core/internal/exporter/raster_test.go`: rasterizer unit tests.
- Create `packages/core/internal/exporter/fonts/fonts.go`: embedded default font loader.
- Add `packages/core/internal/exporter/fonts/Mx437_IBM_BIOS.ttf`: bundled third-party font asset.
- Create `packages/core/internal/exporter/export.go`: streaming export pipeline.
- Create `packages/core/internal/exporter/export_test.go`: focused export preflight or helper tests that do not require long FFmpeg runs.
- Create `docs/qa/export.md`: export QA checklist.
- Create `scripts/export-qa.sh`: generated clip export smoke script.
- Modify `package.json`: add `qa:export`.
- Create or modify `THIRD_PARTY.md`: record bundled font attribution and license.

---

### Task 1: Add Export Command Parsing

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Write failing parser tests**

Add these tests to `packages/core/internal/cli/cli_test.go`:

```go
func TestParseExportCommand(t *testing.T) {
	cmd, err := Parse([]string{"export", "clip.mp4", "out.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != ExportCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, ExportCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if cmd.OutputPath != "out.mp4" {
		t.Fatalf("OutputPath = %q, want out.mp4", cmd.OutputPath)
	}
}

func TestParseExportFlags(t *testing.T) {
	cmd, err := Parse([]string{"export", "--width", "1280", "--fps", "24", "--bitrate", "4M", "--overwrite", "clip.mp4", "out.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Export.Width != 1280 {
		t.Fatalf("Width = %d, want 1280", cmd.Export.Width)
	}
	if cmd.Export.FPS != 24 {
		t.Fatalf("FPS = %f, want 24", cmd.Export.FPS)
	}
	if cmd.Export.Bitrate != "4M" {
		t.Fatalf("Bitrate = %q, want 4M", cmd.Export.Bitrate)
	}
	if !cmd.Export.Overwrite {
		t.Fatal("Overwrite = false, want true")
	}
}

func TestParseExportRejectsMissingOutput(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for missing export output")
	}
}

func TestParseExportRejectsExtraInputs(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mp4", "out.mp4", "extra.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for extra export input")
	}
}

func TestParseExportRejectsProtocolInputs(t *testing.T) {
	for _, input := range []string{
		"https://example.com/demo.mp4",
		"file:///tmp/demo.mp4",
		"pipe:0",
		"concat:part1.mp4|part2.mp4",
		"-",
	} {
		_, err := Parse([]string{"export", input, "out.mp4"})
		if err == nil {
			t.Fatalf("Parse accepted non-local export input %q", input)
		}
	}
}

func TestParseExportRejectsProtocolOutputs(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mp4", "pipe:1"})
	if err == nil {
		t.Fatal("Parse accepted protocol export output")
	}
}

func TestParseExportRejectsNonMP4Output(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mp4", "out.webm"})
	if err == nil {
		t.Fatal("Parse accepted non-MP4 export output")
	}
}

func TestParseExportRejectsInvalidWidth(t *testing.T) {
	_, err := Parse([]string{"export", "--width", "0", "clip.mp4", "out.mp4"})
	if err == nil {
		t.Fatal("Parse accepted invalid export width")
	}
}

func TestParseExportRejectsInvalidFPS(t *testing.T) {
	_, err := Parse([]string{"export", "--fps", "-24", "clip.mp4", "out.mp4"})
	if err == nil {
		t.Fatal("Parse accepted invalid export fps")
	}
}
```

Update `TestHelpTextMentionsCommands` to include:

```go
"mojify export [options] <video> <output.mp4>",
"Export Mojify visuals to an MP4 file",
```

- [ ] **Step 2: Run parser tests and verify they fail**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected: FAIL because `ExportCommand`, `OutputPath`, and `Export` are not defined.

- [ ] **Step 3: Implement export parsing**

In `packages/core/internal/cli/cli.go`, extend the command model:

```go
type CommandKind int

const (
	HelpCommand CommandKind = iota
	PlayCommand
	ProbeCommand
	ExportCommand
)

type ExportOptions struct {
	Width     int
	FPS       float64
	Bitrate   string
	Overwrite bool
}

type Command struct {
	Kind       CommandKind
	InputPath  string
	OutputPath string
	Stats      bool
	Export     ExportOptions
}
```

Add `export` to `Parse`:

```go
case "export":
	return parseExportCommand(args)
```

Add imports:

```go
import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)
```

Add the parser helpers:

```go
func parseExportCommand(args []string) (Command, error) {
	var paths []string
	options := ExportOptions{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--overwrite":
			if options.Overwrite {
				return Command{}, fmt.Errorf("export accepts --overwrite only once")
			}
			options.Overwrite = true
		case "--width":
			value, next, err := parseFlagValue(args, i, "--width")
			if err != nil {
				return Command{}, err
			}
			width, err := strconv.Atoi(value)
			if err != nil || width <= 0 {
				return Command{}, fmt.Errorf("export --width must be a positive integer")
			}
			options.Width = width
			i = next
		case "--fps":
			value, next, err := parseFlagValue(args, i, "--fps")
			if err != nil {
				return Command{}, err
			}
			fps, err := strconv.ParseFloat(value, 64)
			if err != nil || fps <= 0 {
				return Command{}, fmt.Errorf("export --fps must be a positive number")
			}
			options.FPS = fps
			i = next
		case "--bitrate":
			value, next, err := parseFlagValue(args, i, "--bitrate")
			if err != nil {
				return Command{}, err
			}
			if !validBitrate(value) {
				return Command{}, fmt.Errorf("export --bitrate must look like 4000k or 4M")
			}
			options.Bitrate = value
			i = next
		default:
			if strings.HasPrefix(arg, "-") {
				return Command{}, fmt.Errorf("unknown export option %q", arg)
			}
			paths = append(paths, arg)
		}
	}
	if len(paths) != 2 {
		return Command{}, fmt.Errorf("export requires a video input and MP4 output path")
	}
	inputPath := paths[0]
	outputPath := paths[1]
	if hasProtocolInput(inputPath) {
		return Command{}, fmt.Errorf("export accepts local video file paths only")
	}
	if hasProtocolInput(outputPath) {
		return Command{}, fmt.Errorf("export output must be a local MP4 file path")
	}
	if strings.ToLower(filepath.Ext(outputPath)) != ".mp4" {
		return Command{}, fmt.Errorf("export output must end in .mp4")
	}
	return Command{Kind: ExportCommand, InputPath: inputPath, OutputPath: outputPath, Export: options}, nil
}

func parseFlagValue(args []string, index int, flag string) (string, int, error) {
	next := index + 1
	if next >= len(args) || strings.HasPrefix(args[next], "-") {
		return "", index, fmt.Errorf("%s requires a value", flag)
	}
	return args[next], next, nil
}

func validBitrate(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		isLast := i == len(value)-1
		if isLast && (r == 'k' || r == 'K' || r == 'm' || r == 'M') {
			return i > 0
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
```

Update `HelpText`:

```go
Usage:
  mojify play [--stats] <video>              Play a local video file in the terminal
  mojify probe <video>                       Print media and render metadata
  mojify export [options] <video> <output.mp4>  Export Mojify visuals to an MP4 file
  mojify --help                              Show this help

Export options:
  --width <px>       Output MP4 width in pixels
  --fps <n>          Output frame rate, default source FPS
  --bitrate <value>  Video bitrate such as 4000k or 4M
  --overwrite        Replace an existing output file
```

- [ ] **Step 4: Run parser tests and verify they pass**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit parser work**

Run:

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
git commit -m "feat: parse export command"
```

---

### Task 2: Add FFmpeg Export Decoder And Encoder Args

**Files:**
- Modify: `packages/core/internal/media/decode.go`
- Modify: `packages/core/internal/media/decode_test.go`
- Create: `packages/core/internal/media/encode.go`
- Create: `packages/core/internal/media/encode_test.go`

- [ ] **Step 1: Write failing decoder arg tests**

Add these tests to `packages/core/internal/media/decode_test.go`:

```go
func TestExportDecodeArgsWithoutFPS(t *testing.T) {
	args := ExportDecodeArgs("clip.mp4", 160, 90, 0)
	got := strings.Join(args, " ")
	for _, want := range []string{
		"-i clip.mp4",
		"scale=160:90:force_original_aspect_ratio=decrease",
		"pad=160:90:(ow-iw)/2:(oh-ih)/2",
		"-f rawvideo",
		"-pix_fmt rgb24",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("ExportDecodeArgs missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "fps=") {
		t.Fatalf("ExportDecodeArgs included fps filter unexpectedly: %q", got)
	}
}

func TestExportDecodeArgsWithFPS(t *testing.T) {
	args := ExportDecodeArgs("clip.mp4", 160, 90, 12)
	got := strings.Join(args, " ")
	if !strings.Contains(got, "fps=12") {
		t.Fatalf("ExportDecodeArgs missing fps filter in %q", got)
	}
}
```

Ensure `decode_test.go` imports `strings`.

- [ ] **Step 2: Write failing encoder arg tests**

Create `packages/core/internal/media/encode_test.go`:

```go
package media

import (
	"strings"
	"testing"
)

func TestMP4EncodeArgsMapsOptionalAudio(t *testing.T) {
	args, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "clip.webm",
		OutputPath: "out.mp4",
		Width:      1280,
		Height:     720,
		FPS:        24,
		Bitrate:    "4M",
		Overwrite:  false,
	})
	if err != nil {
		t.Fatalf("MP4EncodeArgs returned error: %v", err)
	}
	got := strings.Join(args, " ")
	for _, want := range []string{
		"-n",
		"-f rawvideo",
		"-pix_fmt rgb24",
		"-s 1280x720",
		"-r 24",
		"-i pipe:0",
		"-i clip.webm",
		"-map 0:v:0",
		"-map 1:a?",
		"-c:v libx264",
		"-pix_fmt yuv420p",
		"-b:v 4M",
		"-c:a aac",
		"-shortest",
		"out.mp4",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("MP4EncodeArgs missing %q in %q", want, got)
		}
	}
}

func TestMP4EncodeArgsUsesOverwriteFlag(t *testing.T) {
	args, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "clip.mp4",
		OutputPath: "out.mp4",
		Width:      320,
		Height:     180,
		FPS:        30,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("MP4EncodeArgs returned error: %v", err)
	}
	got := strings.Join(args, " ")
	if !strings.Contains(got, "-y") {
		t.Fatalf("MP4EncodeArgs missing -y in %q", got)
	}
	if strings.Contains(got, "-n") {
		t.Fatalf("MP4EncodeArgs included -n with overwrite in %q", got)
	}
}

func TestMP4EncodeArgsRejectsInvalidDimensions(t *testing.T) {
	_, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "clip.mp4",
		OutputPath: "out.mp4",
		Width:      0,
		Height:     180,
		FPS:        30,
	})
	if err == nil {
		t.Fatal("MP4EncodeArgs returned nil error for invalid dimensions")
	}
}
```

- [ ] **Step 3: Run media tests and verify they fail**

Run:

```bash
go test ./packages/core/internal/media
```

Expected: FAIL because `ExportDecodeArgs`, `MP4EncodeOptions`, and `MP4EncodeArgs` are not defined.

- [ ] **Step 4: Implement export decoder args**

In `packages/core/internal/media/decode.go`, add:

```go
func ExportDecodeArgs(path string, width int, height int, fps float64) []string {
	widthText := strconv.Itoa(width)
	heightText := strconv.Itoa(height)
	filter := "scale=" + widthText + ":" + heightText + ":force_original_aspect_ratio=decrease,pad=" + widthText + ":" + heightText + ":(ow-iw)/2:(oh-ih)/2"
	if fps > 0 {
		filter += ",fps=" + formatFloat(fps)
	}
	return []string{
		"-v", "error",
		"-i", path,
		"-vf", filter,
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
}

func StartExportDecoderContext(ctx context.Context, path string, width int, height int, fps float64) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", ExportDecodeArgs(path, width, height, fps)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdout, nil
}

func formatFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', 3, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	return text
}
```

Add `strings` to the imports in `decode.go`.

- [ ] **Step 5: Implement MP4 encoder args and process launcher**

Create `packages/core/internal/media/encode.go`:

```go
package media

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type MP4EncodeOptions struct {
	InputPath  string
	OutputPath string
	Width      int
	Height     int
	FPS        float64
	Bitrate    string
	Overwrite  bool
}

func MP4EncodeArgs(options MP4EncodeOptions) ([]string, error) {
	if options.InputPath == "" {
		return nil, fmt.Errorf("missing input path")
	}
	if options.OutputPath == "" {
		return nil, fmt.Errorf("missing output path")
	}
	if options.Width <= 0 || options.Height <= 0 {
		return nil, fmt.Errorf("invalid output dimensions %dx%d", options.Width, options.Height)
	}
	if options.FPS <= 0 {
		return nil, fmt.Errorf("invalid export fps %.3f", options.FPS)
	}

	overwriteFlag := "-n"
	if options.Overwrite {
		overwriteFlag = "-y"
	}
	size := fmt.Sprintf("%dx%d", options.Width, options.Height)
	fps := formatFloat(options.FPS)
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		overwriteFlag,
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", size,
		"-r", fps,
		"-i", "pipe:0",
		"-i", options.InputPath,
		"-map", "0:v:0",
		"-map", "1:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
	}
	if options.Bitrate != "" {
		args = append(args, "-b:v", options.Bitrate)
	}
	args = append(args,
		"-c:a", "aac",
		"-shortest",
		options.OutputPath,
	)
	return args, nil
}

func StartMP4EncoderContext(ctx context.Context, options MP4EncodeOptions, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	args, err := MP4EncodeArgs(options)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdin, nil
}
```

- [ ] **Step 6: Run media tests and verify they pass**

Run:

```bash
go test ./packages/core/internal/media
```

Expected: PASS.

- [ ] **Step 7: Run a rawvideo stdin smoke check**

Run:

```bash
dd if=/dev/zero bs=768 count=1 2>/dev/null | ffmpeg -hide_banner -loglevel error -y -f rawvideo -pix_fmt rgb24 -s 16x16 -r 1 -i pipe:0 -i "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm" -map 0:v:0 -map '1:a?' -c:v libx264 -pix_fmt yuv420p -c:a aac -shortest /private/tmp/mojify-rawvideo-audio-smoke.mp4
ffprobe -v error -select_streams a -show_entries stream=codec_name -of csv=p=0 /private/tmp/mojify-rawvideo-audio-smoke.mp4
```

Expected: `aac` when the local real sample exists. If the real sample does not exist, skip this step and use the generated silent QA smoke in Task 6.

- [ ] **Step 8: Commit media FFmpeg args**

Run:

```bash
git add packages/core/internal/media/decode.go packages/core/internal/media/decode_test.go packages/core/internal/media/encode.go packages/core/internal/media/encode_test.go
git commit -m "feat: add mp4 export ffmpeg args"
```

---

### Task 3: Add Export Layout And Rasterizer

**Files:**
- Create: `packages/core/internal/exporter/layout.go`
- Create: `packages/core/internal/exporter/layout_test.go`
- Create: `packages/core/internal/exporter/raster.go`
- Create: `packages/core/internal/exporter/raster_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add font rendering dependency**

Run:

```bash
go get golang.org/x/image/font/opentype
```

Expected: `go.mod` gains `golang.org/x/image`.

- [ ] **Step 2: Write failing layout tests**

Create `packages/core/internal/exporter/layout_test.go`:

```go
package exporter

import "testing"

func TestResolveLayoutDefaultsToCappedSourceWidth(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 1920, Height: 1080, FPS: 24}, Options{})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}
	if layout.OutputWidth != 1280 || layout.OutputHeight != 720 {
		t.Fatalf("output = %dx%d, want 1280x720", layout.OutputWidth, layout.OutputHeight)
	}
	if layout.Grid.Cols != 160 || layout.Grid.Rows != 90 {
		t.Fatalf("grid = %dx%d, want 160x90", layout.Grid.Cols, layout.Grid.Rows)
	}
}

func TestResolveLayoutUsesRequestedWidth(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 320, Height: 180, FPS: 24}, Options{Width: 640})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}
	if layout.OutputWidth != 640 || layout.OutputHeight != 360 {
		t.Fatalf("output = %dx%d, want 640x360", layout.OutputWidth, layout.OutputHeight)
	}
}

func TestResolveLayoutUsesRequestedFPS(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 320, Height: 180, FPS: 60}, Options{FPS: 24})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}
	if layout.FPS != 24 {
		t.Fatalf("FPS = %f, want 24", layout.FPS)
	}
}

func TestResolveLayoutRoundsToCellMultiples(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 1001, Height: 563, FPS: 24}, Options{Width: 1001})
	if err != nil {
		t.Fatalf("ResolveLayout returned error: %v", err)
	}
	if layout.OutputWidth%ExportCellWidth != 0 || layout.OutputHeight%ExportCellHeight != 0 {
		t.Fatalf("output = %dx%d, want multiples of %dx%d", layout.OutputWidth, layout.OutputHeight, ExportCellWidth, ExportCellHeight)
	}
}

func TestResolveLayoutRejectsInvalidInput(t *testing.T) {
	_, err := ResolveLayout(InputInfo{Width: 0, Height: 180, FPS: 24}, Options{})
	if err == nil {
		t.Fatal("ResolveLayout returned nil error for invalid input")
	}
}
```

- [ ] **Step 3: Implement export layout**

Create `packages/core/internal/exporter/layout.go`:

```go
package exporter

import (
	"fmt"
	"math"

	"github.com/jass/mojify/packages/core/internal/render"
)

const ExportCellWidth = 8
const ExportCellHeight = 8
const DefaultExportMaxWidth = 1280

type Options struct {
	Width     int
	FPS       float64
	Bitrate   string
	Overwrite bool
}

type InputInfo struct {
	Width  int
	Height int
	FPS    float64
}

type Layout struct {
	OutputWidth  int
	OutputHeight int
	Grid         render.Grid
	FPS          float64
}

func ResolveLayout(input InputInfo, options Options) (Layout, error) {
	if input.Width <= 0 || input.Height <= 0 {
		return Layout{}, fmt.Errorf("invalid input dimensions %dx%d", input.Width, input.Height)
	}
	fps := input.FPS
	if options.FPS > 0 {
		fps = options.FPS
	}
	if fps <= 0 {
		fps = 24
	}

	width := options.Width
	if width == 0 {
		width = min(input.Width, DefaultExportMaxWidth)
	}
	if width <= 0 {
		return Layout{}, fmt.Errorf("invalid export width %d", width)
	}
	width = alignUp(width, ExportCellWidth)
	aspect := float64(input.Height) / float64(input.Width)
	height := int(math.Round(float64(width) * aspect))
	height = max(ExportCellHeight, alignUp(height, ExportCellHeight))

	return Layout{
		OutputWidth:  width,
		OutputHeight: height,
		Grid:         render.Grid{Cols: width / ExportCellWidth, Rows: height / ExportCellHeight},
		FPS:          fps,
	}, nil
}

func alignUp(value int, multiple int) int {
	if value%multiple == 0 {
		return value
	}
	return value + multiple - value%multiple
}
```

- [ ] **Step 4: Write failing rasterizer tests**

Create `packages/core/internal/exporter/raster_test.go`:

```go
package exporter

import (
	"testing"

	"golang.org/x/image/font/basicfont"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestRasterizerDimensions(t *testing.T) {
	rasterizer := Rasterizer{Face: basicfont.Face7x13, CellWidth: 8, CellHeight: 16, Background: DarkBackground}
	frame := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: '@', R: 255, G: 0, B: 0},
			{Ch: '#', R: 0, G: 255, B: 0},
		},
	}
	rgb, width, height, err := rasterizer.RasterizeRGB(frame)
	if err != nil {
		t.Fatalf("RasterizeRGB returned error: %v", err)
	}
	if width != 16 || height != 16 {
		t.Fatalf("size = %dx%d, want 16x16", width, height)
	}
	if len(rgb) != width*height*3 {
		t.Fatalf("rgb length = %d, want %d", len(rgb), width*height*3)
	}
}

func TestRasterizerUsesDarkBackground(t *testing.T) {
	rasterizer := Rasterizer{Face: basicfont.Face7x13, CellWidth: 8, CellHeight: 16, Background: DarkBackground}
	frame := render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: ' ', R: 255, G: 255, B: 255}}}
	rgb, _, _, err := rasterizer.RasterizeRGB(frame)
	if err != nil {
		t.Fatalf("RasterizeRGB returned error: %v", err)
	}
	if rgb[0] != DarkBackground.R || rgb[1] != DarkBackground.G || rgb[2] != DarkBackground.B {
		t.Fatalf("first pixel = (%d,%d,%d), want dark background", rgb[0], rgb[1], rgb[2])
	}
}

func TestRasterizerDrawsPerCellColor(t *testing.T) {
	rasterizer := Rasterizer{Face: basicfont.Face7x13, CellWidth: 8, CellHeight: 16, Background: DarkBackground}
	frame := render.CharacterFrame{Width: 1, Height: 1, Cells: []render.Cell{{Ch: '@', R: 255, G: 0, B: 0}}}
	rgb, width, _, err := rasterizer.RasterizeRGB(frame)
	if err != nil {
		t.Fatalf("RasterizeRGB returned error: %v", err)
	}
	foundRed := false
	for i := 0; i < len(rgb); i += 3 {
		if rgb[i] > 200 && rgb[i+1] < 50 && rgb[i+2] < 50 {
			foundRed = true
			break
		}
	}
	if !foundRed {
		t.Fatalf("did not find red glyph pixels in %d-wide raster", width)
	}
}

func TestRasterizerRejectsInvalidFrameShape(t *testing.T) {
	rasterizer := Rasterizer{Face: basicfont.Face7x13, CellWidth: 8, CellHeight: 16, Background: DarkBackground}
	_, _, _, err := rasterizer.RasterizeRGB(render.CharacterFrame{Width: 2, Height: 1, Cells: []render.Cell{{Ch: '@'}}})
	if err == nil {
		t.Fatal("RasterizeRGB returned nil error for invalid frame shape")
	}
}
```

- [ ] **Step 5: Implement rasterizer**

Create `packages/core/internal/exporter/raster.go`:

```go
package exporter

import (
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/jass/mojify/packages/core/internal/render"
)

var DarkBackground = color.RGBA{R: 5, G: 5, B: 8, A: 255}

type Rasterizer struct {
	Face       font.Face
	CellWidth  int
	CellHeight int
	Background color.RGBA
}

func (r Rasterizer) RasterizeRGB(frame render.CharacterFrame) ([]byte, int, int, error) {
	if r.Face == nil {
		return nil, 0, 0, fmt.Errorf("missing export font face")
	}
	if r.CellWidth <= 0 || r.CellHeight <= 0 {
		return nil, 0, 0, fmt.Errorf("invalid cell size %dx%d", r.CellWidth, r.CellHeight)
	}
	if frame.Width <= 0 || frame.Height <= 0 {
		return nil, 0, 0, fmt.Errorf("invalid frame dimensions %dx%d", frame.Width, frame.Height)
	}
	if len(frame.Cells) != frame.Width*frame.Height {
		return nil, 0, 0, fmt.Errorf("invalid frame shape: %d cells for %dx%d", len(frame.Cells), frame.Width, frame.Height)
	}

	width := frame.Width * r.CellWidth
	height := frame.Height * r.CellHeight
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{C: r.Background}, image.Point{}, imagedraw.Src)

	metrics := r.Face.Metrics()
	ascent := metrics.Ascent
	drawer := font.Drawer{Dst: img, Face: r.Face}
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			cell := frame.Cells[y*frame.Width+x]
			drawer.Src = image.NewUniform(color.RGBA{R: cell.R, G: cell.G, B: cell.B, A: 255})
			drawer.Dot = fixed.Point26_6{
				X: fixed.I(x * r.CellWidth),
				Y: fixed.I(y*r.CellHeight) + ascent,
			}
			drawer.DrawString(string(cell.Ch))
		}
	}

	rgb := make([]byte, width*height*3)
	offset := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.RGBAAt(x, y)
			rgb[offset] = pixel.R
			rgb[offset+1] = pixel.G
			rgb[offset+2] = pixel.B
			offset += 3
		}
	}
	return rgb, width, height, nil
}
```

- [ ] **Step 6: Run exporter unit tests**

Run:

```bash
go test ./packages/core/internal/exporter
```

Expected: PASS.

- [ ] **Step 7: Commit layout and rasterizer**

Run:

```bash
git add go.mod go.sum packages/core/internal/exporter/layout.go packages/core/internal/exporter/layout_test.go packages/core/internal/exporter/raster.go packages/core/internal/exporter/raster_test.go
git commit -m "feat: add export rasterizer"
```

---

### Task 4: Bundle And Load Mx437 IBM BIOS Font

**Files:**
- Create: `packages/core/internal/exporter/fonts/Mx437_IBM_BIOS.ttf`
- Create: `packages/core/internal/exporter/fonts/fonts.go`
- Create: `packages/core/internal/exporter/fonts/fonts_test.go`
- Create: `THIRD_PARTY.md`

- [ ] **Step 1: Download and copy the font asset**

Run:

```bash
mkdir -p /private/tmp/mojify-fonts
curl -L -o /private/tmp/mojify-fonts/oldschool_pc_font_pack_v2.2_FULL.zip "https://int10h.org/oldschool-pc-fonts/download/oldschool_pc_font_pack_v2.2_FULL.zip"
unzip -j /private/tmp/mojify-fonts/oldschool_pc_font_pack_v2.2_FULL.zip "*Mx437_IBM_BIOS.ttf" -d /private/tmp/mojify-fonts
mkdir -p packages/core/internal/exporter/fonts
cp /private/tmp/mojify-fonts/Mx437_IBM_BIOS.ttf packages/core/internal/exporter/fonts/Mx437_IBM_BIOS.ttf
```

Expected: `packages/core/internal/exporter/fonts/Mx437_IBM_BIOS.ttf` exists.

- [ ] **Step 2: Write font loader test**

Create `packages/core/internal/exporter/fonts/fonts_test.go`:

```go
package fonts

import "testing"

func TestDefaultFaceLoads(t *testing.T) {
	face, err := DefaultFace()
	if err != nil {
		t.Fatalf("DefaultFace returned error: %v", err)
	}
	if face == nil {
		t.Fatal("DefaultFace returned nil face")
	}
}
```

- [ ] **Step 3: Implement embedded font loader**

Create `packages/core/internal/exporter/fonts/fonts.go`:

```go
package fonts

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed Mx437_IBM_BIOS.ttf
var mx437IBMBIOS []byte

func DefaultFace() (font.Face, error) {
	parsed, err := opentype.Parse(mx437IBMBIOS)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    8,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
```

- [ ] **Step 4: Add third-party attribution**

Create `THIRD_PARTY.md`:

```md
# Third-Party Assets

## Mx437 IBM BIOS

Mojify bundles `Mx437_IBM_BIOS.ttf` from The Ultimate Oldschool PC Font Pack by VileR.

- Source: https://int10h.org/oldschool-pc-fonts/
- Download: https://int10h.org/oldschool-pc-fonts/download/
- License: Creative Commons Attribution-ShareAlike 4.0 International
- License URL: https://creativecommons.org/licenses/by-sa/4.0/

The font is used as Mojify's default export font for rasterizing terminal-style character frames into MP4 video frames.
```

- [ ] **Step 5: Run font tests**

Run:

```bash
go test ./packages/core/internal/exporter/fonts
```

Expected: PASS.

- [ ] **Step 6: Commit font asset and attribution**

Run:

```bash
git add packages/core/internal/exporter/fonts THIRD_PARTY.md
git commit -m "feat: bundle export font"
```

---

### Task 5: Implement Streaming MP4 Export Pipeline

**Files:**
- Create: `packages/core/internal/exporter/export.go`
- Create: `packages/core/internal/exporter/export_test.go`
- Create: `packages/core/internal/cli/export.go`

- [ ] **Step 1: Write export preflight tests**

Create `packages/core/internal/exporter/export_test.go`:

```go
package exporter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOutputPathRejectsExistingWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	err := checkOutputPath(output, Options{})
	if err == nil {
		t.Fatal("checkOutputPath returned nil error for existing output")
	}
}

func TestCheckOutputPathAllowsOverwrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	err := checkOutputPath(output, Options{Overwrite: true})
	if err != nil {
		t.Fatalf("checkOutputPath returned error: %v", err)
	}
}
```

- [ ] **Step 2: Implement exporter pipeline**

Create `packages/core/internal/exporter/export.go`:

```go
package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jass/mojify/packages/core/internal/exporter/fonts"
	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/render"
)

func ExportMP4(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) error {
	if err := checkOutputPath(outputPath, options); err != nil {
		return err
	}
	info, err := media.ProbeContext(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	layout, err := ResolveLayout(InputInfo{Width: info.Width, Height: info.Height, FPS: info.FPS}, options)
	if err != nil {
		return err
	}
	if stderr != nil {
		fmt.Fprintf(stderr, "export: %s -> %s\n", inputPath, outputPath)
		fmt.Fprintf(stderr, "output: %dx%d @ %.3f fps\n", layout.OutputWidth, layout.OutputHeight, layout.FPS)
	}

	decodeCmd, decodePipe, err := media.StartExportDecoderContext(ctx, inputPath, layout.Grid.Cols, layout.Grid.Rows, options.FPS)
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	decodeCleaned := false
	defer func() {
		if !decodeCleaned {
			_ = cleanupProcess(decodeCmd, decodePipe)
		}
	}()

	encodeCmd, encodePipe, err := media.StartMP4EncoderContext(ctx, media.MP4EncodeOptions{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Width:      layout.OutputWidth,
		Height:     layout.OutputHeight,
		FPS:        layout.FPS,
		Bitrate:    options.Bitrate,
		Overwrite:  options.Overwrite,
	}, stderr)
	if err != nil {
		return fmt.Errorf("start encoder: %w", err)
	}
	encodeClosed := false
	defer func() {
		if !encodeClosed {
			_ = encodePipe.Close()
			_ = encodeCmd.Wait()
		}
	}()

	face, err := fonts.DefaultFace()
	if err != nil {
		return fmt.Errorf("load export font: %w", err)
	}
	rasterizer := Rasterizer{Face: face, CellWidth: ExportCellWidth, CellHeight: ExportCellHeight, Background: DarkBackground}
	renderer := render.DefaultRenderer{}

	for {
		rgb, err := media.ReadRawFrame(decodePipe, layout.Grid.Cols, layout.Grid.Rows)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return fmt.Errorf("read decoded frame: %w", err)
		}
		charFrame := renderer.Render(rgb, layout.Grid)
		raw, width, height, err := rasterizer.RasterizeRGB(charFrame)
		if err != nil {
			return fmt.Errorf("rasterize frame: %w", err)
		}
		if width != layout.OutputWidth || height != layout.OutputHeight {
			return fmt.Errorf("raster frame size %dx%d did not match layout %dx%d", width, height, layout.OutputWidth, layout.OutputHeight)
		}
		if _, err := encodePipe.Write(raw); err != nil {
			return fmt.Errorf("write encoder frame: %w", err)
		}
	}

	if err := decodePipe.Close(); err != nil {
		return fmt.Errorf("close decoder pipe: %w", err)
	}
	decodeErr := decodeCmd.Wait()
	decodeCleaned = true

	if err := encodePipe.Close(); err != nil {
		return fmt.Errorf("close encoder pipe: %w", err)
	}
	encodeClosed = true
	encodeErr := encodeCmd.Wait()

	if decodeErr != nil {
		return fmt.Errorf("decoder failed: %w", decodeErr)
	}
	if encodeErr != nil {
		return fmt.Errorf("encoder failed: %w", encodeErr)
	}
	if stderr != nil {
		fmt.Fprintf(stderr, "export complete: %s\n", outputPath)
	}
	return nil
}

func checkOutputPath(outputPath string, options Options) error {
	if outputPath == "" {
		return fmt.Errorf("missing output path")
	}
	_, err := os.Stat(outputPath)
	if err == nil && !options.Overwrite {
		return fmt.Errorf("output exists: %s; pass --overwrite to replace it", outputPath)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat output: %w", err)
	}
	return nil
}

func cleanupProcess(cmd *exec.Cmd, pipe io.Closer) error {
	_ = pipe.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return cmd.Wait()
}
```

- [ ] **Step 3: Implement CLI export runner**

Create `packages/core/internal/cli/export.go`:

```go
package cli

import (
	"context"
	"io"

	"github.com/jass/mojify/packages/core/internal/exporter"
)

func RunExport(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions) error {
	return exporter.ExportMP4(ctx, inputPath, outputPath, stderr, exporter.Options{
		Width:     options.Width,
		FPS:       options.FPS,
		Bitrate:   options.Bitrate,
		Overwrite: options.Overwrite,
	})
}
```

- [ ] **Step 4: Run exporter and CLI tests**

Run:

```bash
go test ./packages/core/internal/exporter ./packages/core/internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit export pipeline**

Run:

```bash
git add packages/core/internal/exporter packages/core/internal/cli/export.go
git commit -m "feat: stream mp4 export"
```

---

### Task 6: Wire Main Command, QA Script, And Docs

**Files:**
- Modify: `packages/core/cmd/mojify/main.go`
- Create: `docs/qa/export.md`
- Create: `scripts/export-qa.sh`
- Modify: `package.json`
- Modify: `README.md`

- [ ] **Step 1: Wire main dispatch**

In `packages/core/cmd/mojify/main.go`, add a dispatch branch:

```go
case cli.ExportCommand:
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.RunExport(
		ctx,
		cmd.InputPath,
		cmd.OutputPath,
		os.Stderr,
		cmd.Export,
	); err != nil {
		fmt.Fprintf(os.Stderr, "export failed: %v\n", err)
		os.Exit(1)
	}
```

- [ ] **Step 2: Build and verify help output**

Run:

```bash
bun run build
./bin/mojify --help
```

Expected: help includes `mojify export [options] <video> <output.mp4>`.

- [ ] **Step 3: Create export QA script**

Create `scripts/export-qa.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist/qa/export

./bin/mojify export --overwrite --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
ffprobe -v error -select_streams v -show_entries stream=codec_name,width,height -of csv=p=0 dist/qa/export/low-motion-bars-export.mp4

if [ -f "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm" ]; then
  ./bin/mojify export --overwrite --width 320 "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm" dist/qa/export/real-audio-export.mp4
  ffprobe -v error -select_streams a -show_entries stream=codec_name -of csv=p=0 dist/qa/export/real-audio-export.mp4
fi

printf 'Export QA complete.\n'
```

Run:

```bash
chmod +x scripts/export-qa.sh
```

- [ ] **Step 4: Add Bun script**

In root `package.json`, add:

```json
"qa:export": "bash scripts/export-qa.sh"
```

- [ ] **Step 5: Create export QA docs**

Create `docs/qa/export.md`:

```md
# Export QA

## Canonical Smoke

Run:

```bash
bun run qa:clips
bun run build
bun run qa:export
```

Expected:

- `dist/qa/export/low-motion-bars-export.mp4` exists.
- `ffprobe` reports an H.264 video stream.
- The generated silent clip exports successfully without an audio stream.

## Optional Real Sample

If local real samples exist in `dist/`, `bun run qa:export` also exports a short real sample path and checks for an audio stream in the output.

Expected:

- `dist/qa/export/real-audio-export.mp4` exists when the real sample exists.
- `ffprobe` reports an AAC audio stream.
- The exported video shows Mojify glyphs on a dark background.
```

- [ ] **Step 6: Update README command list**

In `README.md`, add export usage near the existing play/probe examples:

```md
./bin/mojify export ./demo.mp4 ./demo-mojify.mp4
./bin/mojify export --width 1280 --bitrate 4M ./demo.mp4 ./demo-mojify.mp4
```

Update the included feature list to mention:

```md
- MP4 export with Mojify visuals and source audio content when available
```

- [ ] **Step 7: Run QA script**

Run:

```bash
bun run qa:clips
bun run build
bun run qa:export
```

Expected:

- `bun run qa:clips` generates three clips.
- `bun run build` creates `bin/mojify`.
- `bun run qa:export` prints `Export QA complete.`
- The generated export file exists.
- If the real sample exists, the optional audio export reports `aac`.

- [ ] **Step 8: Commit wiring and QA docs**

Run:

```bash
git add packages/core/cmd/mojify/main.go docs/qa/export.md scripts/export-qa.sh package.json README.md
git commit -m "feat: add export command qa"
```

---

### Task 7: Final Verification

**Files:**
- All changed files.

- [ ] **Step 1: Format**

Run:

```bash
bun run fmt
```

Expected: command exits 0.

- [ ] **Step 2: Verify formatting**

Run:

```bash
bun run fmt:check
```

Expected: command exits 0.

- [ ] **Step 3: Run full tests**

Run:

```bash
bun run test
```

Expected: command exits 0.

- [ ] **Step 4: Run typecheck**

Run:

```bash
bun run typecheck
```

Expected: command exits 0.

- [ ] **Step 5: Run build**

Run:

```bash
bun run build
```

Expected: command exits 0 and `bin/mojify` exists.

- [ ] **Step 6: Check module tidy**

Run:

```bash
go mod tidy -diff
```

Expected: no diff.

- [ ] **Step 7: Run core race tests**

Run:

```bash
go test -race ./packages/core/internal/... ./packages/core/cmd/...
```

Expected: command exits 0.

- [ ] **Step 8: Run playback QA clips**

Run:

```bash
bun run qa:clips
```

Expected: command exits 0 and generated clips exist in `dist/qa/`.

- [ ] **Step 9: Run export QA**

Run:

```bash
bun run qa:export
```

Expected: command exits 0 and writes generated export output.

- [ ] **Step 10: Inspect exported streams**

Run:

```bash
ffprobe -v error -show_streams -print_format json dist/qa/export/low-motion-bars-export.mp4
```

Expected: JSON contains one video stream. It may contain no audio stream because the generated clip is silent.

If `dist/qa/export/real-audio-export.mp4` exists, run:

```bash
ffprobe -v error -select_streams a -show_entries stream=codec_name -of csv=p=0 dist/qa/export/real-audio-export.mp4
```

Expected: `aac`.

- [ ] **Step 11: Manual visual QA**

Open the generated export in a video player.

Expected:

- The output is an ordinary MP4.
- The video shows Mojify glyphs on a dark background.
- The generated silent export has no audio-related failure.
- The optional real sample export has audible source audio content.

- [ ] **Step 12: Commit final verification docs if changed**

If verification changed docs or scripts, run:

```bash
git add docs scripts README.md package.json
git commit -m "docs: describe mp4 export qa"
```

Skip this commit if no files changed after the previous commits.

---

## Self-Review

Spec coverage:

- Export command: Task 1 and Task 6.
- Local file input only: Task 1.
- MP4 output path: Task 1.
- FFmpeg CLI raw RGB stdin: Task 2 and Task 5.
- Optional audio mapping: Task 2.
- AAC-compatible audio preservation: Task 2 and Task 6.
- Silent input success: Task 6 and Task 7.
- Bundled `Mx437_IBM_BIOS`: Task 4.
- Cell-level truecolor rasterization: Task 3.
- Generated QA and optional real sample QA: Task 6 and Task 7.
- No distribution, URL input, live terminal audio, live resize, or video zoom: Scope section.

Placeholder scan:

- The plan contains no `TBD`, no implementation gaps, and no unspecified test expectations.

Type consistency:

- `cli.ExportOptions` maps into `exporter.Options`.
- `exporter.Layout` supplies dimensions to `media.MP4EncodeOptions`.
- `media.ExportDecodeArgs` emits frame data at `layout.Grid` dimensions.
- `exporter.Rasterizer` emits raw RGB frames at `layout.OutputWidth` x `layout.OutputHeight`.
