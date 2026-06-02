# Export Throughput Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `mojify export` faster by measuring export stage timings, using a faster MP4 encoder preset, and parallelizing Mojify render/raster work without breaking output order, source audio, or honest progress.

**Architecture:** Keep FFmpeg decode and encode as separate CLI processes. Replace the serial render/raster/write loop with a bounded in-process worker pool: decoded frames are indexed in read order, workers render and rasterize frames concurrently, and a single ordered writer sends raw RGB frames to FFmpeg stdin in exact frame order. Export progress remains tied to frames successfully written to the encoder, not frames merely rendered by workers.

**Tech Stack:** Go, FFmpeg CLI, libx264, existing Mojify renderer/rasterizer, `runtime.GOMAXPROCS`, Bun/Turbo verification scripts.

## Implementation Status

Implemented on `feat/export-throughput-hardening` with one combined branch commit instead of per-task commits. Verification completed:

- `bun run fmt:check`
- `GOCACHE=/private/tmp/mojify-gocache go mod tidy -diff`
- `GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...`
- `bun run test`
- `bun run typecheck`
- `bun run build`
- `GOCACHE=/private/tmp/mojify-gocache go test -race ./packages/core/internal/exporter`
- `bun run qa:clips`
- `bun run qa:export`
- `/usr/bin/time -p ./bin/mojify export --overwrite --stats --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4`
- `git diff --check`

The stats smoke reported 8 export workers, 120 read/rendered/rasterized/written frames, `elapsed: 47.422084ms`, and no ETA/time-remaining output.

---

## Decisions Already Made

- This is the next roadmap stage: **Export throughput hardening**.
- Speed work must not change visual frame ordering, source audio behavior, or MP4 output validity.
- Progress remains honest: it advances when frames are written to the encoder, not when worker jobs complete out of order.
- No ETA or time remaining is added.
- Start with timing metrics so we can tell whether Mojify render/raster or FFmpeg encode is the bottleneck.
- Use bounded parallelism. Do not pre-render the whole video.
- Each render/raster worker should own its rasterizer/font face. Do not assume `font.Face` is safe for concurrent use.
- Ordered output is mandatory because FFmpeg rawvideo stdin expects frames in display order.
- Hardware encoders are out of scope for this stage.

## File Structure

- Modify: `CONTEXT.md`
  - Already adds the `Export throughput hardening` glossary term.
- Modify: `packages/core/internal/cli/cli.go`
  - Add export `--stats` and `--workers <n>` parsing/help.
- Modify: `packages/core/internal/cli/cli_test.go`
  - Lock new export flags and validation.
- Modify: `packages/core/internal/cli/export.go`
  - Pass export stats/workers options into exporter.
- Create: `packages/core/internal/exporter/metrics.go`
  - Track export elapsed time, worker count, rendered/rasterized/written frame counts, and average stage timings.
- Create: `packages/core/internal/exporter/metrics_test.go`
  - Lock summary formatting and timing math.
- Modify: `packages/core/internal/exporter/layout.go`
  - Add `Stats bool`, `Workers int`, and optional test clock to `Options`.
- Modify: `packages/core/internal/exporter/fonts/fonts.go`
  - Parse the embedded font once and create a fresh face per worker.
- Modify: `packages/core/internal/exporter/fonts/fonts_test.go`
  - Lock repeated face creation.
- Modify: `packages/core/internal/media/encode.go`
  - Add default libx264 `-preset veryfast`.
- Modify: `packages/core/internal/media/encode_test.go`
  - Lock default preset in FFmpeg args.
- Create: `packages/core/internal/exporter/pipeline.go`
  - Implement bounded parallel render/raster with ordered frame writes.
- Create: `packages/core/internal/exporter/pipeline_test.go`
  - Lock ordered writes, worker-count resolution, progress semantics, and error behavior.
- Modify: `packages/core/internal/exporter/export.go`
  - Replace serial loop with the pipeline, print stats when requested, and preserve current cleanup/error behavior.
- Modify: `docs/qa/export.md`
  - Add export throughput QA and stats/baseline guidance.
- Modify: `docs/superpowers/plans/2026-06-02-mojify-export-throughput-hardening.md`
  - Track implementation status.

---

### Task 1: Export Stats and Worker CLI Options

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`
- Modify: `packages/core/internal/cli/export.go`
- Modify: `packages/core/internal/exporter/layout.go`

- [ ] **Step 1: Add failing CLI tests for export `--stats` and `--workers`**

Append to `packages/core/internal/cli/cli_test.go`:

```go
func TestParseExportStatsAndWorkers(t *testing.T) {
	cmd, err := Parse([]string{
		"export",
		"--stats",
		"--workers", "6",
		"clip.mov",
		"clip.mp4",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !cmd.Export.Stats {
		t.Fatal("Export.Stats = false, want true")
	}
	if cmd.Export.Workers != 6 {
		t.Fatalf("Export.Workers = %d, want 6", cmd.Export.Workers)
	}
}

func TestParseExportRejectsDuplicateStats(t *testing.T) {
	_, err := Parse([]string{"export", "--stats", "clip.mov", "--stats", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for duplicate export --stats")
	}
}

func TestParseExportRejectsInvalidWorkers(t *testing.T) {
	for _, workers := range []string{"0", "-1", "many"} {
		_, err := Parse([]string{"export", "--workers", workers, "clip.mov", "clip.mp4"})
		if err == nil {
			t.Fatalf("Parse returned nil error for invalid --workers %q", workers)
		}
	}
}
```

In `TestHelpTextMentionsCommands`, add:

```go
		"--stats",
		"--workers <n>",
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail because `ExportOptions.Stats`, `ExportOptions.Workers`, and export flag parsing are missing.

- [ ] **Step 3: Add export option fields**

Modify `packages/core/internal/cli/cli.go`:

```go
type ExportOptions struct {
	Width     int
	FPS       float64
	Bitrate   string
	Overwrite bool
	Stats     bool
	Workers   int
}
```

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
}
```

- [ ] **Step 4: Parse export `--stats` and `--workers`**

In `parseExportCommand`, add a `seenStats := false` local next to `seenOverwrite`.

Add these cases inside the export flag switch:

```go
		case "--stats":
			if seenStats {
				return Command{}, fmt.Errorf("export accepts --stats only once")
			}
			seenStats = true
			options.Stats = true
		case "--workers":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --workers")
			}
			i++
			workers, err := strconv.Atoi(args[i])
			if err != nil || workers <= 0 {
				return Command{}, fmt.Errorf("export requires --workers to be greater than 0")
			}
			options.Workers = workers
```

Update the help text export options:

```text
  --stats            Print export timing stats after completion
  --workers <n>      Render and rasterize with n workers
```

- [ ] **Step 5: Pass options through `RunExport`**

Modify `packages/core/internal/cli/export.go` so the exporter options include:

```go
		Stats:               options.Stats,
		Workers:             options.Workers,
```

- [ ] **Step 6: Run CLI tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go packages/core/internal/cli/export.go packages/core/internal/exporter/layout.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: pass.

- [ ] **Step 7: Commit**

Run:

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go packages/core/internal/cli/export.go packages/core/internal/exporter/layout.go
git commit --no-gpg-sign -m "feat: add export throughput options"
```

---

### Task 2: Export Metrics

**Files:**
- Create: `packages/core/internal/exporter/metrics.go`
- Create: `packages/core/internal/exporter/metrics_test.go`
- Modify: `packages/core/internal/exporter/export.go`

- [ ] **Step 1: Add failing metrics tests**

Create `packages/core/internal/exporter/metrics_test.go`:

```go
package exporter

import (
	"strings"
	"testing"
	"time"
)

func TestExportMetricsSummary(t *testing.T) {
	metrics := newExportMetrics(4, fakeExportClock(time.Unix(0, 0)))
	metrics.Start()
	metrics.RecordRead(2 * time.Millisecond)
	metrics.RecordRender(3 * time.Millisecond)
	metrics.RecordRasterize(5 * time.Millisecond)
	metrics.RecordWrite(7 * time.Millisecond)
	metrics.RecordWrite(9 * time.Millisecond)
	metrics.Finish()

	summary := metrics.Summary()
	for _, want := range []string{
		"export stats",
		"workers: 4",
		"written frames: 2",
		"avg read time: 2ms",
		"avg render time: 3ms",
		"avg rasterize time: 5ms",
		"avg encoder write time: 8ms",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("Summary missing %q in:\n%s", want, summary)
		}
	}
}

func TestExportMetricsSummaryHandlesNoFrames(t *testing.T) {
	metrics := newExportMetrics(1, fakeExportClock(time.Unix(0, 0)))
	metrics.Start()
	metrics.Finish()

	summary := metrics.Summary()
	for _, want := range []string{
		"workers: 1",
		"written frames: 0",
		"avg read time: 0s",
		"avg render time: 0s",
		"avg rasterize time: 0s",
		"avg encoder write time: 0s",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("Summary missing %q in:\n%s", want, summary)
		}
	}
}

type fakeExportClock time.Time

func (c fakeExportClock) Now() time.Time {
	return time.Time(c)
}
```

- [ ] **Step 2: Run metrics tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail because `newExportMetrics` is missing.

- [ ] **Step 3: Implement export metrics**

Create `packages/core/internal/exporter/metrics.go`:

```go
package exporter

import (
	"fmt"
	"sync"
	"time"
)

type exportClock interface {
	Now() time.Time
}

type realExportClock struct{}

func (realExportClock) Now() time.Time {
	return time.Now()
}

type exportMetrics struct {
	mu sync.Mutex

	workers int
	clock   exportClock

	startedAt  time.Time
	finishedAt time.Time

	readFrames      int
	renderedFrames  int
	rasterizedFrames int
	writtenFrames   int

	readTime      time.Duration
	renderTime    time.Duration
	rasterizeTime time.Duration
	writeTime     time.Duration
}

func newExportMetrics(workers int, clock exportClock) *exportMetrics {
	if workers < 1 {
		workers = 1
	}
	if clock == nil {
		clock = realExportClock{}
	}
	return &exportMetrics{workers: workers, clock: clock}
}

func (m *exportMetrics) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedAt = m.clock.Now()
	m.finishedAt = time.Time{}
}

func (m *exportMetrics) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finishedAt = m.clock.Now()
}

func (m *exportMetrics) RecordRead(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readFrames++
	m.readTime += duration
}

func (m *exportMetrics) RecordRender(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renderedFrames++
	m.renderTime += duration
}

func (m *exportMetrics) RecordRasterize(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rasterizedFrames++
	m.rasterizeTime += duration
}

func (m *exportMetrics) RecordWrite(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenFrames++
	m.writeTime += duration
}

func (m *exportMetrics) Summary() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	elapsed := time.Duration(0)
	if !m.startedAt.IsZero() && !m.finishedAt.IsZero() {
		elapsed = m.finishedAt.Sub(m.startedAt)
	}

	return fmt.Sprintf(
		"export stats\nworkers: %d\nread frames: %d\nrendered frames: %d\nrasterized frames: %d\nwritten frames: %d\nelapsed: %s\navg read time: %s\navg render time: %s\navg rasterize time: %s\navg encoder write time: %s\n",
		m.workers,
		m.readFrames,
		m.renderedFrames,
		m.rasterizedFrames,
		m.writtenFrames,
		elapsed,
		averageDuration(m.readTime, m.readFrames),
		averageDuration(m.renderTime, m.renderedFrames),
		averageDuration(m.rasterizeTime, m.rasterizedFrames),
		averageDuration(m.writeTime, m.writtenFrames),
	)
}

func averageDuration(total time.Duration, count int) time.Duration {
	if count <= 0 {
		return 0
	}
	return total / time.Duration(count)
}
```

- [ ] **Step 4: Run metrics tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/metrics.go packages/core/internal/exporter/metrics_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Wire stats printing after export completion**

In `ExportMP4`, create metrics after resolving worker count in Task 4. For now, add the `printExportStats` helper at the bottom of `packages/core/internal/exporter/export.go`:

```go
func printExportStats(stderr io.Writer, options Options, metrics *exportMetrics) {
	if !options.Stats || stderr == nil || metrics == nil {
		return
	}
	fmt.Fprintln(stderr)
	fmt.Fprint(stderr, metrics.Summary())
}
```

Task 4 will call this helper after `progress.Complete(outputPath)`.

- [ ] **Step 6: Commit**

Run:

```bash
git add packages/core/internal/exporter/metrics.go packages/core/internal/exporter/metrics_test.go packages/core/internal/exporter/export.go
git commit --no-gpg-sign -m "feat: add export timing metrics"
```

---

### Task 3: Faster Encoder Preset

**Files:**
- Modify: `packages/core/internal/media/encode.go`
- Modify: `packages/core/internal/media/encode_test.go`

- [ ] **Step 1: Add failing encoder preset test**

Append to `packages/core/internal/media/encode_test.go`:

```go
func TestMP4EncodeArgsUsesVeryfastPreset(t *testing.T) {
	args, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "clip.mov",
		OutputPath: "out.mp4",
		Width:      320,
		Height:     184,
		FPS:        24,
	})
	if err != nil {
		t.Fatalf("MP4EncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-preset", "veryfast") {
		t.Fatalf("args missing -preset veryfast: %#v", args)
	}
}
```

- [ ] **Step 2: Run media tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: fail because the encoder args do not include `-preset veryfast`.

- [ ] **Step 3: Add the default preset**

Modify `packages/core/internal/media/encode.go`:

```go
const defaultMP4VideoPreset = "veryfast"
```

Add this pair after `"-c:v", "libx264"` in `MP4EncodeArgs`:

```go
		"-preset", defaultMP4VideoPreset,
```

- [ ] **Step 4: Run media tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/media/encode.go packages/core/internal/media/encode_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: pass.

- [ ] **Step 5: Commit**

Run:

```bash
git add packages/core/internal/media/encode.go packages/core/internal/media/encode_test.go
git commit --no-gpg-sign -m "feat: use faster mp4 encoder preset"
```

---

### Task 4: Worker-Safe Font Faces

**Files:**
- Modify: `packages/core/internal/exporter/fonts/fonts.go`
- Modify: `packages/core/internal/exporter/fonts/fonts_test.go`

- [ ] **Step 1: Add failing repeated-face test**

Append to `packages/core/internal/exporter/fonts/fonts_test.go`:

```go
func TestDefaultFaceReturnsDistinctFaces(t *testing.T) {
	first, err := DefaultFace()
	if err != nil {
		t.Fatalf("DefaultFace first call returned error: %v", err)
	}
	second, err := DefaultFace()
	if err != nil {
		t.Fatalf("DefaultFace second call returned error: %v", err)
	}
	if first == nil || second == nil {
		t.Fatal("DefaultFace returned nil face")
	}
	if first == second {
		t.Fatal("DefaultFace returned the same face instance twice")
	}
}
```

- [ ] **Step 2: Run font tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter/fonts
```

Expected: pass before and after implementation; this test locks the behavior needed by parallel workers.

- [ ] **Step 3: Parse embedded font once**

Modify `packages/core/internal/exporter/fonts/fonts.go`:

```go
package fonts

import (
	_ "embed"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed Mx437_IBM_BIOS.ttf
var mx437IBMBIOS []byte

var (
	defaultFontOnce sync.Once
	defaultFont     *opentype.Font
	defaultFontErr  error
)

func DefaultFace() (font.Face, error) {
	defaultFontOnce.Do(func() {
		defaultFont, defaultFontErr = opentype.Parse(mx437IBMBIOS)
	})
	if defaultFontErr != nil {
		return nil, defaultFontErr
	}
	return opentype.NewFace(defaultFont, &opentype.FaceOptions{
		Size:    8,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
```

- [ ] **Step 4: Run font tests and exporter tests**

Run:

```bash
gofmt -w packages/core/internal/exporter/fonts/fonts.go packages/core/internal/exporter/fonts/fonts_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter/fonts ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Commit**

Run:

```bash
git add packages/core/internal/exporter/fonts/fonts.go packages/core/internal/exporter/fonts/fonts_test.go
git commit --no-gpg-sign -m "perf: prepare export font faces for workers"
```

---

### Task 5: Ordered Parallel Export Pipeline

**Files:**
- Create: `packages/core/internal/exporter/pipeline.go`
- Create: `packages/core/internal/exporter/pipeline_test.go`
- Modify: `packages/core/internal/exporter/export.go`

- [ ] **Step 1: Add failing worker-count tests**

Create `packages/core/internal/exporter/pipeline_test.go` with:

```go
package exporter

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestResolveExportWorkersUsesExplicitValue(t *testing.T) {
	if got := resolveExportWorkers(3); got != 3 {
		t.Fatalf("resolveExportWorkers(3) = %d, want 3", got)
	}
}

func TestResolveExportWorkersDefaultsToPositiveBound(t *testing.T) {
	got := resolveExportWorkers(0)
	if got < 1 {
		t.Fatalf("resolveExportWorkers(0) = %d, want positive", got)
	}
	if got > 8 {
		t.Fatalf("resolveExportWorkers(0) = %d, want at most 8", got)
	}
}
```

- [ ] **Step 2: Add failing ordered writer tests**

Append to `packages/core/internal/exporter/pipeline_test.go`:

```go
func TestWriteOrderedFrameResultsWritesInFrameOrder(t *testing.T) {
	results := make(chan exportFrameResult, 3)
	results <- exportFrameResult{Index: 1, RGB: []byte{1}}
	results <- exportFrameResult{Index: 0, RGB: []byte{0}}
	results <- exportFrameResult{Index: 2, RGB: []byte{2}}
	close(results)

	writes := make([][]byte, 0, 3)
	written, err := writeOrderedFrameResults(context.Background(), results, func(data []byte) error {
		writes = append(writes, append([]byte(nil), data...))
		return nil
	}, nil, nil)
	if err != nil {
		t.Fatalf("writeOrderedFrameResults returned error: %v", err)
	}
	if written != 3 {
		t.Fatalf("written = %d, want 3", written)
	}
	if !reflect.DeepEqual(writes, [][]byte{{0}, {1}, {2}}) {
		t.Fatalf("writes = %#v, want ordered frame bytes", writes)
	}
}

func TestWriteOrderedFrameResultsReturnsResultError(t *testing.T) {
	wantErr := errors.New("rasterize failed")
	results := make(chan exportFrameResult, 1)
	results <- exportFrameResult{Index: 0, Err: wantErr}
	close(results)

	_, err := writeOrderedFrameResults(context.Background(), results, func([]byte) error {
		t.Fatal("writer should not be called for errored result")
		return nil
	}, nil, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestWriteOrderedFrameResultsReturnsWriterError(t *testing.T) {
	wantErr := errors.New("encoder write failed")
	results := make(chan exportFrameResult, 1)
	results <- exportFrameResult{Index: 0, RGB: []byte{0}}
	close(results)

	_, err := writeOrderedFrameResults(context.Background(), results, func([]byte) error {
		return wantErr
	}, nil, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}
```

- [ ] **Step 3: Add failing pipeline smoke test**

Append to `packages/core/internal/exporter/pipeline_test.go`:

```go
func TestRunExportFramePipelineWritesFramesAndUpdatesProgress(t *testing.T) {
	frames := []render.RGBFrame{
		render.NewRGBFrame(1, 1, []byte{1, 0, 0}),
		render.NewRGBFrame(1, 1, []byte{2, 0, 0}),
		render.NewRGBFrame(1, 1, []byte{3, 0, 0}),
	}
	next := 0
	writes := make([][]byte, 0, len(frames))
	progress := &countingProgress{}

	written, err := runExportFramePipeline(context.Background(), exportFramePipelineOptions{
		Workers: 2,
		ReadFrame: func() (render.RGBFrame, error) {
			if next == len(frames) {
				return render.RGBFrame{}, io.EOF
			}
			frame := frames[next]
			next++
			return frame, nil
		},
		ProcessFrame: func(index int, frame render.RGBFrame) ([]byte, error) {
			return []byte{byte(index)}, nil
		},
		WriteFrame: func(data []byte) error {
			writes = append(writes, append([]byte(nil), data...))
			return nil
		},
		Progress: progress,
	})
	if err != nil {
		t.Fatalf("runExportFramePipeline returned error: %v", err)
	}
	if written != 3 {
		t.Fatalf("written = %d, want 3", written)
	}
	if !reflect.DeepEqual(writes, [][]byte{{0}, {1}, {2}}) {
		t.Fatalf("writes = %#v, want ordered frame bytes", writes)
	}
	if !reflect.DeepEqual(progress.frames, []int{1, 2, 3}) {
		t.Fatalf("progress frames = %#v, want [1 2 3]", progress.frames)
	}
}

type countingProgress struct {
	frames []int
}

func (p *countingProgress) Frame(renderedFrames int) {
	p.frames = append(p.frames, renderedFrames)
}
```

- [ ] **Step 4: Run pipeline tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail because `resolveExportWorkers`, `exportFrameResult`, `writeOrderedFrameResults`, `runExportFramePipeline`, and `exportFramePipelineOptions` are missing.

- [ ] **Step 5: Implement ordered parallel pipeline**

Create `packages/core/internal/exporter/pipeline.go`:

```go
package exporter

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

type exportProgress interface {
	Frame(renderedFrames int)
}

type exportFramePipelineOptions struct {
	Workers      int
	ReadFrame    func() (render.RGBFrame, error)
	ProcessFrame func(index int, frame render.RGBFrame) ([]byte, error)
	WriteFrame   func([]byte) error
	Progress     exportProgress
	Metrics      *exportMetrics
	Clock        exportClock
}

type exportFrameJob struct {
	Index int
	Frame render.RGBFrame
}

type exportFrameResult struct {
	Index int
	RGB   []byte
	Err   error
}

func resolveExportWorkers(requested int) int {
	if requested > 0 {
		return requested
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		return 1
	}
	return min(workers, 8)
}

func runExportFramePipeline(ctx context.Context, options exportFramePipelineOptions) (int, error) {
	if options.ReadFrame == nil {
		return 0, fmt.Errorf("read frame function is required")
	}
	if options.ProcessFrame == nil {
		return 0, fmt.Errorf("process frame function is required")
	}
	if options.WriteFrame == nil {
		return 0, fmt.Errorf("write frame function is required")
	}

	workers := resolveExportWorkers(options.Workers)
	clock := options.Clock
	if clock == nil {
		clock = realExportClock{}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan exportFrameJob, workers*2)
	results := make(chan exportFrameResult, workers*2)
	readErr := make(chan error, 1)

	go readExportFrames(ctx, jobs, readErr, options.ReadFrame, options.Metrics, clock)

	var workerGroup sync.WaitGroup
	workerGroup.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer workerGroup.Done()
			for job := range jobs {
				start := clock.Now()
				rgb, err := options.ProcessFrame(job.Index, job.Frame)
				if options.Metrics != nil {
					options.Metrics.RecordRasterize(clock.Now().Sub(start))
				}
				select {
				case <-ctx.Done():
					return
				case results <- exportFrameResult{Index: job.Index, RGB: rgb, Err: err}:
				}
				if err != nil {
					return
				}
			}
		}()
	}

	go func() {
		workerGroup.Wait()
		close(results)
	}()

	written, writeErr := writeOrderedFrameResults(ctx, results, options.WriteFrame, options.Progress, options.Metrics)
	if writeErr != nil {
		cancel()
		return written, writeErr
	}
	if err := <-readErr; err != nil {
		cancel()
		return written, err
	}
	return written, nil
}

func readExportFrames(
	ctx context.Context,
	jobs chan<- exportFrameJob,
	readErr chan<- error,
	readFrame func() (render.RGBFrame, error),
	metrics *exportMetrics,
	clock exportClock,
) {
	defer close(jobs)
	index := 0
	for {
		start := clock.Now()
		frame, err := readFrame()
		if errors.Is(err, io.EOF) {
			readErr <- nil
			return
		}
		if err != nil {
			readErr <- fmt.Errorf("read decoded frame: %w", err)
			return
		}
		if metrics != nil {
			metrics.RecordRead(clock.Now().Sub(start))
		}
		select {
		case <-ctx.Done():
			readErr <- ctx.Err()
			return
		case jobs <- exportFrameJob{Index: index, Frame: frame}:
			index++
		}
	}
}

func writeOrderedFrameResults(
	ctx context.Context,
	results <-chan exportFrameResult,
	writeFrame func([]byte) error,
	progress exportProgress,
	metrics *exportMetrics,
) (int, error) {
	next := 0
	written := 0
	pending := map[int]exportFrameResult{}
	for {
		result, ok := pending[next]
		if !ok {
			var open bool
			select {
			case <-ctx.Done():
				return written, ctx.Err()
			case result, open = <-results:
				if !open {
					if len(pending) != 0 {
						return written, fmt.Errorf("missing frame result %d", next)
					}
					return written, nil
				}
			}
			if result.Index != next {
				pending[result.Index] = result
				continue
			}
		} else {
			delete(pending, next)
		}

		if result.Err != nil {
			return written, result.Err
		}
		start := time.Now()
		if err := writeFrame(result.RGB); err != nil {
			return written, fmt.Errorf("write encoder frame: %w", err)
		}
		if metrics != nil {
			metrics.RecordWrite(time.Since(start))
		}
		written++
		if progress != nil {
			progress.Frame(written)
		}
		next++
	}
}
```

After writing this file, fix the import list by adding `errors` and removing unused imports if Go reports them. The final import block should include `context`, `errors`, `fmt`, `io`, `runtime`, `sync`, and `time`.

On write or processing errors, the pipeline should cancel its local context and return promptly. `ExportMP4` already owns decoder and encoder process cleanup through defers, so the integration must keep those cleanup defers active when pipeline errors return.

- [ ] **Step 6: Run pipeline tests**

Run:

```bash
gofmt -w packages/core/internal/exporter/pipeline.go packages/core/internal/exporter/pipeline_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 7: Commit pipeline primitives**

Run:

```bash
git add packages/core/internal/exporter/pipeline.go packages/core/internal/exporter/pipeline_test.go
git commit --no-gpg-sign -m "feat: add ordered export pipeline"
```

---

### Task 6: Integrate Pipeline Into MP4 Export

**Files:**
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/exporter/pipeline.go`
- Modify: `packages/core/internal/exporter/pipeline_test.go`

- [ ] **Step 1: Add real frame processor helper**

In `packages/core/internal/exporter/pipeline.go`, add:

```go
func newExportFrameProcessor(layout Layout) (func(int, render.RGBFrame) ([]byte, error), error) {
	face, err := fonts.DefaultFace()
	if err != nil {
		return nil, fmt.Errorf("load export font: %w", err)
	}
	rasterizer := NewRasterizer(face)
	renderer := render.DefaultRenderer{}
	return func(_ int, rgbFrame render.RGBFrame) ([]byte, error) {
		charFrame := renderer.Render(rgbFrame, layout.Grid)
		return rasterizer.Rasterize(charFrame, layout)
	}, nil
}
```

Add the `fonts` import:

```go
	"github.com/jass/mojify/packages/core/internal/exporter/fonts"
```

- [ ] **Step 2: Replace serial loop in `ExportMP4`**

In `packages/core/internal/exporter/export.go`, remove these imports if they become unused:

```go
	"github.com/jass/mojify/packages/core/internal/exporter/fonts"
	"github.com/jass/mojify/packages/core/internal/render"
```

After starting the encoder, resolve workers and metrics:

```go
	workers := resolveExportWorkers(options.Workers)
	var metrics *exportMetrics
	if options.Stats {
		metrics = newExportMetrics(workers, exportMetricsClock(options))
		metrics.Start()
		defer metrics.Finish()
	}
```

Add this helper near the bottom of `export.go`:

```go
func exportMetricsClock(options Options) exportClock {
	if options.MetricsClock != nil {
		return exportClockFunc(options.MetricsClock)
	}
	return realExportClock{}
}

type exportClockFunc func() time.Time

func (f exportClockFunc) Now() time.Time {
	return f()
}
```

Add `time` to the import block for this helper.

Replace the serial render loop with:

```go
	processor, err := newExportFrameProcessor(layout)
	if err != nil {
		return err
	}
	renderedFrames, err := runExportFramePipeline(ctx, exportFramePipelineOptions{
		Workers: workers,
		ReadFrame: func() (render.RGBFrame, error) {
			return media.ReadRawFrame(decodePipe, layout.Grid.Cols, layout.Grid.Rows)
		},
		ProcessFrame: processor,
		WriteFrame: func(raw []byte) error {
			_, err := encodePipe.Write(raw)
			return err
		},
		Progress: progress,
		Metrics:  metrics,
		Clock:    exportMetricsClock(options),
	})
	if err != nil {
		return err
	}
```

Keep the existing decoder close/wait, encoder close/wait, `progress.AllFramesWritten(renderedFrames)`, `progress.Finalizing()`, and `progress.Complete(outputPath)` ordering. After `progress.Complete(outputPath)`, call:

```go
	printExportStats(stderr, options, metrics)
```

- [ ] **Step 3: Avoid double-counting raster timing**

Adjust `newExportFrameProcessor` so it records render and rasterize separately. Change its signature to:

```go
func newExportFrameProcessor(layout Layout, metrics *exportMetrics, clock exportClock) (func(int, render.RGBFrame) ([]byte, error), error)
```

Use:

```go
		renderStart := clock.Now()
		charFrame := renderer.Render(rgbFrame, layout.Grid)
		if metrics != nil {
			metrics.RecordRender(clock.Now().Sub(renderStart))
		}
		rasterStart := clock.Now()
		raw, err := rasterizer.Rasterize(charFrame, layout)
		if metrics != nil {
			metrics.RecordRasterize(clock.Now().Sub(rasterStart))
		}
		return raw, err
```

Remove the `metrics.RecordRasterize` call from `runExportFramePipeline`; the processor now owns render/raster timing.

- [ ] **Step 4: Run focused tests**

Run:

```bash
gofmt -w packages/core/internal/exporter/export.go packages/core/internal/exporter/pipeline.go packages/core/internal/exporter/pipeline_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [ ] **Step 5: Commit integration**

Run:

```bash
git add packages/core/internal/exporter/export.go packages/core/internal/exporter/pipeline.go packages/core/internal/exporter/pipeline_test.go
git commit --no-gpg-sign -m "feat: parallelize export rendering"
```

---

### Task 7: Export Throughput QA Docs

**Files:**
- Modify: `docs/qa/export.md`
- Modify: `docs/superpowers/plans/2026-06-02-mojify-export-throughput-hardening.md`

- [x] **Step 1: Add throughput QA section**

In `docs/qa/export.md`, after `Canonical Smoke`, add this Markdown:

````md
## Throughput QA

Use export stats to compare the current branch against the previous `main` baseline with the same source, output width, terminal, and machine load.

```bash
bun run qa:clips
bun run build
time ./bin/mojify export --overwrite --stats --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
```

Record ignored local notes under `dist/qa/export-throughput-after.md` when comparing a branch. Include:

- source file
- output width
- worker count from `export stats`
- elapsed wall-clock time from `time`
- `export stats` summary
- whether exported video/audio QA still passes
````

- [x] **Step 2: Add throughput checklist items**

Append to the checklist:

```md
- Export stats print when `--stats` is passed.
- Export stats do not print by default.
- Parallel export preserves ordered video frames and source audio behavior.
- Throughput comparisons use the same source file, width, machine, and terminal conditions.
```

- [ ] **Step 3: Commit docs**

Run:

```bash
git add docs/qa/export.md docs/superpowers/plans/2026-06-02-mojify-export-throughput-hardening.md
git commit --no-gpg-sign -m "docs: add export throughput qa"
```

---

### Task 8: Final Verification

**Files:**
- Read: `git status --short --branch`
- Read: `git log --oneline --decorate -5`

- [ ] **Step 1: Run formatting checks**

Run:

```bash
bun run fmt:check
```

Expected: pass.

- [ ] **Step 2: Run module tidy diff**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go mod tidy -diff
```

Expected: no output and exit code 0.

- [ ] **Step 3: Run full tests**

Run:

```bash
bun run test
```

Expected: pass. If the sandbox blocks the default Go build cache, rerun with elevated permission rather than changing repository files.

- [ ] **Step 4: Run typecheck and build**

Run:

```bash
bun run typecheck
bun run build
```

Expected: both pass.

- [ ] **Step 5: Run race test for exporter**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -race ./packages/core/internal/exporter
```

Expected: pass. This is required because the stage introduces goroutines and shared metrics/progress state.

- [ ] **Step 6: Run export QA**

Run:

```bash
bun run qa:clips
bun run qa:export
```

Expected:

- Synthetic export succeeds.
- Progress remains sparse in non-TTY output.
- Synthetic output exists at `dist/qa/export/low-motion-bars-export.mp4`.
- `ffprobe` finds the output video stream.
- Optional real-sample audio QA passes when a top-level `dist/` sample with audio exists.

- [ ] **Step 7: Run throughput smoke with stats**

Run:

```bash
time ./bin/mojify export --overwrite --stats --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
```

Expected:

- Export succeeds.
- Output includes `export stats`.
- Output includes a positive worker count.
- Output includes no ETA or time remaining.
- Output remains an MP4 with a valid video stream.

- [ ] **Step 8: Inspect final diff**

Run:

```bash
git diff --check
git status --short --branch
```

Expected:

- `git diff --check` passes.
- Working tree is clean after commits.

---

## Self-Review

- Spec coverage: The plan covers timing metrics, faster encoder preset, worker-safe font faces, bounded parallel render/raster, ordered writes, progress semantics, docs, and verification.
- Placeholder scan: No unresolved placeholders or unbound "write tests for this" steps remain.
- Type consistency: `Options.Workers`, `Options.Stats`, `exportMetrics`, `exportFramePipelineOptions`, and `resolveExportWorkers` are defined before use.
- Scope check: Hardware encoders, GIF/PNG export, live terminal audio, and package distribution remain out of scope.
