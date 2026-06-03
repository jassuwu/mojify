# Curated Multi-Format Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `mojify export SOURCE OUTPUT` from MP4-only output to a curated multi-format export surface for video, animated, still-image, and single-frame text outputs.

**Architecture:** Keep extension-routed export: the output path extension selects a curated format family. Reuse existing source resolution, FFmpeg decoding, layout resolution, renderer, export rasterizer, progress reporter, metrics, and ordered worker pipeline. Use FFmpeg for media/image container encoding and renderer-only serializers for `.txt` and `.ansi`.

**Tech Stack:** Go 1.23, FFmpeg CLI, ffprobe CLI, yt-dlp CLI, existing Mojify renderer/rasterizer/exporter packages, Bun/Turbo QA scripts.

---

## Decisions Already Made

- Keep one command: `mojify export SOURCE OUTPUT`.
- Route by output extension.
- Supported in this stage:
  - video: `.mp4`, `.webm`, `.mov`
  - animated visual: `.gif`, `.apng`
  - still visual: `.png`, `.jpg`, `.jpeg`
  - still text: `.txt`, `.ansi`
- Deferred:
  - `.webp`
  - animated text streams
  - exact source-frame selection
  - arbitrary "any FFmpeg output" support
- `--at <timestamp>` is timestamp-based and valid for every export format.
- `--duration <duration>` is valid for time-based outputs: `.mp4`, `.webm`, `.mov`, `.gif`, `.apng`.
- `--duration` is invalid for single-frame outputs: `.png`, `.jpg`, `.jpeg`, `.txt`, `.ansi`.
- `.txt` writes a plain single `CharacterFrame`.
- `.ansi` writes a colored single `CharacterFrame`.
- For media/image outputs, `--width` means output pixels.
- For text outputs, `--width` means character columns.
- `.mp4` keeps the existing source-audio behavior.
- `.webm` and `.mov` should include source audio when available.
- `.gif`, `.apng`, `.png`, `.jpg`, `.jpeg`, `.txt`, and `.ansi` are visual/text-only.
- GIF/APNG/still image output should use FFmpeg where practical.

## File Structure

- Modify `CONTEXT.md`
  - Add glossary terms for curated multi-format export, timestamp export selection, still export, animated export, text export, and WebP deferral.
- Modify `packages/core/internal/cli/cli.go`
  - Parse `--at` and `--duration`.
  - Accept curated output extensions instead of `.mp4` only.
  - Update help text.
- Modify `packages/core/internal/cli/cli_test.go`
  - Lock supported extensions, time flag parsing, invalid combinations, and help text.
- Modify `packages/core/internal/cli/export.go`
  - Pass new export options through.
  - Change the injected runner from `ExportMP4` to a general `Export` function while keeping tests focused.
- Modify `packages/core/internal/cli/export_test.go`
  - Update URL handoff tests for generalized export.
- Create `packages/core/internal/exporter/format.go`
  - Own output extension registry and format-family metadata.
- Create `packages/core/internal/exporter/format_test.go`
  - Lock supported formats and family behavior.
- Modify `packages/core/internal/exporter/layout.go`
  - Add timestamp/duration options and text-width handling.
- Modify `packages/core/internal/exporter/layout_test.go`
  - Lock text width as character columns and media/image width as pixels.
- Modify `packages/core/internal/media/decode.go`
  - Add timestamp and duration-aware export decoder args.
- Modify `packages/core/internal/media/decode_test.go`
  - Lock FFmpeg `-ss` and `-t` placement.
- Modify `packages/core/internal/media/encode.go`
  - Add curated raw-video encoder args for `.webm`, `.mov`, `.gif`, `.apng`, `.png`, `.jpg`, and `.jpeg`.
  - Keep MP4 behavior unchanged through compatibility wrappers.
- Modify `packages/core/internal/media/encode_test.go`
  - Lock encoder args per supported media/image format.
- Modify `packages/core/internal/exporter/export.go`
  - Add generalized `Export`.
  - Keep `ExportMP4` as a compatibility wrapper.
  - Route to time-based media, still image, or text output.
- Create `packages/core/internal/exporter/text.go`
  - Serialize `.txt` and `.ansi` single-frame outputs.
- Create `packages/core/internal/exporter/text_test.go`
  - Lock plain text and ANSI output.
- Modify `packages/core/internal/exporter/pipeline.go`
  - Reuse existing ordered frame pipeline for time-based media encoders.
  - Add a small single-frame decode/render helper if needed.
- Modify `packages/core/internal/exporter/pipeline_test.go`
  - Lock single-frame helper behavior if added.
- Modify `packages/core/internal/exporter/progress.go`
  - Replace hardcoded `video` wording with format-family-aware status text.
- Modify `packages/core/internal/exporter/progress_test.go`
  - Lock progress text for video, animated, image, and text exports.
- Modify `docs/qa/export.md`
  - Add curated multi-format export QA.
- Modify `scripts/export-qa.sh`
  - Smoke-test representative formats.
- Modify `README.md`
  - Update current capabilities and examples after implementation.

---

### Task 1: Add Curated Output Format Registry

**Files:**
- Create: `packages/core/internal/exporter/format.go`
- Create: `packages/core/internal/exporter/format_test.go`

- [ ] **Step 1: Write failing format registry tests**

Create `packages/core/internal/exporter/format_test.go`:

```go
package exporter

import "testing"

func TestResolveOutputFormatSupportsCuratedExtensions(t *testing.T) {
	tests := []struct {
		path   string
		ext    string
		family OutputFamily
	}{
		{"out.mp4", ".mp4", OutputFamilyVideo},
		{"out.webm", ".webm", OutputFamilyVideo},
		{"out.mov", ".mov", OutputFamilyVideo},
		{"out.gif", ".gif", OutputFamilyAnimated},
		{"out.apng", ".apng", OutputFamilyAnimated},
		{"out.png", ".png", OutputFamilyStillImage},
		{"out.jpg", ".jpg", OutputFamilyStillImage},
		{"out.jpeg", ".jpeg", OutputFamilyStillImage},
		{"out.txt", ".txt", OutputFamilyText},
		{"out.ansi", ".ansi", OutputFamilyText},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			format, err := ResolveOutputFormat(tc.path)
			if err != nil {
				t.Fatalf("ResolveOutputFormat returned error: %v", err)
			}
			if format.Extension != tc.ext {
				t.Fatalf("Extension = %q, want %q", format.Extension, tc.ext)
			}
			if format.Family != tc.family {
				t.Fatalf("Family = %q, want %q", format.Family, tc.family)
			}
		})
	}
}

func TestResolveOutputFormatRejectsUnsupportedExtensions(t *testing.T) {
	for _, path := range []string{"out.webp", "out.bmp", "out", ".gitignore"} {
		_, err := ResolveOutputFormat(path)
		if err == nil {
			t.Fatalf("ResolveOutputFormat(%q) returned nil error", path)
		}
	}
}

func TestOutputFormatCapabilities(t *testing.T) {
	tests := []struct {
		path          string
		timeBased     bool
		singleFrame   bool
		supportsAudio bool
		text          bool
	}{
		{"out.mp4", true, false, true, false},
		{"out.webm", true, false, true, false},
		{"out.mov", true, false, true, false},
		{"out.gif", true, false, false, false},
		{"out.apng", true, false, false, false},
		{"out.png", false, true, false, false},
		{"out.jpg", false, true, false, false},
		{"out.txt", false, true, false, true},
		{"out.ansi", false, true, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			format, err := ResolveOutputFormat(tc.path)
			if err != nil {
				t.Fatalf("ResolveOutputFormat returned error: %v", err)
			}
			if format.TimeBased != tc.timeBased {
				t.Fatalf("TimeBased = %v, want %v", format.TimeBased, tc.timeBased)
			}
			if format.SingleFrame != tc.singleFrame {
				t.Fatalf("SingleFrame = %v, want %v", format.SingleFrame, tc.singleFrame)
			}
			if format.SupportsAudio != tc.supportsAudio {
				t.Fatalf("SupportsAudio = %v, want %v", format.SupportsAudio, tc.supportsAudio)
			}
			if format.Text != tc.text {
				t.Fatalf("Text = %v, want %v", format.Text, tc.text)
			}
		})
	}
}

func TestSupportedOutputExtensionsText(t *testing.T) {
	got := SupportedOutputExtensionsText()
	want := ".mp4, .webm, .mov, .gif, .apng, .png, .jpg, .jpeg, .txt, .ansi"
	if got != want {
		t.Fatalf("SupportedOutputExtensionsText() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail with undefined `ResolveOutputFormat`, `OutputFamily`, and `SupportedOutputExtensionsText`.

- [ ] **Step 3: Implement the registry**

Create `packages/core/internal/exporter/format.go`:

```go
package exporter

import (
	"fmt"
	"path/filepath"
	"strings"
)

type OutputFamily string

const (
	OutputFamilyVideo      OutputFamily = "video"
	OutputFamilyAnimated   OutputFamily = "animated"
	OutputFamilyStillImage OutputFamily = "still image"
	OutputFamilyText       OutputFamily = "text"
)

type OutputFormat struct {
	Extension     string
	Family        OutputFamily
	TimeBased     bool
	SingleFrame   bool
	SupportsAudio bool
	Text          bool
}

var supportedOutputFormats = []OutputFormat{
	{Extension: ".mp4", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true},
	{Extension: ".webm", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true},
	{Extension: ".mov", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true},
	{Extension: ".gif", Family: OutputFamilyAnimated, TimeBased: true},
	{Extension: ".apng", Family: OutputFamilyAnimated, TimeBased: true},
	{Extension: ".png", Family: OutputFamilyStillImage, SingleFrame: true},
	{Extension: ".jpg", Family: OutputFamilyStillImage, SingleFrame: true},
	{Extension: ".jpeg", Family: OutputFamilyStillImage, SingleFrame: true},
	{Extension: ".txt", Family: OutputFamilyText, SingleFrame: true, Text: true},
	{Extension: ".ansi", Family: OutputFamilyText, SingleFrame: true, Text: true},
}

func ResolveOutputFormat(outputPath string) (OutputFormat, error) {
	ext := strings.ToLower(filepath.Ext(outputPath))
	for _, format := range supportedOutputFormats {
		if ext == format.Extension {
			return format, nil
		}
	}
	return OutputFormat{}, fmt.Errorf("unsupported export output extension %q; supported extensions: %s", ext, SupportedOutputExtensionsText())
}

func SupportedOutputExtensionsText() string {
	extensions := make([]string, 0, len(supportedOutputFormats))
	for _, format := range supportedOutputFormats {
		extensions = append(extensions, format.Extension)
	}
	return strings.Join(extensions, ", ")
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/format.go packages/core/internal/exporter/format_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/exporter/format.go packages/core/internal/exporter/format_test.go
git commit --no-gpg-sign -m "feat: add export format registry"
```

---

### Task 2: Parse Export Format, `--at`, and `--duration`

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Add failing CLI tests**

Append to `packages/core/internal/cli/cli_test.go`:

```go
func TestParseExportAcceptsCuratedOutputExtensions(t *testing.T) {
	for _, output := range []string{
		"out.mp4", "out.webm", "out.mov", "out.gif", "out.apng",
		"out.png", "out.jpg", "out.jpeg", "out.txt", "out.ansi",
	} {
		cmd, err := Parse([]string{"export", "clip.mov", output})
		if err != nil {
			t.Fatalf("Parse accepted output %q? err = %v", output, err)
		}
		if cmd.OutputPath != output {
			t.Fatalf("OutputPath = %q, want %q", cmd.OutputPath, output)
		}
	}
}

func TestParseExportRejectsUnsupportedOutputExtension(t *testing.T) {
	for _, output := range []string{"out.webp", "out.bmp", "out"} {
		_, err := Parse([]string{"export", "clip.mov", output})
		if err == nil {
			t.Fatalf("Parse returned nil error for unsupported output %q", output)
		}
	}
}

func TestParseExportAtAndDuration(t *testing.T) {
	cmd, err := Parse([]string{
		"export",
		"--at", "01:02:03.250",
		"--duration", "3.5s",
		"clip.mov",
		"out.gif",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !cmd.Export.HasAt || cmd.Export.AtSeconds != 3723.25 {
		t.Fatalf("At = (%v, %v), want (true, 3723.25)", cmd.Export.HasAt, cmd.Export.AtSeconds)
	}
	if !cmd.Export.HasDuration || cmd.Export.DurationSeconds != 3.5 {
		t.Fatalf("Duration = (%v, %v), want (true, 3.5)", cmd.Export.HasDuration, cmd.Export.DurationSeconds)
	}
}

func TestParseExportAcceptsTimeValueShapes(t *testing.T) {
	tests := map[string]float64{
		"10":           10,
		"10s":          10,
		"90.5s":        90.5,
		"1:23":         83,
		"01:02:03.25":  3723.25,
	}
	for value, want := range tests {
		cmd, err := Parse([]string{"export", "--at", value, "clip.mov", "out.png"})
		if err != nil {
			t.Fatalf("Parse --at %q returned error: %v", value, err)
		}
		if cmd.Export.AtSeconds != want {
			t.Fatalf("AtSeconds for %q = %v, want %v", value, cmd.Export.AtSeconds, want)
		}
	}
}

func TestParseExportRejectsDurationForSingleFrameOutputs(t *testing.T) {
	for _, output := range []string{"out.png", "out.jpg", "out.jpeg", "out.txt", "out.ansi"} {
		_, err := Parse([]string{"export", "--duration", "3s", "clip.mov", output})
		if err == nil {
			t.Fatalf("Parse returned nil error for --duration with %s", output)
		}
	}
}

func TestParseExportRejectsInvalidTimeValues(t *testing.T) {
	for _, value := range []string{"", "abc", "-1s", "1:2:3:4", "1m"} {
		_, err := Parse([]string{"export", "--at", value, "clip.mov", "out.png"})
		if err == nil {
			t.Fatalf("Parse returned nil error for invalid --at %q", value)
		}
	}
}
```

Update `TestParseExportRejectsNonMP4Output` into `TestParseExportRejectsUnsupportedOutputExtension` or remove it once the broader extension test exists.

Update `TestHelpTextMentionsCommands` expectations:

```go
"mojify export [options] <source> <output>",
"Export Mojify output to a supported file format",
"--at <timestamp>",
"--duration <duration>",
".mp4, .webm, .mov, .gif, .apng, .png, .jpg, .jpeg, .txt, .ansi",
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail because `ExportOptions` lacks timestamp fields and parser still rejects non-MP4 outputs.

- [ ] **Step 3: Add export time fields and parser helpers**

Modify `packages/core/internal/cli/cli.go` imports to include exporter:

```go
import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/jass/mojify/packages/core/internal/exporter"
)
```

Remove the unused `path/filepath` import after replacing `.mp4` validation.

Extend `ExportOptions`:

```go
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
}
```

Add helpers near `isValidExportBitrate`:

```go
func parseExportTimeValue(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("time value is required")
	}
	if strings.HasSuffix(value, "s") {
		seconds, err := strconv.ParseFloat(strings.TrimSuffix(value, "s"), 64)
		if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
			return 0, fmt.Errorf("invalid time value %q", value)
		}
		return seconds, nil
	}
	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return 0, fmt.Errorf("invalid time value %q", value)
		}
		total := 0.0
		for i, part := range parts {
			number, err := strconv.ParseFloat(part, 64)
			if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
				return 0, fmt.Errorf("invalid time value %q", value)
			}
			if i < len(parts)-1 && number >= 60 {
				return 0, fmt.Errorf("invalid time value %q", value)
			}
			total = total*60 + number
		}
		return total, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, fmt.Errorf("invalid time value %q", value)
	}
	return seconds, nil
}
```

- [ ] **Step 4: Parse `--at`, `--duration`, and supported extension**

In `parseExportCommand`, add cases:

```go
		case "--at":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --at")
			}
			i++
			seconds, err := parseExportTimeValue(args[i])
			if err != nil {
				return Command{}, fmt.Errorf("export requires --at to be a timestamp: %w", err)
			}
			options.HasAt = true
			options.AtSeconds = seconds
		case "--duration":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --duration")
			}
			i++
			seconds, err := parseExportTimeValue(args[i])
			if err != nil || seconds <= 0 {
				return Command{}, fmt.Errorf("export requires --duration to be greater than 0")
			}
			options.HasDuration = true
			options.DurationSeconds = seconds
```

Replace the `.mp4` check with:

```go
	format, err := exporter.ResolveOutputFormat(paths[1])
	if err != nil {
		return Command{}, err
	}
	if options.HasDuration && format.SingleFrame {
		return Command{}, fmt.Errorf("export --duration is valid only for video and animated outputs")
	}
```

Update help text:

```text
  mojify export [options] <source> <output>             Export Mojify output to a supported file format

Export output formats:
  .mp4, .webm, .mov, .gif, .apng, .png, .jpg, .jpeg, .txt, .ansi

Export options:
  --width <n>          Pixel width for media/image outputs; columns for text outputs
  --fps <n>            Output frames per second for time-based outputs
  --at <timestamp>     Start from timestamp: 10, 10s, 1:23, or 01:02:03.250
  --duration <value>   Limit time-based exports; valid for video and animated outputs
```

- [ ] **Step 5: Run CLI tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
git commit --no-gpg-sign -m "feat: parse multi-format export options"
```

---

### Task 3: Pass Export Format and Time Options Through CLI Handoff

**Files:**
- Modify: `packages/core/internal/cli/export.go`
- Modify: `packages/core/internal/cli/export_test.go`
- Modify: `packages/core/internal/exporter/layout.go`

- [ ] **Step 1: Add failing CLI handoff test**

Append to `packages/core/internal/cli/export_test.go`:

```go
func TestRunExportPassesTimeOptionsAndUsesGeneralExporter(t *testing.T) {
	exportErr := errors.New("stop after export handoff")
	var gotOptions exporter.Options

	err := runExportWithOptions(context.Background(), "clip.mov", "out.gif", io.Discard, ExportOptions{
		HasAt:           true,
		AtSeconds:       10.5,
		HasDuration:     true,
		DurationSeconds: 3,
		Width:           320,
		FPS:             12,
		Overwrite:       true,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotOptions = options
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if !gotOptions.HasAt || gotOptions.AtSeconds != 10.5 {
		t.Fatalf("At = (%v, %v), want (true, 10.5)", gotOptions.HasAt, gotOptions.AtSeconds)
	}
	if !gotOptions.HasDuration || gotOptions.DurationSeconds != 3 {
		t.Fatalf("Duration = (%v, %v), want (true, 3)", gotOptions.HasDuration, gotOptions.DurationSeconds)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail because `exportRunnerOptions.Export` and timestamp fields do not exist in `exporter.Options`.

- [ ] **Step 3: Extend exporter options**

Modify `packages/core/internal/exporter/layout.go`:

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
}
```

- [ ] **Step 4: Generalize CLI runner injection**

Modify `packages/core/internal/cli/export.go`:

```go
type exportRunnerOptions struct {
	YTDLPPath string
	Export    func(context.Context, string, string, io.Writer, exporter.Options) error
}
```

Build `exportOptions` with:

```go
		HasAt:           options.HasAt,
		AtSeconds:       options.AtSeconds,
		HasDuration:     options.HasDuration,
		DurationSeconds: options.DurationSeconds,
```

Replace the `ExportMP4` fallback with:

```go
	exportFn := runnerOptions.Export
	if exportFn == nil {
		exportFn = exporter.Export
	}
	return exportFn(ctx, resolved.Path, outputPath, stderr, exportOptions)
```

Update older tests in `export_test.go` so injected functions use `Export:` instead of `ExportMP4:`.

- [ ] **Step 5: Add a compatibility wrapper in exporter**

In `packages/core/internal/exporter/export.go`, add this stub before full generalized export is implemented:

```go
func Export(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) error {
	return ExportMP4(ctx, inputPath, outputPath, stderr, options)
}
```

This intentionally keeps runtime behavior unchanged until format routing is implemented in later tasks.

- [ ] **Step 6: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go packages/core/internal/exporter/layout.go packages/core/internal/exporter/export.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go packages/core/internal/exporter/layout.go packages/core/internal/exporter/export.go
git commit --no-gpg-sign -m "feat: pass export timing options"
```

---

### Task 4: Add Timestamp and Duration-Aware Decoding

**Files:**
- Modify: `packages/core/internal/media/decode.go`
- Modify: `packages/core/internal/media/decode_test.go`

- [ ] **Step 1: Add failing decoder arg tests**

Append to `packages/core/internal/media/decode_test.go`:

```go
func TestExportDecodeArgsWithAtAndDuration(t *testing.T) {
	args := ExportDecodeArgsWithOptions(ExportDecodeOptions{
		Path:            "clip.mov",
		Width:           80,
		Height:          24,
		FPS:             12,
		HasAt:           true,
		AtSeconds:       10.5,
		HasDuration:     true,
		DurationSeconds: 3,
	})
	want := []string{
		"-v", "error",
		"-ss", "10.5",
		"-i", "clip.mov",
		"-t", "3",
		"-vf", "scale=80:24:force_original_aspect_ratio=decrease,pad=80:24:(ow-iw)/2:(oh-ih)/2,fps=12",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestExportDecodeArgsWithoutTimeOptionsMatchesExistingShape(t *testing.T) {
	args := ExportDecodeArgsWithOptions(ExportDecodeOptions{Path: "clip.mov", Width: 80, Height: 24})
	want := ExportDecodeArgs("clip.mov", 80, 24, 0)
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}
```

If `decode_test.go` does not already import `reflect`, add it.

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: fail because `ExportDecodeOptions` and `ExportDecodeArgsWithOptions` do not exist.

- [ ] **Step 3: Implement decode options**

Modify `packages/core/internal/media/decode.go`:

```go
type ExportDecodeOptions struct {
	Path            string
	Width           int
	Height          int
	FPS             float64
	HasAt           bool
	AtSeconds       float64
	HasDuration     bool
	DurationSeconds float64
}

func ExportDecodeArgsWithOptions(options ExportDecodeOptions) []string {
	widthText := strconv.Itoa(options.Width)
	heightText := strconv.Itoa(options.Height)
	filter := "scale=" + widthText + ":" + heightText + ":force_original_aspect_ratio=decrease,pad=" + widthText + ":" + heightText + ":(ow-iw)/2:(oh-ih)/2"
	if options.FPS > 0 {
		filter += ",fps=" + formatFPS(options.FPS)
	}

	args := []string{"-v", "error"}
	if options.HasAt {
		args = append(args, "-ss", formatFPS(options.AtSeconds))
	}
	args = append(args, "-i", options.Path)
	if options.HasDuration {
		args = append(args, "-t", formatFPS(options.DurationSeconds))
	}
	args = append(args,
		"-vf", filter,
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	)
	return args
}
```

Replace `ExportDecodeArgs` body with:

```go
func ExportDecodeArgs(path string, width int, height int, fps float64) []string {
	return ExportDecodeArgsWithOptions(ExportDecodeOptions{Path: path, Width: width, Height: height, FPS: fps})
}
```

Add a new starter:

```go
func StartExportDecoderWithOptions(ctx context.Context, options ExportDecodeOptions) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", ExportDecodeArgsWithOptions(options)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, formatToolStartError("ffmpeg", err)
	}
	return cmd, stdout, nil
}
```

Keep `StartExportDecoderContext` as a wrapper:

```go
func StartExportDecoderContext(ctx context.Context, path string, width int, height int, fps float64) (*exec.Cmd, io.ReadCloser, error) {
	return StartExportDecoderWithOptions(ctx, ExportDecodeOptions{Path: path, Width: width, Height: height, FPS: fps})
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/media/decode.go packages/core/internal/media/decode_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/media/decode.go packages/core/internal/media/decode_test.go
git commit --no-gpg-sign -m "feat: add timestamped export decoding"
```

---

### Task 5: Add Curated FFmpeg Encoder Args

**Files:**
- Modify: `packages/core/internal/media/encode.go`
- Modify: `packages/core/internal/media/encode_test.go`

- [ ] **Step 1: Add failing encoder tests**

Append to `packages/core/internal/media/encode_test.go`:

```go
func TestRawVideoEncodeArgsForWebM(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatWebM,
		InputPath:  "source.mov",
		OutputPath: "out.webm",
		Width:      320,
		Height:     184,
		FPS:        12,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	for _, pair := range [][2]string{
		{"-c:v", "libvpx-vp9"},
		{"-pix_fmt", "yuv420p"},
		{"-c:a", "libopus"},
	} {
		if !containsAdjacent(args, pair[0], pair[1]) {
			t.Fatalf("args missing %s %s: %#v", pair[0], pair[1], args)
		}
	}
}

func TestRawVideoEncodeArgsForAnimatedGIF(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatGIF,
		OutputPath: "out.gif",
		Width:      320,
		Height:     184,
		FPS:        12,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-filter_complex", "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse") {
		t.Fatalf("args missing GIF palette filter: %#v", args)
	}
}

func TestRawVideoEncodeArgsForStillPNG(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatPNG,
		OutputPath: "out.png",
		Width:      320,
		Height:     184,
		FPS:        1,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-frames:v", "1") {
		t.Fatalf("args missing -frames:v 1: %#v", args)
	}
	if !containsAdjacent(args, "-c:v", "png") {
		t.Fatalf("args missing -c:v png: %#v", args)
	}
}

func TestRawVideoEncodeArgsRejectsAudioFormatWithoutInputPath(t *testing.T) {
	_, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatWebM,
		OutputPath: "out.webm",
		Width:      320,
		Height:     184,
		FPS:        12,
	})
	if err == nil {
		t.Fatal("RawVideoEncodeArgs returned nil error without input path for audio-capable format")
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: fail because `RawVideoEncodeArgs`, `RawVideoEncodeOptions`, and encode format constants are undefined.

- [ ] **Step 3: Add encode format model**

Modify `packages/core/internal/media/encode.go`:

```go
type EncodeFormat string

const (
	EncodeFormatMP4  EncodeFormat = "mp4"
	EncodeFormatWebM EncodeFormat = "webm"
	EncodeFormatMOV  EncodeFormat = "mov"
	EncodeFormatGIF  EncodeFormat = "gif"
	EncodeFormatAPNG EncodeFormat = "apng"
	EncodeFormatPNG  EncodeFormat = "png"
	EncodeFormatJPEG EncodeFormat = "jpeg"
)

type RawVideoEncodeOptions struct {
	Format        EncodeFormat
	InputPath     string
	OutputPath    string
	Width         int
	Height        int
	FPS           float64
	Bitrate       string
	Overwrite     bool
	IncludeAudio  bool
	SingleFrame   bool
}
```

- [ ] **Step 4: Implement curated args**

Add to `packages/core/internal/media/encode.go`:

```go
func RawVideoEncodeArgs(options RawVideoEncodeOptions) ([]string, error) {
	if strings.TrimSpace(options.OutputPath) == "" {
		return nil, fmt.Errorf("output path is required")
	}
	if options.Width <= 0 || options.Height <= 0 {
		return nil, fmt.Errorf("invalid output dimensions %dx%d", options.Width, options.Height)
	}
	if options.FPS <= 0 {
		return nil, fmt.Errorf("invalid FPS %s", formatFPS(options.FPS))
	}
	if options.IncludeAudio && strings.TrimSpace(options.InputPath) == "" {
		return nil, fmt.Errorf("input path is required for audio-capable export")
	}

	overwriteFlag := "-n"
	if options.Overwrite {
		overwriteFlag = "-y"
	}
	args := []string{
		"-v", "error",
		overwriteFlag,
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", fmt.Sprintf("%dx%d", options.Width, options.Height),
		"-r", formatFPS(options.FPS),
		"-i", "pipe:0",
	}
	if options.IncludeAudio {
		args = append(args, "-i", options.InputPath, "-map", "0:v:0", "-map", "1:a?")
	}

	switch options.Format {
	case EncodeFormatMP4:
		args = append(args, "-c:v", "libx264", "-preset", defaultMP4VideoPreset, "-pix_fmt", "yuv420p")
		if options.IncludeAudio {
			args = append(args, "-c:a", "aac", "-shortest")
		}
	case EncodeFormatMOV:
		args = append(args, "-c:v", "libx264", "-preset", defaultMP4VideoPreset, "-pix_fmt", "yuv420p")
		if options.IncludeAudio {
			args = append(args, "-c:a", "aac", "-shortest")
		}
	case EncodeFormatWebM:
		args = append(args, "-c:v", "libvpx-vp9", "-pix_fmt", "yuv420p")
		if options.IncludeAudio {
			args = append(args, "-c:a", "libopus", "-shortest")
		}
	case EncodeFormatGIF:
		args = append(args, "-filter_complex", "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", "-loop", "0")
	case EncodeFormatAPNG:
		args = append(args, "-c:v", "apng", "-plays", "0")
	case EncodeFormatPNG:
		args = append(args, "-frames:v", "1", "-c:v", "png")
	case EncodeFormatJPEG:
		args = append(args, "-frames:v", "1", "-c:v", "mjpeg", "-q:v", "2")
	default:
		return nil, fmt.Errorf("unsupported encode format %q", options.Format)
	}
	if bitrate := strings.TrimSpace(options.Bitrate); bitrate != "" && (options.Format == EncodeFormatMP4 || options.Format == EncodeFormatMOV || options.Format == EncodeFormatWebM) {
		args = append(args, "-b:v", bitrate)
	}
	args = append(args, options.OutputPath)
	return args, nil
}
```

Refactor `MP4EncodeArgs` to call the generic helper:

```go
func MP4EncodeArgs(options MP4EncodeOptions) ([]string, error) {
	return RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:       EncodeFormatMP4,
		InputPath:    options.InputPath,
		OutputPath:   options.OutputPath,
		Width:        options.Width,
		Height:       options.Height,
		FPS:          options.FPS,
		Bitrate:      options.Bitrate,
		Overwrite:    options.Overwrite,
		IncludeAudio: true,
	})
}
```

Add starter:

```go
func StartRawVideoEncoderContext(ctx context.Context, options RawVideoEncodeOptions, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	args, err := RawVideoEncodeArgs(options)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if stderr != nil {
		cmd.Stderr = stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, formatToolStartError("ffmpeg", err)
	}
	return cmd, stdin, nil
}
```

Keep `StartMP4EncoderContext` as a wrapper.

- [ ] **Step 5: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/media/encode.go packages/core/internal/media/encode_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/media/encode.go packages/core/internal/media/encode_test.go
git commit --no-gpg-sign -m "feat: add curated export encoders"
```

---

### Task 6: Add Text Export Serializers

**Files:**
- Create: `packages/core/internal/exporter/text.go`
- Create: `packages/core/internal/exporter/text_test.go`

- [ ] **Step 1: Add failing text serializer tests**

Create `packages/core/internal/exporter/text_test.go`:

```go
package exporter

import (
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestSerializePlainTextFrame(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  2,
		Height: 2,
		Cells: []render.Cell{
			{Ch: 'A'}, {Ch: 'B'},
			{Ch: 'C'}, {Ch: 'D'},
		},
	}
	got, err := SerializeTextFrame(frame, OutputFormat{Extension: ".txt", Family: OutputFamilyText, Text: true})
	if err != nil {
		t.Fatalf("SerializeTextFrame returned error: %v", err)
	}
	want := "AB\nCD\n"
	if got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

func TestSerializeANSIFrame(t *testing.T) {
	frame := render.CharacterFrame{
		Width:  1,
		Height: 1,
		Cells: []render.Cell{{Ch: '@', R: 255, G: 1, B: 2}},
	}
	got, err := SerializeTextFrame(frame, OutputFormat{Extension: ".ansi", Family: OutputFamilyText, Text: true})
	if err != nil {
		t.Fatalf("SerializeTextFrame returned error: %v", err)
	}
	for _, want := range []string{"\x1b[38;2;255;1;2m", "@", "\x1b[0m", "\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ANSI output missing %q in %q", want, got)
		}
	}
}

func TestSerializeTextFrameRejectsInvalidFrame(t *testing.T) {
	_, err := SerializeTextFrame(render.CharacterFrame{Width: 2, Height: 1}, OutputFormat{Extension: ".txt", Text: true})
	if err == nil {
		t.Fatal("SerializeTextFrame returned nil error for invalid frame")
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail because `SerializeTextFrame` does not exist.

- [ ] **Step 3: Implement text serializers**

Create `packages/core/internal/exporter/text.go`:

```go
package exporter

import (
	"fmt"
	"strings"

	"github.com/jass/mojify/packages/core/internal/render"
)

func SerializeTextFrame(frame render.CharacterFrame, format OutputFormat) (string, error) {
	if err := validateTextFrame(frame); err != nil {
		return "", err
	}
	switch format.Extension {
	case ".txt":
		return serializePlainTextFrame(frame), nil
	case ".ansi":
		return serializeANSITextFrame(frame), nil
	default:
		return "", fmt.Errorf("unsupported text export format %q", format.Extension)
	}
}

func validateTextFrame(frame render.CharacterFrame) error {
	if frame.Width <= 0 || frame.Height <= 0 || len(frame.Cells) != frame.Width*frame.Height {
		return fmt.Errorf("invalid character frame: width=%d height=%d cells=%d", frame.Width, frame.Height, len(frame.Cells))
	}
	return nil
}

func serializePlainTextFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	for row := 0; row < frame.Height; row++ {
		for col := 0; col < frame.Width; col++ {
			ch := frame.Cells[row*frame.Width+col].Ch
			if ch == 0 {
				ch = ' '
			}
			b.WriteRune(ch)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func serializeANSITextFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	var lastR, lastG, lastB uint8
	hasColor := false
	for row := 0; row < frame.Height; row++ {
		for col := 0; col < frame.Width; col++ {
			cell := frame.Cells[row*frame.Width+col]
			if !hasColor || cell.R != lastR || cell.G != lastG || cell.B != lastB {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
				lastR, lastG, lastB = cell.R, cell.G, cell.B
				hasColor = true
			}
			ch := cell.Ch
			if ch == 0 {
				ch = ' '
			}
			b.WriteRune(ch)
		}
		if row != frame.Height-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteString("\x1b[0m\n")
	return b.String()
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/text.go packages/core/internal/exporter/text_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/exporter/text.go packages/core/internal/exporter/text_test.go
git commit --no-gpg-sign -m "feat: add text export serializers"
```

---

### Task 7: Route Export by Format Family

**Files:**
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/exporter/export_test.go`
- Modify: `packages/core/internal/exporter/layout.go`
- Modify: `packages/core/internal/exporter/layout_test.go`

- [ ] **Step 1: Add failing layout test for text width**

Append to `packages/core/internal/exporter/layout_test.go`:

```go
func TestResolveLayoutUsesWidthAsColumnsForTextOutput(t *testing.T) {
	layout, err := ResolveLayout(InputInfo{Width: 1920, Height: 1080, FPS: 24}, Options{
		Width: 80,
		Format: OutputFormat{Extension: ".txt", Family: OutputFamilyText, Text: true, SingleFrame: true},
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
```

- [ ] **Step 2: Add failing export preflight tests**

Append to `packages/core/internal/exporter/export_test.go`:

```go
func TestCheckOutputPathRejectsUnsupportedFormat(t *testing.T) {
	err := CheckOutputPath("out.webp", Options{})
	if err == nil {
		t.Fatal("CheckOutputPath returned nil error for unsupported format")
	}
}

func TestCheckOutputPathRejectsDurationForSingleFrameFormat(t *testing.T) {
	err := CheckOutputPath("out.png", Options{HasDuration: true, DurationSeconds: 3})
	if err == nil {
		t.Fatal("CheckOutputPath returned nil error for duration with still output")
	}
}
```

- [ ] **Step 3: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail because `Options.Format` does not exist and `CheckOutputPath` does not resolve formats.

- [ ] **Step 4: Add `Format` to options and resolve it during preflight**

Modify `packages/core/internal/exporter/layout.go`:

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
}
```

Add helper in `export.go`:

```go
func resolveOptionsFormat(outputPath string, options Options) (OutputFormat, error) {
	if options.Format.Extension != "" {
		return options.Format, nil
	}
	return ResolveOutputFormat(outputPath)
}
```

Modify `CheckOutputPath`:

```go
	format, err := resolveOptionsFormat(outputPath, options)
	if err != nil {
		return err
	}
	if options.HasDuration && format.SingleFrame {
		return fmt.Errorf("export --duration is valid only for video and animated outputs")
	}
```

- [ ] **Step 5: Add text width behavior**

Modify `ResolveLayout` in `layout.go` before output width calculation:

```go
	if options.Format.Text {
		cols := options.Width
		if cols == 0 {
			cols = min(input.Width, DefaultExportMaxWidth) / ExportCellWidth
		}
		if cols <= 0 {
			cols = 1
		}
		outputWidth := cols * ExportCellWidth
		derivedRows := int(math.Round(float64(cols) * float64(input.Height) / float64(input.Width)))
		rows := max(derivedRows, 1)
		return Layout{
			OutputWidth:  outputWidth,
			OutputHeight: rows * ExportCellHeight,
			Grid:         render.Grid{Cols: cols, Rows: rows},
			FPS:          fps,
		}, nil
	}
```

- [ ] **Step 6: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/export.go packages/core/internal/exporter/export_test.go packages/core/internal/exporter/layout.go packages/core/internal/exporter/layout_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add packages/core/internal/exporter/export.go packages/core/internal/exporter/export_test.go packages/core/internal/exporter/layout.go packages/core/internal/exporter/layout_test.go
git commit --no-gpg-sign -m "feat: route export formats"
```

---

### Task 8: Implement Single-Frame Text Exports

**Files:**
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/exporter/export_test.go`

- [ ] **Step 1: Add failing text export integration-style unit test**

Append to `packages/core/internal/exporter/export_test.go`:

```go
func TestExportTextWritesSingleFrameFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "frame.txt")
	frame := render.NewRGBFrame(2, 2, []byte{
		255, 255, 255, 0, 0, 0,
		0, 0, 0, 255, 255, 255,
	})
	err := exportSingleTextFrameForTest(frame, output, Options{
		Width:     2,
		Overwrite: true,
		Format:    OutputFormat{Extension: ".txt", Family: OutputFamilyText, Text: true, SingleFrame: true},
	})
	if err != nil {
		t.Fatalf("exportSingleTextFrameForTest returned error: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "\n") {
		t.Fatalf("text output = %q, want non-empty multi-line text", string(data))
	}
}
```

Add imports if needed: `strings` and `github.com/jass/mojify/packages/core/internal/render`.

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail because helper does not exist.

- [ ] **Step 3: Implement single-frame text writer helper**

In `packages/core/internal/exporter/export.go`, add:

```go
func exportSingleTextFrameForTest(frame render.RGBFrame, outputPath string, options Options) error {
	layout, err := ResolveLayout(InputInfo{Width: frame.Width, Height: frame.Height, FPS: 1}, options)
	if err != nil {
		return err
	}
	charFrame := render.DefaultRenderer{}.Render(frame, layout.Grid)
	text, err := SerializeTextFrame(charFrame, options.Format)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(text), 0o644)
}
```

Then add production helper:

```go
func exportText(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) (err error) {
	info, err := media.ProbeContext(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	options.Format = format
	layout, err := ResolveLayout(InputInfo{Width: info.Width, Height: info.Height, FPS: info.FPS}, options)
	if err != nil {
		return err
	}
	cmd, pipe, err := media.StartExportDecoderWithOptions(ctx, media.ExportDecodeOptions{
		Path:      inputPath,
		Width:     layout.Grid.Cols,
		Height:    layout.Grid.Rows,
		HasAt:     options.HasAt,
		AtSeconds: options.AtSeconds,
	})
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	defer func() {
		_ = cleanupProcess(cmd, pipe)
	}()
	rgbFrame, err := media.ReadRawFrame(pipe, layout.Grid.Cols, layout.Grid.Rows)
	if err != nil {
		return fmt.Errorf("read decoded frame: %w", err)
	}
	charFrame := render.DefaultRenderer{}.Render(rgbFrame, layout.Grid)
	text, err := SerializeTextFrame(charFrame, format)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write text output: %w", err)
	}
	_ = pipe.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("decoder failed: %w", err)
	}
	fmt.Fprintf(stderr, "export complete: %s\n", outputPath)
	return nil
}
```

Do not route production `Export` to this helper until Task 10; this task just makes the helper testable.

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/export.go packages/core/internal/exporter/export_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/exporter/export.go packages/core/internal/exporter/export_test.go
git commit --no-gpg-sign -m "feat: add text frame export"
```

---

### Task 9: Implement Time-Based Media/Image Export Routing

**Files:**
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/media/encode.go`

- [ ] **Step 1: Add encode-format mapping helper**

In `packages/core/internal/exporter/export.go`, add:

```go
func encodeFormatForOutput(format OutputFormat) (media.EncodeFormat, error) {
	switch format.Extension {
	case ".mp4":
		return media.EncodeFormatMP4, nil
	case ".webm":
		return media.EncodeFormatWebM, nil
	case ".mov":
		return media.EncodeFormatMOV, nil
	case ".gif":
		return media.EncodeFormatGIF, nil
	case ".apng":
		return media.EncodeFormatAPNG, nil
	case ".png":
		return media.EncodeFormatPNG, nil
	case ".jpg", ".jpeg":
		return media.EncodeFormatJPEG, nil
	default:
		return "", fmt.Errorf("unsupported media export format %q", format.Extension)
	}
}
```

- [ ] **Step 2: Generalize current MP4 export path**

In `export.go`, extract the current `ExportMP4` body into:

```go
func exportTimeBasedMedia(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) (err error) {
	// Same structure as current ExportMP4:
	// probe -> ResolveLayout -> progress -> decoder -> encoder -> ordered frame pipeline -> finalize.
}
```

Use these differences from current MP4:

```go
	options.Format = format
	encodeFormat, err := encodeFormatForOutput(format)
	if err != nil {
		return err
	}
	decodeCmd, decodePipe, err := media.StartExportDecoderWithOptions(ctx, media.ExportDecodeOptions{
		Path:            inputPath,
		Width:           layout.Grid.Cols,
		Height:          layout.Grid.Rows,
		FPS:             options.FPS,
		HasAt:           options.HasAt,
		AtSeconds:       options.AtSeconds,
		HasDuration:     options.HasDuration,
		DurationSeconds: options.DurationSeconds,
	})
```

Start the encoder with:

```go
	encodeCmd, encodePipe, err := media.StartRawVideoEncoderContext(ctx, media.RawVideoEncodeOptions{
		Format:       encodeFormat,
		InputPath:    inputPath,
		OutputPath:   outputPath,
		Width:        layout.OutputWidth,
		Height:       layout.OutputHeight,
		FPS:          layout.FPS,
		Bitrate:      options.Bitrate,
		Overwrite:    options.Overwrite,
		IncludeAudio: format.SupportsAudio,
	}, progress.lineSafeWriter(stderr))
```

Change `progress.Finalizing()` to use a new generic phase if Task 11 already exists; otherwise keep `finalizing mp4...` for now and fix wording in Task 11.

- [ ] **Step 3: Keep MP4 compatibility wrapper**

Make `ExportMP4` call the generalized path with explicit MP4 format:

```go
func ExportMP4(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) error {
	format := OutputFormat{Extension: ".mp4", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true}
	options.Format = format
	return exportTimeBasedMedia(ctx, inputPath, outputPath, stderr, options, format)
}
```

- [ ] **Step 4: Add single-frame media/image export helper**

Add:

```go
func exportSingleFrameMedia(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) (err error) {
	info, err := media.ProbeContext(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	options.Format = format
	layout, err := ResolveLayout(InputInfo{Width: info.Width, Height: info.Height, FPS: 1}, options)
	if err != nil {
		return err
	}
	encodeFormat, err := encodeFormatForOutput(format)
	if err != nil {
		return err
	}
	decodeCmd, decodePipe, err := media.StartExportDecoderWithOptions(ctx, media.ExportDecodeOptions{
		Path:      inputPath,
		Width:     layout.Grid.Cols,
		Height:    layout.Grid.Rows,
		HasAt:     options.HasAt,
		AtSeconds: options.AtSeconds,
	})
	if err != nil {
		return fmt.Errorf("start decoder: %w", err)
	}
	defer func() {
		_ = cleanupProcess(decodeCmd, decodePipe)
	}()
	encodeCmd, encodePipe, err := media.StartRawVideoEncoderContext(ctx, media.RawVideoEncodeOptions{
		Format:      encodeFormat,
		OutputPath:  outputPath,
		Width:       layout.OutputWidth,
		Height:      layout.OutputHeight,
		FPS:         1,
		Overwrite:   options.Overwrite,
		SingleFrame: true,
	}, stderr)
	if err != nil {
		return fmt.Errorf("start encoder: %w", err)
	}
	defer func() {
		_ = encodePipe.Close()
		_ = encodeCmd.Wait()
	}()
	rgbFrame, err := media.ReadRawFrame(decodePipe, layout.Grid.Cols, layout.Grid.Rows)
	if err != nil {
		return fmt.Errorf("read decoded frame: %w", err)
	}
	face, err := fonts.DefaultFace()
	if err != nil {
		return fmt.Errorf("load export font: %w", err)
	}
	raw, err := NewRasterizer(face).Rasterize(render.DefaultRenderer{}.Render(rgbFrame, layout.Grid), layout)
	if err != nil {
		return err
	}
	if _, err := encodePipe.Write(raw); err != nil {
		return fmt.Errorf("write encoder frame: %w", err)
	}
	if err := encodePipe.Close(); err != nil {
		return fmt.Errorf("close encoder pipe: %w", err)
	}
	if err := encodeCmd.Wait(); err != nil {
		return fmt.Errorf("encoder failed: %w", err)
	}
	_ = decodePipe.Close()
	if err := decodeCmd.Wait(); err != nil {
		return fmt.Errorf("decoder failed: %w", err)
	}
	fmt.Fprintf(stderr, "export complete: %s\n", outputPath)
	return nil
}
```

Add missing imports: `github.com/jass/mojify/packages/core/internal/exporter/fonts`.

- [ ] **Step 5: Run focused tests**

Run:

```bash
gofmt -w packages/core/internal/exporter/export.go packages/core/internal/media/encode.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter ./packages/core/internal/media
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/exporter/export.go packages/core/internal/media/encode.go
git commit --no-gpg-sign -m "feat: generalize media export pipeline"
```

---

### Task 10: Wire Public `Export` Routing

**Files:**
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/exporter/export_test.go`

- [ ] **Step 1: Add failing routing tests with injected hooks**

Because production export launches FFmpeg, keep routing tests focused. Add package-level variables in `export.go`:

```go
var (
	exportTimeBasedMediaFunc = exportTimeBasedMedia
	exportSingleFrameMediaFunc = exportSingleFrameMedia
	exportTextFunc = exportText
)
```

Append tests:

```go
func TestExportRoutesByFormatFamily(t *testing.T) {
	tests := []struct {
		output string
		want   string
	}{
		{"out.mp4", "time"},
		{"out.gif", "time"},
		{"out.png", "single"},
		{"out.txt", "text"},
	}
	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			oldTime := exportTimeBasedMediaFunc
			oldSingle := exportSingleFrameMediaFunc
			oldText := exportTextFunc
			t.Cleanup(func() {
				exportTimeBasedMediaFunc = oldTime
				exportSingleFrameMediaFunc = oldSingle
				exportTextFunc = oldText
			})
			called := ""
			exportTimeBasedMediaFunc = func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
				called = "time"
				return nil
			}
			exportSingleFrameMediaFunc = func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
				called = "single"
				return nil
			}
			exportTextFunc = func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
				called = "text"
				return nil
			}
			if err := Export(context.Background(), "in.mov", tc.output, io.Discard, Options{Overwrite: true}); err != nil {
				t.Fatalf("Export returned error: %v", err)
			}
			if called != tc.want {
				t.Fatalf("called = %q, want %q", called, tc.want)
			}
		})
	}
}
```

Add imports if needed: `context` and `io`.

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail until routing variables and public `Export` are implemented.

- [ ] **Step 3: Implement `Export` routing**

Replace the temporary `Export` stub:

```go
func Export(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) error {
	if err := CheckOutputPath(outputPath, options); err != nil {
		return err
	}
	format, err := resolveOptionsFormat(outputPath, options)
	if err != nil {
		return err
	}
	options.Format = format
	switch {
	case format.TimeBased:
		return exportTimeBasedMediaFunc(ctx, inputPath, outputPath, stderr, options, format)
	case format.Text:
		return exportTextFunc(ctx, inputPath, outputPath, stderr, options, format)
	case format.SingleFrame:
		return exportSingleFrameMediaFunc(ctx, inputPath, outputPath, stderr, options, format)
	default:
		return fmt.Errorf("unsupported export format family %q", format.Family)
	}
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/export.go packages/core/internal/exporter/export_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/exporter/export.go packages/core/internal/exporter/export_test.go
git commit --no-gpg-sign -m "feat: route exports by output format"
```

---

### Task 11: Make Progress Text Format-Neutral

**Files:**
- Modify: `packages/core/internal/exporter/progress.go`
- Modify: `packages/core/internal/exporter/progress_test.go`
- Modify: `packages/core/internal/exporter/export.go`

- [ ] **Step 1: Add failing progress wording test**

Append to `packages/core/internal/exporter/progress_test.go`:

```go
func TestProgressReporterUsesFormatLabel(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 2,
		Now: func() time.Time { return time.Unix(0, 0) },
		Label: "animated export",
		FinalizingLabel: "finalizing gif...",
	})
	reporter.Start("in.mp4", "out.gif", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 12})
	reporter.AllFramesWritten(2)
	reporter.Finalizing()
	got := out.String()
	for _, want := range []string{"exporting animated export: 0/2 frames 0%", "exporting animated export: 2/2 frames 100%", "finalizing gif..."} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail because options do not include labels.

- [ ] **Step 3: Add labels to progress reporter**

Modify `progressReporterOptions`:

```go
type progressReporterOptions struct {
	Interactive     bool
	TotalFrames     int
	Now             func() time.Time
	Label           string
	FinalizingLabel string
}
```

Add fields to `progressReporter`:

```go
	label           string
	finalizingLabel string
```

In `newProgressReporter`:

```go
	label := options.Label
	if label == "" {
		label = "video"
	}
	finalizingLabel := options.FinalizingLabel
	if finalizingLabel == "" {
		finalizingLabel = "finalizing output..."
	}
```

Set fields and change `formatFrameStatus`:

```go
return fmt.Sprintf("exporting %s: %d frames", p.label, renderedFrames)
return fmt.Sprintf("exporting %s: %d/%d frames %d%%", p.label, displayFrames, p.totalFrames, percent)
```

Change `Finalizing`:

```go
p.writePhaseLocked(p.finalizingLabel)
```

- [ ] **Step 4: Pass labels from export**

Add helper in `export.go`:

```go
func progressLabelsForFormat(format OutputFormat) (string, string) {
	switch format.Family {
	case OutputFamilyVideo:
		return "video", "finalizing " + strings.TrimPrefix(format.Extension, ".") + "..."
	case OutputFamilyAnimated:
		return "animated export", "finalizing " + strings.TrimPrefix(format.Extension, ".") + "..."
	default:
		return "export", "finalizing output..."
	}
}
```

Use it when creating progress reporters in `exportTimeBasedMedia`.

- [ ] **Step 5: Run tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/progress.go packages/core/internal/exporter/progress_test.go packages/core/internal/exporter/export.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/exporter/progress.go packages/core/internal/exporter/progress_test.go packages/core/internal/exporter/export.go
git commit --no-gpg-sign -m "fix: generalize export progress wording"
```

---

### Task 12: Update QA Script and Docs

**Files:**
- Modify: `scripts/export-qa.sh`
- Modify: `docs/qa/export.md`
- Modify: `README.md`
- Modify: `CONTEXT.md`

- [ ] **Step 1: Update `scripts/export-qa.sh` representative matrix**

Modify `scripts/export-qa.sh` so the synthetic smoke exports representative formats:

```bash
declare -A synthetic_outputs=(
  [mp4]="${export_dir}/low-motion-bars-export.mp4"
  [gif]="${export_dir}/low-motion-bars-export.gif"
  [apng]="${export_dir}/low-motion-bars-export.apng"
  [png]="${export_dir}/low-motion-bars-frame.png"
  [jpg]="${export_dir}/low-motion-bars-frame.jpg"
  [txt]="${export_dir}/low-motion-bars-frame.txt"
  [ansi]="${export_dir}/low-motion-bars-frame.ansi"
)
```

Run exports:

```bash
./bin/mojify export --overwrite --width 320 "${synthetic_source}" "${synthetic_outputs[mp4]}"
./bin/mojify export --overwrite --width 320 --at 0s --duration 2s "${synthetic_source}" "${synthetic_outputs[gif]}"
./bin/mojify export --overwrite --width 320 --at 0s --duration 2s "${synthetic_source}" "${synthetic_outputs[apng]}"
./bin/mojify export --overwrite --width 320 --at 0s "${synthetic_source}" "${synthetic_outputs[png]}"
./bin/mojify export --overwrite --width 320 --at 0s "${synthetic_source}" "${synthetic_outputs[jpg]}"
./bin/mojify export --overwrite --width 80 --at 0s "${synthetic_source}" "${synthetic_outputs[txt]}"
./bin/mojify export --overwrite --width 80 --at 0s "${synthetic_source}" "${synthetic_outputs[ansi]}"
```

Keep optional real-sample audio QA for MP4, and add WebM/MOV optional audio checks only if runtime stays fast enough:

```bash
for ext in mp4 webm mov; do
  out="${export_dir}/real-sample-export.${ext}"
  ./bin/mojify export --overwrite --width 320 --duration 2s "${real_source}" "${out}"
  # ffprobe audio stream as existing script does
done
```

- [ ] **Step 2: Update export QA docs**

In `docs/qa/export.md`, document:

- supported formats
- `--at` timestamp examples
- `--duration` valid for video/animated only
- `.txt` and `.ansi` are single-frame
- `.webp` is deferred
- smoke matrix produced by `bun run qa:export`

- [ ] **Step 3: Update README**

In `README.md`, update examples:

```bash
mojify export --overwrite --width 320 ./demo.mp4 ./dist/demo-mojify.mp4
mojify export --overwrite --width 320 --at 10s --duration 3s ./demo.mp4 ./dist/demo-mojify.gif
mojify export --overwrite --width 320 --at 10s ./demo.mp4 ./dist/demo-frame.png
mojify export --overwrite --width 80 --at 10s ./demo.mp4 ./dist/demo-frame.ansi
```

Update capabilities:

```markdown
- Curated export formats: MP4, WebM, MOV, GIF, APNG, PNG, JPEG, plain text, and ANSI text
```

Remove roadmap items that imply GIF/PNG/still output is still missing, but keep WebP/custom recipes as future work.

- [ ] **Step 4: Update glossary**

In `CONTEXT.md`, add terms:

```markdown
**Curated multi-format export**:
The export stage where `mojify export SOURCE OUTPUT` selects a supported output family by extension: video, animated visual, still image, or still text. Mojify owns these output contracts even when FFmpeg performs the encoding.
_Avoid_: Any FFmpeg-compatible output, raw format passthrough

**Timestamp export selection**:
The `--at` and `--duration` model for choosing where export starts and, for time-based outputs, how much source media to render. Selection is timestamp-based, not exact source-frame addressing.
_Avoid_: Exact frame selection, frame number export

**Text export**:
Single-frame `.txt` or `.ansi` output generated from a rendered Mojify character frame without rasterizing to pixels.
_Avoid_: Animated text export, terminal recording
```

- [ ] **Step 5: Run docs/script checks**

Run:

```bash
bash -n scripts/export-qa.sh
git diff --check
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add scripts/export-qa.sh docs/qa/export.md README.md CONTEXT.md
git commit --no-gpg-sign -m "docs: document multi-format export"
```

---

### Task 13: Full Verification

**Files:**
- No code edits unless verification finds defects.

- [ ] **Step 1: Run formatting checks**

```bash
bun run fmt:check
```

Expected: pass.

- [ ] **Step 2: Run Go tests**

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...
```

Expected: pass.

- [ ] **Step 3: Run workspace tests**

```bash
bun run test
```

Expected: pass.

- [ ] **Step 4: Build binary**

```bash
bun run build
```

Expected: pass and writes `./bin/mojify`.

- [ ] **Step 5: Generate QA clips**

```bash
bun run qa:clips
```

Expected: writes generated clips under `dist/qa/`.

- [ ] **Step 6: Run export QA**

```bash
bun run qa:export
```

Expected:

- MP4 output exists and has a video stream.
- GIF output exists and has frames.
- APNG output exists.
- PNG output exists.
- JPG output exists.
- TXT output is non-empty text.
- ANSI output is non-empty and contains ANSI color escapes.
- Optional real-sample MP4/WebM/MOV audio checks pass when local real samples exist.

- [ ] **Step 7: Manual smoke commands**

```bash
./bin/mojify export --overwrite --width 320 --at 0s --duration 2s dist/qa/low-motion-bars.mp4 dist/qa/export/manual.gif
./bin/mojify export --overwrite --width 320 --at 0s dist/qa/low-motion-bars.mp4 dist/qa/export/manual.png
./bin/mojify export --overwrite --width 80 --at 0s dist/qa/low-motion-bars.mp4 dist/qa/export/manual.ansi
```

Expected: all commands complete and outputs are inspectable.

- [ ] **Step 8: Final diff check**

```bash
git diff --check
git status --short
```

Expected: no whitespace errors; status contains only intentional tracked changes if any fixes were made during verification.

- [ ] **Step 9: Commit verification fixes if needed**

If verification required code/doc fixes:

```bash
git add <fixed-files>
git commit --no-gpg-sign -m "fix: stabilize multi-format export"
```

If no fixes were needed, do not create an empty commit.

---

## Self-Review

- Spec coverage:
  - Extension-routed CLI: Tasks 1, 2, 10.
  - `.mp4`, `.webm`, `.mov`: Tasks 1, 5, 9, 12, 13.
  - `.gif`, `.apng`: Tasks 1, 5, 9, 12, 13.
  - `.png`, `.jpg`, `.jpeg`: Tasks 1, 5, 9, 12, 13.
  - `.txt`, `.ansi`: Tasks 1, 6, 8, 10, 12, 13.
  - `--at` and `--duration`: Tasks 2, 3, 4, 7, 12.
  - Text width as columns: Task 7.
  - Media/image width as pixels: existing layout behavior plus Task 7 regression.
  - WebP deferral: Tasks 1, 12.
  - Existing MP4 behavior preservation: Tasks 5, 9, 13.
- Placeholder scan:
  - No unresolved placeholder markers are intentionally left.
- Type consistency:
  - `ExportOptions` CLI fields map to `exporter.Options`.
  - `OutputFormat` and `OutputFamily` live in `exporter`.
  - FFmpeg encode format constants live in `media`.
  - `Export` is the generalized public exporter; `ExportMP4` remains as a compatibility wrapper.
