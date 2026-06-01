# Playback Quality Measurement Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the measuring stick for playback quality hardening: `mojify play --stats <video>`, runtime playback metrics, generated QA clips in `dist/qa/`, and a manual QA checklist.

**Architecture:** Keep playback behavior unchanged except for optional measurement. CLI parsing records whether `--stats` was requested; the play orchestration creates a metrics collector only for stats runs. Rendering, presenting, and scheduling report timing/count data into the collector, and normal playback prints a post-run summary only after the terminal has been restored.

**Tech Stack:** Go 1.23, FFmpeg CLI, Bun/Turborepo scripts, shell script QA clip generation.

---

## Decisions Already Made

- Next stage: playback quality hardening, not a new product surface.
- First implementation plan: measurement baseline only.
- Acceptance: practical terminal smoothness supported by metrics.
- Metrics: rendered frames, skipped frames, effective FPS, average frame render time, average present/write time, output bytes per frame, and render grid size.
- Stats surface: `mojify play --stats <video>` prints a post-run human-readable summary.
- Canonical QA clips: generated synthetic low-motion, high-motion, and high-contrast edge clips in ignored `dist/qa/`.
- Real videos in ignored `dist/` can supplement manual QA, but are not required for repeatability.

## File Structure

- `packages/core/internal/cli/cli.go`: parse `play --stats <video>` and store `Command.Stats`.
- `packages/core/internal/cli/cli_test.go`: CLI parsing tests for stats flag placement and invalid stats usage.
- `packages/core/cmd/mojify/main.go`: pass stats option and stderr writer into play orchestration.
- `packages/core/internal/cli/play.go`: create metrics collector, record render timing, attach presenter/player measurement, print summary after cleanup.
- `packages/core/internal/playback/metrics.go`: metrics collector, snapshot, and summary formatting.
- `packages/core/internal/playback/metrics_test.go`: unit tests for metrics math and summary content.
- `packages/core/internal/player/player.go`: record skipped frames when late buffered frames are dropped.
- `packages/core/internal/player/player_test.go`: verify skipped-frame metrics.
- `packages/core/internal/terminal/presenter.go`: record present/write duration and output bytes per presented frame.
- `packages/core/internal/terminal/presenter_test.go`: verify presenter writes and records bytes/duration.
- `scripts/generate-qa-clips.sh`: generate canonical QA clips into `dist/qa/`.
- `package.json`: add `qa:clips` script.
- `docs/qa/playback-quality.md`: manual QA checklist and expected stats workflow.
- `README.md`: mention `--stats` and `bun run qa:clips`.

---

## Task 1: Parse `play --stats`

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`

- [x] **Step 1: Add failing CLI parse tests**

Add these tests to `packages/core/internal/cli/cli_test.go` after `TestParsePlayCommand`:

```go
func TestParsePlayStatsBeforeInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "--stats", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != PlayCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, PlayCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if !cmd.Stats {
		t.Fatal("Stats = false, want true")
	}
}

func TestParsePlayStatsAfterInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "clip.mp4", "--stats"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if !cmd.Stats {
		t.Fatal("Stats = false, want true")
	}
}

func TestParseRejectsStatsForProbe(t *testing.T) {
	_, err := Parse([]string{"probe", "--stats", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for probe --stats")
	}
}

func TestParseRejectsDuplicateStats(t *testing.T) {
	_, err := Parse([]string{"play", "--stats", "clip.mp4", "--stats"})
	if err == nil {
		t.Fatal("Parse returned nil error for duplicate --stats")
	}
}
```

- [x] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected:

```text
FAIL
packages/core/internal/cli/cli_test.go:...: cmd.Stats undefined
```

- [x] **Step 3: Implement stats parsing**

Modify `packages/core/internal/cli/cli.go` so `Command` and `parseInputCommand` become:

```go
type Command struct {
	Kind      CommandKind
	InputPath string
	Stats     bool
}
```

```go
func parseInputCommand(kind CommandKind, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%s requires a video input", args[0])
	}

	var inputPath string
	stats := false
	for _, arg := range args[1:] {
		switch arg {
		case "--stats":
			if kind != PlayCommand {
				return Command{}, fmt.Errorf("%s does not accept --stats", args[0])
			}
			if stats {
				return Command{}, fmt.Errorf("%s accepts --stats only once", args[0])
			}
			stats = true
		default:
			if inputPath != "" {
				return Command{}, fmt.Errorf("%s accepts exactly one video input", args[0])
			}
			inputPath = arg
		}
	}
	if inputPath == "" {
		return Command{}, fmt.Errorf("%s requires a video input", args[0])
	}
	if hasProtocolInput(inputPath) {
		return Command{}, fmt.Errorf("%s accepts local video file paths only", args[0])
	}
	return Command{Kind: kind, InputPath: inputPath, Stats: stats}, nil
}
```

Modify the `Usage` section in `HelpText`:

```text
Usage:
  mojify play [--stats] <video>  Play a local video file in the terminal
  mojify probe <video>           Print media and render metadata
  mojify --help                  Show this help
```

- [x] **Step 4: Run tests**

Run:

```bash
gofmt -w packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
go test ./packages/core/internal/cli
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/cli
```

- [x] **Step 5: Commit**

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
git commit -m "feat: parse playback stats flag"
```

---

## Task 2: Playback Metrics Collector

**Files:**
- Create: `packages/core/internal/playback/metrics.go`
- Create: `packages/core/internal/playback/metrics_test.go`

- [x] **Step 1: Write failing metrics tests**

Create `packages/core/internal/playback/metrics_test.go`:

```go
package playback

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsSnapshotCalculatesPlaybackValues(t *testing.T) {
	metrics := NewMetrics(120, 40)
	start := time.Unix(10, 0)
	metrics.Start(start)
	metrics.RecordRendered(10 * time.Millisecond)
	metrics.RecordRendered(30 * time.Millisecond)
	metrics.RecordPresented(1000, 5*time.Millisecond)
	metrics.RecordPresented(1400, 15*time.Millisecond)
	metrics.RecordSkipped(3)
	metrics.Finish(start.Add(2 * time.Second))

	snapshot := metrics.Snapshot()
	if snapshot.GridCols != 120 || snapshot.GridRows != 40 {
		t.Fatalf("grid = %dx%d, want 120x40", snapshot.GridCols, snapshot.GridRows)
	}
	if snapshot.RenderedFrames != 2 {
		t.Fatalf("RenderedFrames = %d, want 2", snapshot.RenderedFrames)
	}
	if snapshot.PresentedFrames != 2 {
		t.Fatalf("PresentedFrames = %d, want 2", snapshot.PresentedFrames)
	}
	if snapshot.SkippedFrames != 3 {
		t.Fatalf("SkippedFrames = %d, want 3", snapshot.SkippedFrames)
	}
	if snapshot.EffectiveFPS != 1 {
		t.Fatalf("EffectiveFPS = %f, want 1", snapshot.EffectiveFPS)
	}
	if snapshot.AverageRenderTime != 20*time.Millisecond {
		t.Fatalf("AverageRenderTime = %s, want 20ms", snapshot.AverageRenderTime)
	}
	if snapshot.AveragePresentTime != 10*time.Millisecond {
		t.Fatalf("AveragePresentTime = %s, want 10ms", snapshot.AveragePresentTime)
	}
	if snapshot.AverageBytesPerFrame != 1200 {
		t.Fatalf("AverageBytesPerFrame = %d, want 1200", snapshot.AverageBytesPerFrame)
	}
}

func TestMetricsSummaryContainsHumanReadableFields(t *testing.T) {
	metrics := NewMetrics(80, 24)
	start := time.Unix(10, 0)
	metrics.Start(start)
	metrics.RecordRendered(2 * time.Millisecond)
	metrics.RecordPresented(512, 3*time.Millisecond)
	metrics.RecordSkipped(1)
	metrics.Finish(start.Add(time.Second))

	summary := metrics.Summary()
	for _, want := range []string{
		"playback stats",
		"render grid: 80x24",
		"rendered frames: 1",
		"presented frames: 1",
		"skipped frames: 1",
		"effective fps: 1.00",
		"avg render time: 2ms",
		"avg present time: 3ms",
		"avg bytes/frame: 512",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("Summary missing %q in:\n%s", want, summary)
		}
	}
}
```

- [x] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/playback
```

Expected:

```text
FAIL
package github.com/jass/mojify/packages/core/internal/playback is not in std
```

- [x] **Step 3: Implement metrics collector**

Create `packages/core/internal/playback/metrics.go`:

```go
package playback

import (
	"fmt"
	"sync"
	"time"
)

type Metrics struct {
	mu sync.Mutex

	gridCols int
	gridRows int

	startedAt  time.Time
	finishedAt time.Time

	renderedFrames int
	presentedFrames int
	skippedFrames int
	renderTime time.Duration
	presentTime time.Duration
	outputBytes int
}

type Snapshot struct {
	GridCols int
	GridRows int

	RenderedFrames int
	PresentedFrames int
	SkippedFrames int
	EffectiveFPS float64
	AverageRenderTime time.Duration
	AveragePresentTime time.Duration
	AverageBytesPerFrame int
	Elapsed time.Duration
}

func NewMetrics(gridCols int, gridRows int) *Metrics {
	return &Metrics{gridCols: gridCols, gridRows: gridRows}
}

func (m *Metrics) Start(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedAt = now
	m.finishedAt = time.Time{}
}

func (m *Metrics) Finish(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finishedAt = now
}

func (m *Metrics) RecordRendered(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renderedFrames++
	m.renderTime += duration
}

func (m *Metrics) RecordPresented(bytes int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presentedFrames++
	m.outputBytes += bytes
	m.presentTime += duration
}

func (m *Metrics) RecordSkipped(count int) {
	if count <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skippedFrames += count
}

func (m *Metrics) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	elapsed := time.Duration(0)
	if !m.startedAt.IsZero() {
		end := m.finishedAt
		if end.IsZero() {
			end = time.Now()
		}
		elapsed = end.Sub(m.startedAt)
	}

	effectiveFPS := 0.0
	if elapsed > 0 {
		effectiveFPS = float64(m.presentedFrames) / elapsed.Seconds()
	}

	averageRenderTime := time.Duration(0)
	if m.renderedFrames > 0 {
		averageRenderTime = m.renderTime / time.Duration(m.renderedFrames)
	}

	averagePresentTime := time.Duration(0)
	averageBytesPerFrame := 0
	if m.presentedFrames > 0 {
		averagePresentTime = m.presentTime / time.Duration(m.presentedFrames)
		averageBytesPerFrame = m.outputBytes / m.presentedFrames
	}

	return Snapshot{
		GridCols: m.gridCols,
		GridRows: m.gridRows,
		RenderedFrames: m.renderedFrames,
		PresentedFrames: m.presentedFrames,
		SkippedFrames: m.skippedFrames,
		EffectiveFPS: effectiveFPS,
		AverageRenderTime: averageRenderTime,
		AveragePresentTime: averagePresentTime,
		AverageBytesPerFrame: averageBytesPerFrame,
		Elapsed: elapsed,
	}
}

func (m *Metrics) Summary() string {
	snapshot := m.Snapshot()
	return fmt.Sprintf(
		"playback stats\nrender grid: %dx%d\nrendered frames: %d\npresented frames: %d\nskipped frames: %d\neffective fps: %.2f\navg render time: %s\navg present time: %s\navg bytes/frame: %d\n",
		snapshot.GridCols,
		snapshot.GridRows,
		snapshot.RenderedFrames,
		snapshot.PresentedFrames,
		snapshot.SkippedFrames,
		snapshot.EffectiveFPS,
		snapshot.AverageRenderTime,
		snapshot.AveragePresentTime,
		snapshot.AverageBytesPerFrame,
	)
}
```

- [x] **Step 4: Run tests**

Run:

```bash
gofmt -w packages/core/internal/playback/metrics.go packages/core/internal/playback/metrics_test.go
go test ./packages/core/internal/playback
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/playback
```

- [x] **Step 5: Commit**

```bash
git add packages/core/internal/playback
git commit -m "feat: add playback metrics collector"
```

---

## Task 3: Instrument Playback Pipeline

**Files:**
- Modify: `packages/core/internal/player/player.go`
- Modify: `packages/core/internal/player/player_test.go`
- Modify: `packages/core/internal/terminal/presenter.go`
- Modify: `packages/core/internal/terminal/presenter_test.go`
- Modify: `packages/core/internal/cli/play.go`
- Modify: `packages/core/cmd/mojify/main.go`
- Test: `packages/core/internal/cli/play_test.go`

- [x] **Step 1: Add failing player skipped-frame metrics test**

Modify `packages/core/internal/player/player_test.go` so `TestPlayerSkipsLateBufferedFrames` creates metrics and asserts skipped count:

```go
	metrics := playback.NewMetrics(1, 1)
	err := playWithControls(context.Background(), frames, presenter, 10, nil, clock, metrics)
	if err != nil {
		t.Fatalf("playWithControls returned error: %v", err)
	}
	if metrics.Snapshot().SkippedFrames == 0 {
		t.Fatal("SkippedFrames = 0, want skipped frame count")
	}
```

Add this import to the same file:

```go
	"github.com/jass/mojify/packages/core/internal/playback"
```

- [x] **Step 2: Add failing presenter metrics test**

Add this test to `packages/core/internal/terminal/presenter_test.go`:

```go
func TestPresenterRecordsPlaybackMetrics(t *testing.T) {
	metrics := playback.NewMetrics(2, 1)
	var out bytes.Buffer
	presenter := Presenter{Out: &out, Metrics: metrics}
	frame := render.CharacterFrame{
		Width: 2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 255, G: 0, B: 0},
			{Ch: 'B', R: 0, G: 255, B: 0},
		},
	}

	if err := presenter.Present(frame); err != nil {
		t.Fatalf("Present returned error: %v", err)
	}

	snapshot := metrics.Snapshot()
	if snapshot.PresentedFrames != 1 {
		t.Fatalf("PresentedFrames = %d, want 1", snapshot.PresentedFrames)
	}
	if snapshot.AverageBytesPerFrame != out.Len() {
		t.Fatalf("AverageBytesPerFrame = %d, want %d", snapshot.AverageBytesPerFrame, out.Len())
	}
	if snapshot.AveragePresentTime <= 0 {
		t.Fatalf("AveragePresentTime = %s, want positive duration", snapshot.AveragePresentTime)
	}
}
```

Add this import to `packages/core/internal/terminal/presenter_test.go`:

```go
	"github.com/jass/mojify/packages/core/internal/playback"
```

- [x] **Step 3: Add failing CLI stats summary test**

Add this test to `packages/core/internal/cli/play_test.go`:

```go
func TestPrintStatsWritesSummaryOnlyWhenEnabled(t *testing.T) {
	metrics := playback.NewMetrics(4, 2)
	start := time.Unix(10, 0)
	metrics.Start(start)
	metrics.RecordRendered(time.Millisecond)
	metrics.RecordPresented(100, time.Millisecond)
	metrics.Finish(start.Add(time.Second))

	var out bytes.Buffer
	printStats(&out, PlayOptions{Stats: true}, metrics)
	if !strings.Contains(out.String(), "playback stats") {
		t.Fatalf("stats output missing summary:\n%s", out.String())
	}

	out.Reset()
	printStats(&out, PlayOptions{Stats: false}, metrics)
	if out.Len() != 0 {
		t.Fatalf("stats disabled wrote %q", out.String())
	}
}
```

Add these imports to `packages/core/internal/cli/play_test.go`:

```go
	"bytes"
	"strings"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
```

- [x] **Step 4: Run tests to verify failure**

Run:

```bash
go test ./packages/core/internal/player ./packages/core/internal/terminal ./packages/core/internal/cli
```

Expected:

```text
FAIL
undefined: playback
unknown field Metrics
undefined: PlayOptions
undefined: printStats
```

- [x] **Step 5: Implement player skipped-frame metrics**

Modify `packages/core/internal/player/player.go` imports:

```go
import (
	"context"
	"errors"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)
```

Replace `Play`, `PlayWithControls`, and `playWithControls` signatures:

```go
func Play(ctx context.Context, frames <-chan render.CharacterFrame, presenter Presenter, fps float64) error {
	return PlayWithControls(ctx, frames, presenter, fps, nil, nil)
}

func PlayWithControls(
	ctx context.Context,
	frames <-chan render.CharacterFrame,
	presenter Presenter,
	fps float64,
	controls <-chan Control,
	metrics *playback.Metrics,
) error {
	return playWithControls(ctx, frames, presenter, fps, controls, realClock{}, metrics)
}

func playWithControls(
	ctx context.Context,
	frames <-chan render.CharacterFrame,
	presenter Presenter,
	fps float64,
	controls <-chan Control,
	clock playbackClock,
	metrics *playback.Metrics,
) error {
```

Update the late-frame skip call inside `playWithControls`:

```go
			} else {
				var skipped int
				frame, nextDeadline, skipped = skipLateBufferedFrames(frames, frame, nextDeadline, frameDuration, now)
				if metrics != nil {
					metrics.RecordSkipped(skipped)
				}
			}
```

Replace `skipLateBufferedFrames`:

```go
func skipLateBufferedFrames(
	frames <-chan render.CharacterFrame,
	frame render.CharacterFrame,
	nextDeadline time.Time,
	frameDuration time.Duration,
	now time.Time,
) (render.CharacterFrame, time.Time, int) {
	skipped := 0
	for now.Sub(nextDeadline) >= frameDuration {
		select {
		case nextFrame, ok := <-frames:
			if !ok {
				return frame, nextDeadline, skipped
			}
			frame = nextFrame
			nextDeadline = nextDeadline.Add(frameDuration)
			skipped++
		default:
			return frame, nextDeadline, skipped
		}
	}
	return frame, nextDeadline, skipped
}
```

Update existing player tests:

```go
err := playWithControls(context.Background(), frames, presenter, 10, nil, clock, metrics)
```

- [x] **Step 6: Implement presenter metrics**

Modify `packages/core/internal/terminal/presenter.go`:

```go
package terminal

import (
	"fmt"
	"io"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter struct {
	Out     io.Writer
	Metrics *playback.Metrics
}

func (p Presenter) Start() error {
	_, err := fmt.Fprint(p.Out, EnterAltScreen, HideCursor, CursorHome, ClearToEnd)
	return err
}

func (p Presenter) Present(frame render.CharacterFrame) error {
	start := time.Now()
	output := SerializeFrame(frame)
	n, err := io.WriteString(p.Out, output)
	if p.Metrics != nil {
		p.Metrics.RecordPresented(n, time.Since(start))
	}
	return err
}

func (p Presenter) Stop() error {
	_, err := fmt.Fprint(p.Out, Reset, ShowCursor, ExitAltScreen)
	return err
}
```

- [x] **Step 7: Wire metrics through play orchestration**

Modify `packages/core/internal/cli/play.go` imports:

```go
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
```

Add options type near the top:

```go
type PlayOptions struct {
	Stats bool
}
```

Change the public play functions:

```go
func RunPlay(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer, stderr io.Writer, options PlayOptions) error {
```

Create metrics after `grid`:

```go
	var metrics *playback.Metrics
	if options.Stats {
		metrics = playback.NewMetrics(grid.Cols, grid.Rows)
	}
```

Record render time in the render goroutine:

```go
				start := time.Now()
				frame := renderer.Render(rgb, grid)
				if metrics != nil {
					metrics.RecordRendered(time.Since(start))
				}
				select {
				case <-ctx.Done():
					return
				case frames <- frame:
				}
```

Attach metrics to the presenter:

```go
	presenter := terminal.Presenter{Out: stdout, Metrics: metrics}
```

Replace the presenter/control defers with guarded cleanup so stats print after terminal restoration:

```go
	presenterStopped := false
	defer func() {
		if !presenterStopped {
			_ = presenter.Stop()
		}
	}()

	controls, stopControls, err := startPlaybackControls(ctx, cancel, stdin)
	if err != nil {
		return err
	}
	controlsStopped := false
	defer func() {
		if !controlsStopped {
			stopControls()
		}
	}()
```

Start/finish metrics around playback:

```go
	if metrics != nil {
		metrics.Start(time.Now())
	}
	playErr := player.PlayWithControls(ctx, frames, presenter, info.FPS, controls, metrics)
	if metrics != nil {
		metrics.Finish(time.Now())
	}
```

Print stats before returning:

```go
	resultErr := playbackResult(ctxErr, playErr, frameErr, decoderErr)
	stopControls()
	controlsStopped = true
	if err := presenter.Stop(); err != nil && resultErr == nil {
		resultErr = err
	}
	presenterStopped = true
	printStats(stderr, options, metrics)
	return resultErr
```

Add helper:

```go
func printStats(w io.Writer, options PlayOptions, metrics *playback.Metrics) {
	if !options.Stats || metrics == nil {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, metrics.Summary())
}
```

- [x] **Step 8: Update main play call**

Modify the `PlayCommand` case in `packages/core/cmd/mojify/main.go`:

```go
	case cli.PlayCommand:
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := cli.RunPlay(
			ctx,
			cmd.InputPath,
			os.Stdin,
			os.Stdout,
			os.Stderr,
			cli.PlayOptions{Stats: cmd.Stats},
		); err != nil {
			fmt.Fprintf(os.Stderr, "play failed: %v\n", err)
			os.Exit(1)
		}
```

- [x] **Step 9: Run tests**

Run:

```bash
gofmt -w packages/core/internal/player/player.go packages/core/internal/player/player_test.go packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go packages/core/cmd/mojify/main.go
go test ./...
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/player
ok  	github.com/jass/mojify/packages/core/internal/terminal
ok  	github.com/jass/mojify/packages/core/internal/cli
```

- [x] **Step 10: Smoke stats output**

Build and run with a local sample:

```bash
bun run build
./bin/mojify play --stats /private/tmp/mojify-smoke.mp4 >/private/tmp/mojify-stats.out 2>/private/tmp/mojify-stats.err
```

Expected:

```text
command exits 0
/private/tmp/mojify-stats.out is non-empty
/private/tmp/mojify-stats.err contains "playback stats"
/private/tmp/mojify-stats.err contains "render grid:"
/private/tmp/mojify-stats.err contains "effective fps:"
```

- [x] **Step 11: Commit**

```bash
git add packages/core/cmd/mojify/main.go packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go packages/core/internal/player/player.go packages/core/internal/player/player_test.go packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go
git commit -m "feat: report playback stats"
```

---

## Task 4: QA Clip Generator And Checklist

**Files:**
- Create: `scripts/generate-qa-clips.sh`
- Create: `docs/qa/playback-quality.md`
- Modify: `package.json`
- Modify: `README.md`

- [x] **Step 1: Create QA clip generator script**

Create `scripts/generate-qa-clips.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist/qa

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "smptebars=size=320x180:rate=24:duration=5" \
  -c:v mpeg4 -q:v 3 -pix_fmt yuv420p \
  dist/qa/low-motion-bars.mp4

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "testsrc2=size=320x180:rate=60:duration=5" \
  -c:v mpeg4 -q:v 3 -pix_fmt yuv420p \
  dist/qa/high-motion-testsrc.mp4

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "color=c=black:size=320x180:rate=24:duration=5,drawgrid=width=32:height=18:thickness=2:color=white" \
  -c:v mpeg4 -q:v 3 -pix_fmt yuv420p \
  dist/qa/high-contrast-grid.mp4

printf 'Generated QA clips:\n'
printf '  dist/qa/low-motion-bars.mp4\n'
printf '  dist/qa/high-motion-testsrc.mp4\n'
printf '  dist/qa/high-contrast-grid.mp4\n'
```

- [x] **Step 2: Add root script**

Modify `package.json` scripts:

```json
"qa:clips": "bash scripts/generate-qa-clips.sh"
```

Keep the existing scripts unchanged.

- [x] **Step 3: Create playback QA checklist**

Create `docs/qa/playback-quality.md`:

````markdown
# Playback Quality QA

Playback quality hardening uses generated synthetic clips as the repeatable baseline and ignored real clips as optional manual references.

## Generate Clips

```bash
bun run qa:clips
```

Expected generated files:

- `dist/qa/low-motion-bars.mp4`
- `dist/qa/high-motion-testsrc.mp4`
- `dist/qa/high-contrast-grid.mp4`

## Manual Runs

Run each clip with stats:

```bash
bun run build
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4
./bin/mojify play --stats dist/qa/high-contrast-grid.mp4
```

Optional local real clips can also be run from ignored `dist/`:

```bash
./bin/mojify play --stats "dist/Call of The Night - Opening ｜ 4K ｜ 60FPS ｜ Creditless ｜ [L96VbQ9ytWk].webm"
./bin/mojify play --stats "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm"
```

## Visual Checklist

For each clip:

- Playback starts in the alternate screen.
- `q` exits and restores the terminal.
- Space pauses and resumes playback.
- Ctrl-C restores the cursor and terminal.
- Playback does not show distracting full-screen flashing.
- Playback does not show obvious top-to-bottom repaint waves at normal terminal size.
- The stats summary appears after exit.
- The stats summary includes render grid, rendered frames, presented frames, skipped frames, effective FPS, average render time, average present time, and average bytes per frame.

## Notes To Record

Capture these observations when comparing changes:

- Terminal app and version.
- Terminal size.
- Clip name.
- Whether repainting is distracting.
- Whether timing feels continuous.
- Stats summary.
````

- [x] **Step 4: Update README**

Add this section after `Run` in `README.md`:

````markdown
## Playback QA

```bash
bun run qa:clips
bun run build
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

The repeatable playback quality checklist lives in `docs/qa/playback-quality.md`.
````

- [x] **Step 5: Run generator and inspect clips**

Run:

```bash
bun run qa:clips
ls -lh dist/qa
```

Expected:

```text
dist/qa/low-motion-bars.mp4
dist/qa/high-motion-testsrc.mp4
dist/qa/high-contrast-grid.mp4
```

- [x] **Step 6: Run docs/script verification**

Run:

```bash
bun run fmt:check
bun run test
bun run build
./bin/mojify probe dist/qa/low-motion-bars.mp4
./bin/mojify probe dist/qa/high-motion-testsrc.mp4
./bin/mojify probe dist/qa/high-contrast-grid.mp4
```

Expected:

```text
all commands pass
each probe prints video metadata
```

- [x] **Step 7: Commit**

```bash
git add package.json README.md scripts/generate-qa-clips.sh docs/qa/playback-quality.md
git commit -m "docs: add playback quality qa workflow"
```

---

## Task 5: Final Verification And Review

**Files:**
- Verify all changed files.

- [x] **Step 1: Run full verification**

Run:

```bash
bun run fmt:check
bun run test
bun run typecheck
bun run build
go mod tidy -diff
bun run qa:clips
./bin/mojify probe dist/qa/low-motion-bars.mp4
./bin/mojify play --stats dist/qa/low-motion-bars.mp4 >/private/tmp/mojify-qa-play.out 2>/private/tmp/mojify-qa-play.err
```

Expected:

```text
all commands pass
/private/tmp/mojify-qa-play.out is non-empty
/private/tmp/mojify-qa-play.err contains "playback stats"
git status shows no generated dist/qa files because dist/ is ignored
```

- [x] **Step 2: Review scope**

Check:

```bash
git diff --stat main...HEAD
git diff main...HEAD -- packages README.md docs/qa scripts package.json | rg -n "synchronized|sync update|no full clear|width cap"
```

Expected:

```text
changes are limited to stats, metrics, generated QA clip workflow, and docs
the rg command prints no matches
```

- [x] **Step 3: Request code review**

Ask a reviewer to inspect:

```text
Playback Quality Measurement Baseline:
- `mojify play --stats <video>` prints post-run metrics without changing normal playback output.
- Metrics include rendered frames, skipped frames, effective FPS, average frame render time, average present/write time, output bytes per frame, and render grid size.
- QA clip generation writes synthetic clips to ignored `dist/qa/`.
- Manual QA checklist documents repeatable evaluation.
- No terminal output optimization is included.
```

- [x] **Step 4: Address review feedback**

If review returns Critical or Important findings, fix them before finishing the branch. Minor findings can be documented for the follow-up output-optimization plan.

- [x] **Step 5: Finish branch**

Run the finishing workflow:

```bash
git status --short
git log --oneline --max-count=8
```

Expected:

```text
working tree clean
recent commits include stats flag, metrics collector, stats reporting, and QA workflow
```

---

## Self-Review Checklist

- Scope coverage:
  - `--stats` parse path: Task 1.
  - Metrics collector and summary: Task 2.
  - Render, present/write, skipped frame accounting: Task 3.
  - QA clip generation to ignored `dist/qa/`: Task 4.
  - Manual QA checklist: Task 4.
  - Full verification and review: Task 5.
- Placeholder scan:
  - No placeholder tokens, vague file paths, or unimplemented placeholders are present.
- Type consistency:
  - `cli.Command.Stats`, `cli.PlayOptions`, `playback.Metrics`, `playback.Snapshot`, `player.PlayWithControls`, and `terminal.Presenter.Metrics` are introduced before being consumed.
