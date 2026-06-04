# Still Source Input Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add local still images as first-class Mojify sources for `probe` and single-frame `export` workflows while keeping `play` time-based only.

**Architecture:** Classify resolved sources in the CLI layer as time-based media or local still images. Reuse the existing FFmpeg probe/decode path and existing single-frame export paths; do not add a separate image renderer or image decoder. Apply still-source rules at command boundaries so exported formats remain extension-routed and the exporter stays focused on rendering/encoding.

**Tech Stack:** Go, FFmpeg/ffprobe CLI, existing Mojify renderer/exporter packages, Bun/Turbo scripts, shell QA scripts.

---

## Locked Product Contract

- Local `.png`, `.jpg`, and `.jpeg` files are still sources.
- Still sources are accepted by `mojify probe` and `mojify export`.
- Still sources are rejected by `mojify play` with a clear export-oriented error.
- Still sources export only to `.txt`, `.ansi`, `.png`, `.jpg`, and `.jpeg`.
- Still sources reject `--at` and `--duration`; still images have no timeline.
- Direct HTTP image URLs stay out of scope. HTTP(S) input continues to mean yt-dlp-compatible platform media.
- Animated image sources stay out of scope.
- WebP stays deferred.

## File Structure

- Modify: `CONTEXT.md`
  - Already updated during grilling with `Still source`, `Still source export`, and `Still source timestamp rejection`.
- Modify: `packages/core/internal/cli/source.go`
  - Add source kind classification and carry it on `resolvedSourceMedia`.
- Modify: `packages/core/internal/cli/source_test.go`
  - Cover local still classification and platform URL classification.
- Modify: `packages/core/internal/cli/cli.go`
  - Update help and parser wording from video-only input to source media.
- Modify: `packages/core/internal/cli/cli_test.go`
  - Cover source wording and keep existing URL/local path parsing stable.
- Modify: `packages/core/internal/cli/probe.go`
  - Print clean image metadata for still sources.
- Modify: `packages/core/internal/cli/probe_test.go`
  - Cover still-source probe output.
- Modify: `packages/core/internal/cli/play.go`
  - Reject still sources before terminal setup, decoder startup, or audio startup.
- Modify: `packages/core/internal/cli/play_test.go`
  - Cover still-source playback rejection.
- Modify: `packages/core/internal/cli/export.go`
  - Validate still-source output family and timeline flags before calling `exporter.Export`.
- Modify: `packages/core/internal/cli/export_test.go`
  - Cover still-source export handoff and still-source rejections.
- Modify: `scripts/generate-qa-clips.sh`
  - Generate an ignored still source fixture under `dist/qa/`.
- Modify: `scripts/export-qa.sh`
  - Add still-source export smoke checks.
- Modify: `docs/qa/export.md`
  - Document still-source QA and validation.
- Modify: `README.md`
  - Add one still-source usage example and update current capability wording.

---

### Task 1: Add Source Kind Classification

**Files:**
- Modify: `packages/core/internal/cli/source.go`
- Modify: `packages/core/internal/cli/source_test.go`

- [ ] **Step 1: Write failing local still-source classification tests**

Add these tests to `packages/core/internal/cli/source_test.go`:

```go
func TestResolveSourceMediaClassifiesLocalStillSources(t *testing.T) {
	for _, source := range []string{
		"frame.png",
		"FRAME.PNG",
		"poster.jpg",
		"poster.jpeg",
	} {
		t.Run(source, func(t *testing.T) {
			resolved, err := resolveSourceMediaWithOptions(context.Background(), source, sourceResolverOptions{
				YTDLPPath: "missing-yt-dlp-for-local-test",
			})
			if err != nil {
				t.Fatalf("resolveSourceMediaWithOptions returned error: %v", err)
			}
			defer resolved.Cleanup()
			if resolved.Kind != sourceKindStill {
				t.Fatalf("Kind = %v, want sourceKindStill", resolved.Kind)
			}
			if resolved.Path != source {
				t.Fatalf("Path = %q, want original local path", resolved.Path)
			}
			if resolved.Temporary {
				t.Fatal("Temporary = true, want false for local still source")
			}
		})
	}
}

func TestResolveSourceMediaClassifiesLocalNonStillSourcesAsTimeBased(t *testing.T) {
	for _, source := range []string{
		"clip.mp4",
		"clip.mov",
		"clip.mkv",
		"source-without-extension",
	} {
		t.Run(source, func(t *testing.T) {
			resolved, err := resolveSourceMediaWithOptions(context.Background(), source, sourceResolverOptions{
				YTDLPPath: "missing-yt-dlp-for-local-test",
			})
			if err != nil {
				t.Fatalf("resolveSourceMediaWithOptions returned error: %v", err)
			}
			defer resolved.Cleanup()
			if resolved.Kind != sourceKindTimeBased {
				t.Fatalf("Kind = %v, want sourceKindTimeBased", resolved.Kind)
			}
		})
	}
}
```

- [ ] **Step 2: Run source tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestResolveSourceMediaClassifies'
```

Expected: fail to compile because `resolved.Kind`, `sourceKindStill`, and `sourceKindTimeBased` do not exist.

- [ ] **Step 3: Implement source kind classification**

In `packages/core/internal/cli/source.go`, add `SourceKind`-style internals near `resolvedSourceMedia`:

```go
type sourceKind int

const (
	sourceKindTimeBased sourceKind = iota
	sourceKindStill
)

func (kind sourceKind) String() string {
	switch kind {
	case sourceKindStill:
		return "still"
	default:
		return "time-based"
	}
}
```

Add `Kind sourceKind` to `resolvedSourceMedia`:

```go
type resolvedSourceMedia struct {
	Original    string
	Path        string
	DisplayName string
	Temporary   bool
	Kind        sourceKind
	Cleanup     func() error
}
```

Set local source kind in `resolveSourceMediaWithOptions`:

```go
if !isHTTPPlatformSource(source) {
	return resolvedSourceMedia{
		Original:    source,
		Path:        source,
		DisplayName: filepath.Base(source),
		Kind:        classifyLocalSourceKind(source),
		Cleanup:     func() error { return nil },
	}, nil
}
```

Set platform URLs as time-based after download:

```go
return resolvedSourceMedia{
	Original:    source,
	Path:        finalPath,
	DisplayName: displayName,
	Temporary:   true,
	Kind:        sourceKindTimeBased,
	Cleanup:     cleanup,
}, nil
```

Add helpers near `isHTTPPlatformSource`:

```go
func classifyLocalSourceKind(source string) sourceKind {
	if isStillSourcePath(source) {
		return sourceKindStill
	}
	return sourceKindTimeBased
}

func isStillSourcePath(source string) bool {
	switch strings.ToLower(filepath.Ext(source)) {
	case ".png", ".jpg", ".jpeg":
		return true
	default:
		return false
	}
}
```

- [ ] **Step 4: Run source tests and verify they pass**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestResolveSourceMediaClassifies|TestResolveSourceMediaLocalBypassesYTDLP|TestResolveSourceMediaDownloadsHTTPSSource'
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/cli/source.go packages/core/internal/cli/source_test.go
git commit --no-gpg-sign -m "feat: classify still image sources"
```

---

### Task 2: Update CLI Help and Parser Wording

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Write failing parser/help wording tests**

Add this test to `packages/core/internal/cli/cli_test.go`:

```go
func TestHelpTextMentionsStillImageSources(t *testing.T) {
	help := HelpText()
	for _, want := range []string{
		"<source> may be a local video file, local still image, or an HTTP(S) platform URL.",
		"Still image sources can be probed and exported, but not played.",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q in:\n%s", want, help)
		}
	}
}
```

Update existing tests that assert missing input wording only if they check for `video input`.

- [ ] **Step 2: Run the help test and verify it fails**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run TestHelpTextMentionsStillImageSources
```

Expected: fail because help still says local video file or HTTP(S) platform URL.

- [ ] **Step 3: Update help and parser error wording**

In `packages/core/internal/cli/cli.go`, change the `Source:` section in `HelpText()` to:

```text
Source:
  <source> may be a local video file, local still image, or an HTTP(S) platform URL.
  Still image sources can be probed and exported, but not played.
```

Change `parseInputCommand` missing and duplicate input errors from video-specific wording to source wording:

```go
if len(args) < 2 {
	return Command{}, fmt.Errorf("%s requires a source input", args[0])
}
```

```go
if inputPath != "" {
	return Command{}, fmt.Errorf("%s accepts exactly one source input", args[0])
}
```

```go
if inputPath == "" {
	return Command{}, fmt.Errorf("%s requires a source input", args[0])
}
```

Change unsupported protocol wording for `play` and `probe`:

```go
return Command{}, fmt.Errorf("%s accepts local source file paths or HTTP(S) platform URLs only", args[0])
```

Change export missing/protocol wording:

```go
return Command{}, fmt.Errorf("export requires a source input and output path")
```

```go
return Command{}, fmt.Errorf("export accepts local source file paths or HTTP(S) platform URLs only")
```

- [ ] **Step 4: Run CLI parser tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestHelpTextMentionsStillImageSources|TestParseMissingInput|TestParseExportMissingOutput|TestParseRejectsUnsupportedProtocolInputs'
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
git commit --no-gpg-sign -m "docs: describe still image source inputs"
```

---

### Task 3: Print Clean Probe Output for Still Sources

**Files:**
- Modify: `packages/core/internal/cli/probe.go`
- Modify: `packages/core/internal/cli/probe_test.go`

- [ ] **Step 1: Write failing still-source probe output tests**

Add this test to `packages/core/internal/cli/probe_test.go`:

```go
func TestPrintProbeInfoForStillSource(t *testing.T) {
	var out bytes.Buffer
	printProbeInfo(&out, probeOutput{
		OriginalSource:  "poster.png",
		SourceKind:      sourceKindStill,
		Width:           800,
		Height:          600,
		FPS:             25,
		FrameCount:      0,
		DurationSeconds: 0,
		HasAudio:        false,
		RenderCols:      120,
		RenderRows:      40,
	})
	got := out.String()
	for _, want := range []string{
		"input: poster.png\n",
		"image: 800x600\n",
		"audio: no\n",
		"render-grid: 120x40 (sample terminal 120x40)\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("probe output missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{
		"video:",
		"fps:",
		"frames:",
		"duration:",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("still probe output contains %q in:\n%s", unwanted, got)
		}
	}
}

func TestRunProbeCarriesStillSourceKindToOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := runProbeWithOptions(context.Background(), "poster.png", &stdout, io.Discard, probeRunnerOptions{
		Probe: func(ctx context.Context, path string) (media.Info, error) {
			if path != "poster.png" {
				t.Fatalf("probe path = %q, want local still path", path)
			}
			return media.Info{
				Width:  800,
				Height: 600,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("runProbeWithOptions returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "image: 800x600\n") {
		t.Fatalf("stdout missing image metadata:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "video:") {
		t.Fatalf("stdout printed video metadata for still source:\n%s", stdout.String())
	}
}
```

Add `io` to the import list in `probe_test.go` if it is not already present.

- [ ] **Step 2: Run probe tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestPrintProbeInfoForStillSource|TestRunProbeCarriesStillSourceKindToOutput'
```

Expected: fail to compile because `probeOutput.SourceKind` does not exist, or fail because still output still prints video fields.

- [ ] **Step 3: Add source kind to probe output**

In `packages/core/internal/cli/probe.go`, add `SourceKind sourceKind` to `probeOutput`:

```go
type probeOutput struct {
	OriginalSource      string
	ResolvedDisplayName string
	SourceKind          sourceKind
	Width               int
	Height              int
	FPS                 float64
	FrameCount          int
	DurationSeconds     float64
	HasAudio            bool
	RenderCols          int
	RenderRows          int
}
```

Pass it from `runProbeWithOptions`:

```go
printProbeInfo(stdout, probeOutput{
	OriginalSource:      resolved.Original,
	ResolvedDisplayName: resolvedDisplayName(resolved),
	SourceKind:          resolved.Kind,
	Width:               info.Width,
	Height:              info.Height,
	FPS:                 info.FPS,
	FrameCount:          info.FrameCount,
	DurationSeconds:     info.DurationSeconds,
	HasAudio:            info.HasAudio,
	RenderCols:          grid.Cols,
	RenderRows:          grid.Rows,
})
```

- [ ] **Step 4: Branch probe formatting by source kind**

Update `printProbeInfo`:

```go
func printProbeInfo(w io.Writer, output probeOutput) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "input: %s\n", output.OriginalSource)
	if output.ResolvedDisplayName != "" {
		fmt.Fprintf(w, "resolved-source: %s\n", output.ResolvedDisplayName)
	}
	if output.SourceKind == sourceKindStill {
		fmt.Fprintf(w, "image: %dx%d\n", output.Width, output.Height)
	} else {
		fmt.Fprintf(w, "video: %dx%d\n", output.Width, output.Height)
		fmt.Fprintf(w, "fps: %.3f\n", output.FPS)
		fmt.Fprintf(w, "frames: %d\n", output.FrameCount)
		fmt.Fprintf(w, "duration: %.3fs\n", output.DurationSeconds)
	}
	if output.HasAudio {
		fmt.Fprintln(w, "audio: yes")
	} else {
		fmt.Fprintln(w, "audio: no")
	}
	fmt.Fprintf(w, "render-grid: %dx%d (sample terminal 120x40)\n", output.RenderCols, output.RenderRows)
}
```

- [ ] **Step 5: Run probe tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestPrintProbeInfo|TestRunProbe'
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/cli/probe.go packages/core/internal/cli/probe_test.go
git commit --no-gpg-sign -m "feat: probe still image sources"
```

---

### Task 4: Reject Still Sources in Playback

**Files:**
- Modify: `packages/core/internal/cli/play.go`
- Modify: `packages/core/internal/cli/play_test.go`

- [ ] **Step 1: Write failing playback rejection test**

Add this test to `packages/core/internal/cli/play_test.go`:

```go
func TestRunPlayRejectsStillSourceBeforeProbe(t *testing.T) {
	probeCalled := false
	err := runPlayWithOptions(context.Background(), "poster.png", os.Stdin, io.Discard, io.Discard, PlayOptions{}, playRunnerOptions{
		Probe: func(ctx context.Context, path string) (media.Info, error) {
			probeCalled = true
			return media.Info{}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "still image sources cannot be played") {
		t.Fatalf("error = %v, want still source playback rejection", err)
	}
	if probeCalled {
		t.Fatal("probe was called for still source playback; want rejection before probe")
	}
}
```

- [ ] **Step 2: Run playback test and verify it fails**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run TestRunPlayRejectsStillSourceBeforeProbe
```

Expected: fail because `runPlayWithOptions` probes and continues instead of rejecting still sources.

- [ ] **Step 3: Reject still sources after resolution**

In `packages/core/internal/cli/play.go`, after `defer resolved.Cleanup()` and before `inputPath = resolved.Path`, add:

```go
if resolved.Kind == sourceKindStill {
	return fmt.Errorf("still image sources cannot be played; use mojify export <source> <output> instead")
}
```

- [ ] **Step 4: Run playback tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestRunPlayRejectsStillSourceBeforeProbe|TestRunPlayResolvesPlatformURLBeforeProbeAndCleansUp|TestRunPlayRejectsMissingYTDLPForPlatformURL'
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go
git commit --no-gpg-sign -m "fix: reject still images for playback"
```

---

### Task 5: Validate Still-Source Export Contract

**Files:**
- Modify: `packages/core/internal/cli/export.go`
- Modify: `packages/core/internal/cli/export_test.go`

- [ ] **Step 1: Write failing still-source export handoff test**

Add this test to `packages/core/internal/cli/export_test.go`:

```go
func TestRunExportAllowsStillSourceToSingleFrameOutputs(t *testing.T) {
	exportErr := errors.New("stop after export handoff")
	var gotInputPath string
	var gotInputLabel string

	err := runExportWithOptions(context.Background(), "poster.png", "out.ansi", io.Discard, ExportOptions{
		Width:     80,
		Overwrite: true,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotInputPath = inputPath
			gotInputLabel = options.InputLabel
			if outputPath != "out.ansi" {
				t.Fatalf("outputPath = %q, want out.ansi", outputPath)
			}
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if gotInputPath != "poster.png" {
		t.Fatalf("inputPath = %q, want still source path", gotInputPath)
	}
	if gotInputLabel != "poster.png" {
		t.Fatalf("InputLabel = %q, want original still source", gotInputLabel)
	}
}
```

Expected today: this may already pass because no still-specific validation exists. Keep it because it locks the allowed path.

- [ ] **Step 2: Write failing still-source export rejection tests**

Add these tests to `packages/core/internal/cli/export_test.go`:

```go
func TestRunExportRejectsStillSourceToTimeBasedOutput(t *testing.T) {
	err := runExportWithOptions(context.Background(), "poster.png", "out.mp4", io.Discard, ExportOptions{}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			t.Fatal("export handoff should not run for still source to time-based output")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "still image sources can only export single-frame outputs") {
		t.Fatalf("error = %v, want still source output-family rejection", err)
	}
}

func TestRunExportRejectsAtForStillSource(t *testing.T) {
	err := runExportWithOptions(context.Background(), "poster.jpg", "out.txt", io.Discard, ExportOptions{
		HasAt:     true,
		AtSeconds: 1,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			t.Fatal("export handoff should not run for still source with --at")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "export --at is not valid for still image sources") {
		t.Fatalf("error = %v, want still source --at rejection", err)
	}
}

func TestRunExportRejectsDurationForStillSourceToTimeBasedOutput(t *testing.T) {
	err := runExportWithOptions(context.Background(), "poster.jpeg", "out.gif", io.Discard, ExportOptions{
		HasDuration:     true,
		DurationSeconds: 2,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			t.Fatal("export handoff should not run for still source with --duration")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "export --duration is not valid for still image sources") {
		t.Fatalf("error = %v, want still source --duration rejection", err)
	}
}
```

- [ ] **Step 3: Run export tests and verify rejection tests fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestRunExportAllowsStillSource|TestRunExportRejects.*StillSource'
```

Expected: rejection tests fail because still-source validation is not implemented.

- [ ] **Step 4: Implement still-source export validation helper**

In `packages/core/internal/cli/export.go`, add this helper:

```go
func validateStillSourceExport(source resolvedSourceMedia, outputPath string, options ExportOptions) error {
	if source.Kind != sourceKindStill {
		return nil
	}
	if options.HasAt {
		return fmt.Errorf("export --at is not valid for still image sources")
	}
	if options.HasDuration {
		return fmt.Errorf("export --duration is not valid for still image sources")
	}
	format, err := exporter.ResolveOutputFormat(outputPath)
	if err != nil {
		return err
	}
	if !format.SingleFrame {
		return fmt.Errorf("still image sources can only export single-frame outputs: .png, .jpg, .jpeg, .txt, .ansi")
	}
	return nil
}
```

Add `fmt` to imports in `export.go`.

- [ ] **Step 5: Call validation before export handoff**

In `runExportWithOptions`, after `defer resolved.Cleanup()` and before selecting `exportFn`, add:

```go
if err := validateStillSourceExport(resolved, outputPath, options); err != nil {
	return err
}
```

- [ ] **Step 6: Run export tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestRunExportAllowsStillSource|TestRunExportRejects.*StillSource|TestRunExportPassesTimeOptionsAndUsesGeneralExporter|TestRunExportUsesOriginalURLForProgressAndResolvedPathForExport'
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go
git commit --no-gpg-sign -m "feat: export still image sources"
```

---

### Task 6: Add Still Source QA Fixture and Export Smoke

**Files:**
- Modify: `scripts/generate-qa-clips.sh`
- Modify: `scripts/export-qa.sh`

- [ ] **Step 1: Add generated still fixture**

In `scripts/generate-qa-clips.sh`, after the existing video fixtures, add:

```bash
ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "testsrc2=size=320x180:rate=1:duration=1" \
  -frames:v 1 \
  dist/qa/still-source.png
```

Add it to the final printed list:

```bash
printf '  dist/qa/still-source.png\n'
```

- [ ] **Step 2: Add still-source outputs to export QA**

In `scripts/export-qa.sh`, add variables near the synthetic source variables:

```bash
still_source="dist/qa/still-source.png"
still_source_png="${export_dir}/still-source-output.png"
still_source_jpg="${export_dir}/still-source-output.jpg"
still_source_txt="${export_dir}/still-source-output.txt"
still_source_ansi="${export_dir}/still-source-output.ansi"
```

After the synthetic source existence check, add:

```bash
if [[ ! -f "${still_source}" ]]; then
  printf 'Missing %s. Run `bun run qa:clips` first.\n' "${still_source}" >&2
  exit 1
fi
```

- [ ] **Step 3: Add still-source export smoke commands**

After the existing synthetic video-source export commands, add:

```bash
printf '\nExporting still source across single-frame formats...\n'
./bin/mojify export --overwrite --width 320 "${still_source}" "${still_source_png}"
./bin/mojify export --overwrite --width 320 "${still_source}" "${still_source_jpg}"
./bin/mojify export --overwrite --width 80 "${still_source}" "${still_source_txt}"
./bin/mojify export --overwrite --width 80 "${still_source}" "${still_source_ansi}"
```

After existing synthetic checks, add:

```bash
check_video_width "${still_source_png}" "320"
check_video_width "${still_source_jpg}" "320"
require_nonempty_file "${still_source_txt}"
require_nonempty_file "${still_source_ansi}"
```

- [ ] **Step 4: Add still-source validation failures**

In the validation failure section, add:

```bash
expect_export_failure \
  "still-source-at" \
  ./bin/mojify export --overwrite --width 80 --at 1s "${still_source}" "${export_dir}/still-source-at.ansi"
expect_export_failure \
  "still-source-time-based" \
  ./bin/mojify export --overwrite --width 320 "${still_source}" "${export_dir}/still-source.mp4"
```

- [ ] **Step 5: Run QA scripts**

Run:

```bash
bun run build
bun run qa:clips
bun run qa:export
```

Expected: all generated video-source exports still pass; still-source PNG/JPG/TXT/ANSI outputs are non-empty or width-checked; still-source `--at` and time-based output failures are recorded under `dist/qa/export/`.

- [ ] **Step 6: Commit**

```bash
git add scripts/generate-qa-clips.sh scripts/export-qa.sh
git commit --no-gpg-sign -m "test: add still source export qa"
```

---

### Task 7: Update README and Export QA Docs

**Files:**
- Modify: `README.md`
- Modify: `docs/qa/export.md`
- Verify: `CONTEXT.md`

- [ ] **Step 1: Update README usage examples**

In `README.md`, under the existing export examples, add:

```markdown
Convert a still image into Mojify text:

```bash
mojify export --overwrite --width 80 ./poster.png ./dist/poster.ansi
```
```

In `What It Does`, change:

```markdown
Mojify accepts local video files and yt-dlp-compatible platform URLs as source media.
```

to:

```markdown
Mojify accepts local video files, local still images, and yt-dlp-compatible platform URLs as source media.
```

Add a current capability bullet:

```markdown
- Local still-image probing and single-frame export
```

- [ ] **Step 2: Update export QA docs**

In `docs/qa/export.md`, add this paragraph after supported formats:

```markdown
Local `.png`, `.jpg`, and `.jpeg` files are supported as still sources for single-frame outputs only. A still source can export to `.png`, `.jpg`, `.jpeg`, `.txt`, or `.ansi`; it cannot export to video or animated outputs.
```

Update timestamp text:

```markdown
`--at <timestamp>` is valid for time-based source media. It is rejected for still sources because still sources have no timeline.
```

Update expected generated output list with:

```markdown
- `dist/qa/still-source.png`
- `dist/qa/export/still-source-output.png`
- `dist/qa/export/still-source-output.jpg`
- `dist/qa/export/still-source-output.txt`
- `dist/qa/export/still-source-output.ansi`
```

Add checklist items:

```markdown
- Still source export writes PNG, JPG, TXT, and ANSI outputs.
- Still source export rejects `--at`.
- Still source export rejects video and animated outputs.
```

- [ ] **Step 3: Verify glossary terms remain present**

Run:

```bash
rg -n "Still source|Still source export|Still source timestamp rejection" CONTEXT.md
```

Expected: all three terms are present.

- [ ] **Step 4: Commit docs**

```bash
git add README.md docs/qa/export.md CONTEXT.md
git commit --no-gpg-sign -m "docs: document still source input"
```

---

### Task 8: Final Verification

**Files:**
- Verify all modified files.

- [ ] **Step 1: Run formatting**

Run:

```bash
bun run fmt:check
```

Expected: no output after the script line and exit code 0.

- [ ] **Step 2: Run Go tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...
```

Expected: all Go packages pass.

- [ ] **Step 3: Run monorepo tests**

Run:

```bash
bun run test
```

Expected: Turbo reports successful `@mojify/core:test` and `@mojify/scripts:test`.

- [ ] **Step 4: Run build**

Run:

```bash
bun run build
```

Expected: `bin/mojify` is rebuilt successfully.

- [ ] **Step 5: Run export QA**

Run:

```bash
bun run qa:clips
bun run qa:export
```

Expected: QA exports all video-source representative outputs and all still-source single-frame outputs; expected validation failures are recorded for unsupported `.webp`, duration with single-frame video-source output, still-source `--at`, and still-source time-based output.

- [ ] **Step 6: Run manual CLI smoke**

Run:

```bash
./bin/mojify probe dist/qa/still-source.png
./bin/mojify export --overwrite --width 80 dist/qa/still-source.png dist/qa/export/manual-still.ansi
./bin/mojify play dist/qa/still-source.png
```

Expected:

- `probe` prints `image: 320x180`, `audio: no`, and render-grid metadata.
- `export` writes a non-empty ANSI file.
- `play` fails with `still image sources cannot be played; use mojify export <source> <output> instead`.

- [ ] **Step 7: Run whitespace check**

Run:

```bash
git diff --check
```

Expected: no output and exit code 0.

- [ ] **Step 8: Review final diff**

Run:

```bash
git status -sb
git diff --stat
```

Expected: changes are limited to CLI source/probe/play/export handling, QA scripts, README, export QA docs, and `CONTEXT.md`.

---

## Self-Review

- Spec coverage:
  - Local still images as `SOURCE`: Task 1.
  - `probe` support: Task 3.
  - `play` rejection: Task 4.
  - Single-frame-only `export`: Task 5.
  - `--at` and `--duration` rejection: Task 5.
  - Direct HTTP image URLs out of scope: preserved by existing HTTP platform URL handling and documented in the locked contract.
  - QA coverage: Task 6 and Task 8.
  - Docs: Task 7.
- Placeholder scan:
  - No placeholder implementation slots are intentionally left in this plan.
- Type consistency:
  - `sourceKind`, `sourceKindTimeBased`, and `sourceKindStill` are introduced in Task 1 and reused consistently in later tasks.
  - `probeOutput.SourceKind` is introduced before the probe formatter uses it.
  - Still-source export validation uses the existing `exporter.ResolveOutputFormat` and `OutputFormat.SingleFrame` contract.
