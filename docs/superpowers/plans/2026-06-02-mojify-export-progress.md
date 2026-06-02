# Export Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add honest export progress for `mojify export` without showing a deceptive ETA or pretending MP4 finalization is part of linear frame progress.

**Architecture:** Keep progress owned by the exporter, emitted to stderr, and separate from FFmpeg's own progress output. Add a small progress reporter that can render interactive carriage-return updates for TTY stderr and sparse newline milestones for non-TTY stderr. Estimate totals from export-frame count when knowable; otherwise show an indeterminate rendered-frame count only.

**Tech Stack:** Go, FFmpeg CLI, ffprobe metadata, `golang.org/x/term`, existing Bun/Turbo scripts.

---

## Decisions Already Made

- Progress is shown by default on stderr.
- No ETA or time remaining is shown.
- A known total shows export-frame progress from `0%` to `100%`.
- Unknown totals never show a fake percent.
- `100%` means Mojify has rendered and written all visual frames to the MP4 encoder stdin.
- After visual frames reach `100%`, status switches to `finalizing mp4...`, then `export complete`.
- Interactive stderr uses one updating line.
- Non-TTY stderr uses sparse readable log lines.
- Errors leave the terminal on a clean line before `export failed: ...`.

## File Structure

- Modify: `CONTEXT.md`
  - Already contains the resolved `Export progress` glossary entry.
- Create: `packages/core/internal/exporter/progress.go`
  - Owns total-frame estimation and progress rendering.
- Create: `packages/core/internal/exporter/progress_test.go`
  - Locks known-total, unknown-total, throttling, finalization, non-TTY, and error-line behavior.
- Modify: `packages/core/internal/exporter/export.go`
  - Wires progress start, per-frame updates, all-frames-written, finalizing, complete, and error cleanup.
- Modify: `packages/core/internal/exporter/layout.go`
  - Adds progress-only option fields to `Options` if needed.
- Modify: `packages/core/internal/cli/export.go`
  - Detects whether stderr is a TTY and passes that into exporter options.
- Create: `packages/core/internal/cli/export_test.go`
  - Locks TTY detection fallback for non-file writers.
- Modify: `docs/qa/export.md`
  - Adds export progress expectations to manual and scripted QA.
- Modify: `docs/superpowers/plans/2026-06-02-mojify-export-progress.md`
  - Track implementation status.

---

### Task 1: Add Export Frame Total Estimation

**Files:**
- Create: `packages/core/internal/exporter/progress.go`
- Create: `packages/core/internal/exporter/progress_test.go`
- Modify: `packages/core/internal/exporter/layout.go`

- [x] **Step 1: Write failing tests for total-frame estimation**

Create `packages/core/internal/exporter/progress_test.go` with:

```go
package exporter

import (
	"testing"
)

func TestEstimateExportFrameTotalUsesDurationAndExplicitFPS(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       60,
		FrameCount:      600,
		DurationSeconds: 10,
	}
	layout := Layout{FPS: 12}
	options := Options{FPS: 12}

	total := estimateExportFrameTotal(info, layout, options)

	if total != 120 {
		t.Fatalf("total = %d, want 120", total)
	}
}

func TestEstimateExportFrameTotalUsesDurationAndProbedFPS(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       29.97,
		FrameCount:      0,
		DurationSeconds: 8.008,
	}
	layout := Layout{FPS: 29.97}

	total := estimateExportFrameTotal(info, layout, Options{})

	if total != 240 {
		t.Fatalf("total = %d, want 240", total)
	}
}

func TestEstimateExportFrameTotalFallsBackToSourceFramesWithoutFPSConversion(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       0,
		FrameCount:      240,
		DurationSeconds: 0,
	}
	layout := Layout{FPS: 24}

	total := estimateExportFrameTotal(info, layout, Options{})

	if total != 240 {
		t.Fatalf("total = %d, want 240", total)
	}
}

func TestEstimateExportFrameTotalRejectsSourceFramesWhenFPSConversionIsRequested(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       60,
		FrameCount:      600,
		DurationSeconds: 0,
	}
	layout := Layout{FPS: 12}
	options := Options{FPS: 12}

	total := estimateExportFrameTotal(info, layout, options)

	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
}

func TestEstimateExportFrameTotalDoesNotUseDefaultFPSForUnknownSourceFPS(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       0,
		FrameCount:      0,
		DurationSeconds: 10,
	}
	layout := Layout{FPS: 24}

	total := estimateExportFrameTotal(info, layout, Options{})

	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
}
```

- [x] **Step 2: Run focused tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail with undefined `InputProgressInfo` and `estimateExportFrameTotal`.

- [x] **Step 3: Add progress option fields**

Modify `packages/core/internal/exporter/layout.go` so `Options` becomes:

```go
type Options struct {
	Width               int
	FPS                 float64
	Bitrate             string
	Overwrite           bool
	ProgressInteractive bool
	ProgressClock       func() time.Time
}
```

Also add this import:

```go
import (
	"fmt"
	"math"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)
```

- [x] **Step 4: Implement total-frame estimation**

Create `packages/core/internal/exporter/progress.go` with:

```go
package exporter

import "math"

type InputProgressInfo struct {
	SourceFPS       float64
	FrameCount      int
	DurationSeconds float64
}

func estimateExportFrameTotal(info InputProgressInfo, layout Layout, options Options) int {
	if info.DurationSeconds > 0 && layout.FPS > 0 && (options.FPS > 0 || info.SourceFPS > 0) {
		total := int(math.Round(info.DurationSeconds * layout.FPS))
		if total > 0 {
			return total
		}
	}
	if options.FPS <= 0 && info.FrameCount > 0 {
		return info.FrameCount
	}
	return 0
}
```

- [x] **Step 5: Run focused tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/progress.go packages/core/internal/exporter/progress_test.go packages/core/internal/exporter/layout.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [x] **Step 6: Commit**

Run:

```bash
git add packages/core/internal/exporter/progress.go packages/core/internal/exporter/progress_test.go packages/core/internal/exporter/layout.go
git commit --no-gpg-sign -m "feat: estimate export progress totals"
```

---

### Task 2: Add the Progress Reporter

**Files:**
- Modify: `packages/core/internal/exporter/progress.go`
- Modify: `packages/core/internal/exporter/progress_test.go`

- [x] **Step 1: Add failing tests for known-total interactive progress**

Append to `packages/core/internal/exporter/progress_test.go`:

```go
func TestProgressReporterInteractiveKnownTotal(t *testing.T) {
	var out bytes.Buffer
	clock := &fakeClock{now: time.Unix(0, 0)}
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 10,
		Now:         clock.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.Frame(5)
	reporter.AllFramesWritten(10)
	reporter.Finalizing()
	reporter.Complete("out.mp4")

	got := out.String()
	for _, want := range []string{
		"export: in.mp4 -> out.mp4\n",
		"output: 320x184 @ 24.000 fps\n",
		"\x1b[2K\rexporting video: 0/10 frames 0%",
		"\x1b[2K\rexporting video: 10/10 frames 100%\n",
		"finalizing mp4...\n",
		"export complete: out.mp4\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
	}
	if strings.Contains(strings.ToLower(got), "eta") || strings.Contains(strings.ToLower(got), "remaining") {
		t.Fatalf("progress output contains ETA wording: %q", got)
	}
}
```

- [x] **Step 2: Add failing tests for unknown-total and non-TTY progress**

Append to `packages/core/internal/exporter/progress_test.go`:

```go
func TestProgressReporterUnknownTotalDoesNotPrintPercent(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 0,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(17)
	reporter.Frame(100)
	reporter.Frame(101)
	reporter.AllFramesWritten(117)
	reporter.Finalizing()

	got := out.String()
	for _, want := range []string{
		"exporting video: 0 frames\n",
		"exporting video: 100 frames\n",
		"exporting video: 117 frames\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "%") {
		t.Fatalf("unknown total printed percent: %q", got)
	}
	if strings.Contains(got, "17 frames") || strings.Contains(got, "101 frames") {
		t.Fatalf("unknown-total non-TTY progress was not sparse: %q", got)
	}
}

func TestProgressReporterNonTTYPrintsSparseMilestones(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 100,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.Frame(2)
	reporter.Frame(10)
	reporter.Frame(11)
	reporter.Frame(20)
	reporter.AllFramesWritten(100)

	got := out.String()
	if strings.Count(got, "exporting video:") != 4 {
		t.Fatalf("progress output = %q, want start plus 10%%, 20%%, and 100%% milestones", got)
	}
	for _, want := range []string{
		"exporting video: 0/100 frames 0%\n",
		"exporting video: 10/100 frames 10%\n",
		"exporting video: 20/100 frames 20%\n",
		"exporting video: 100/100 frames 100%\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
	}
}
```

- [x] **Step 3: Add failing tests for throttling and error-line cleanup**

Append to `packages/core/internal/exporter/progress_test.go`:

```go
func TestProgressReporterInteractiveThrottlesFrameUpdates(t *testing.T) {
	var out bytes.Buffer
	clock := &fakeClock{now: time.Unix(0, 0)}
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 100,
		Now:         clock.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.Frame(2)

	clock.now = clock.now.Add(100 * time.Millisecond)
	reporter.Frame(3)

	got := out.String()
	if strings.Contains(got, "1/100") || strings.Contains(got, "2/100") {
		t.Fatalf("throttled frame update was printed: %q", got)
	}
	if !strings.Contains(got, "3/100 frames 3%") {
		t.Fatalf("progress output missing throttled update after interval: %q", got)
	}
}

func TestProgressReporterClampsPercentUntilAllFramesWritten(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 10,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(10)

	got := out.String()
	if strings.Contains(got, "100%") {
		t.Fatalf("frame update printed 100%% before EOF: %q", got)
	}

	reporter.AllFramesWritten(10)
	got = out.String()
	if !strings.Contains(got, "10/10 frames 100%") {
		t.Fatalf("all-frames-written did not print 100%%: %q", got)
	}
}

func TestProgressReporterErrorLineCleansInteractiveStatus(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 10,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.ErrorLine()

	got := out.String()
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("error cleanup did not leave a clean line: %q", got)
	}
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}
```

Update the import block in `progress_test.go` to:

```go
import (
	"bytes"
	"strings"
	"testing"
	"time"
)
```

- [x] **Step 4: Run focused tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: fail with undefined `newProgressReporter`, `progressReporterOptions`, and reporter methods.

- [x] **Step 5: Implement the progress reporter**

Replace `packages/core/internal/exporter/progress.go` with:

```go
package exporter

import (
	"fmt"
	"io"
	"math"
	"time"
)

const (
	progressUpdateInterval = 100 * time.Millisecond
	progressLogStepPercent = 10
	progressLogStepFrames  = 100
	clearProgressLine      = "\x1b[2K\r"
)

type InputProgressInfo struct {
	SourceFPS       float64
	FrameCount      int
	DurationSeconds float64
}

type progressReporterOptions struct {
	Interactive bool
	TotalFrames int
	Now         func() time.Time
}

type progressReporter struct {
	out             io.Writer
	interactive     bool
	totalFrames     int
	now             func() time.Time
	lastUpdate      time.Time
	nextLogPercent  int
	nextLogFrame    int
	statusLineOpen  bool
	lastStatusValue string
}

func estimateExportFrameTotal(info InputProgressInfo, layout Layout, options Options) int {
	if info.DurationSeconds > 0 && layout.FPS > 0 && (options.FPS > 0 || info.SourceFPS > 0) {
		total := int(math.Round(info.DurationSeconds * layout.FPS))
		if total > 0 {
			return total
		}
	}
	if options.FPS <= 0 && info.FrameCount > 0 {
		return info.FrameCount
	}
	return 0
}

func newProgressReporter(out io.Writer, options progressReporterOptions) *progressReporter {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &progressReporter{
		out:            out,
		interactive:    options.Interactive,
		totalFrames:    max(options.TotalFrames, 0),
		now:            now,
		nextLogPercent: progressLogStepPercent,
		nextLogFrame:   progressLogStepFrames,
	}
}

func (p *progressReporter) Start(inputPath string, outputPath string, layout Layout) {
	if p == nil || p.out == nil {
		return
	}
	fmt.Fprintf(p.out, "export: %s -> %s\n", inputPath, outputPath)
	fmt.Fprintf(p.out, "output: %dx%d @ %.3f fps\n", layout.OutputWidth, layout.OutputHeight, layout.FPS)
	p.writeStatus(p.formatFrameStatus(0, false), true)
}

func (p *progressReporter) Frame(renderedFrames int) {
	if p == nil || p.out == nil {
		return
	}
	if renderedFrames < 0 {
		renderedFrames = 0
	}
	if p.interactive {
		if !p.lastUpdate.IsZero() && p.now().Sub(p.lastUpdate) < progressUpdateInterval {
			return
		}
		p.writeStatus(p.formatFrameStatus(renderedFrames, false), true)
		return
	}
	if p.totalFrames <= 0 {
		if renderedFrames >= p.nextLogFrame {
			p.writeStatus(p.formatFrameStatus(renderedFrames, false), false)
			for p.nextLogFrame <= renderedFrames {
				p.nextLogFrame += progressLogStepFrames
			}
		}
		return
	}
	percent := p.clampedPercent(renderedFrames, false)
	if percent >= p.nextLogPercent {
		p.writeStatus(p.formatFrameStatus(renderedFrames, false), false)
		for p.nextLogPercent <= percent {
			p.nextLogPercent += progressLogStepPercent
		}
	}
}

func (p *progressReporter) AllFramesWritten(renderedFrames int) {
	if p == nil || p.out == nil {
		return
	}
	p.writeStatus(p.formatFrameStatus(renderedFrames, true), true)
}

func (p *progressReporter) Finalizing() {
	if p == nil || p.out == nil {
		return
	}
	p.writePhase("finalizing mp4...")
}

func (p *progressReporter) Complete(outputPath string) {
	if p == nil || p.out == nil {
		return
	}
	p.writePhase(fmt.Sprintf("export complete: %s", outputPath))
}

func (p *progressReporter) ErrorLine() {
	if p == nil || p.out == nil || !p.statusLineOpen {
		return
	}
	fmt.Fprint(p.out, "\n")
	p.statusLineOpen = false
}

func (p *progressReporter) formatFrameStatus(renderedFrames int, complete bool) string {
	if p.totalFrames <= 0 {
		return fmt.Sprintf("exporting video: %d frames", renderedFrames)
	}
	if complete && renderedFrames != p.totalFrames {
		return fmt.Sprintf("exporting video: %d frames complete", renderedFrames)
	}
	percent := p.clampedPercent(renderedFrames, complete)
	displayFrames := renderedFrames
	if complete {
		displayFrames = p.totalFrames
	}
	return fmt.Sprintf("exporting video: %d/%d frames %d%%", displayFrames, p.totalFrames, percent)
}

func (p *progressReporter) clampedPercent(renderedFrames int, complete bool) int {
	if p.totalFrames <= 0 {
		return 0
	}
	if complete {
		return 100
	}
	percent := renderedFrames * 100 / p.totalFrames
	if percent >= 100 {
		return 99
	}
	if percent < 0 {
		return 0
	}
	return percent
}

func (p *progressReporter) writeStatus(status string, force bool) {
	if !force && status == p.lastStatusValue {
		return
	}
	p.lastStatusValue = status
	p.lastUpdate = p.now()
	if p.interactive {
		fmt.Fprintf(p.out, "%s%s", clearProgressLine, status)
		p.statusLineOpen = true
		return
	}
	fmt.Fprintf(p.out, "%s\n", status)
}

func (p *progressReporter) writePhase(status string) {
	if p.interactive && p.statusLineOpen {
		fmt.Fprint(p.out, "\n")
		p.statusLineOpen = false
	}
	fmt.Fprintf(p.out, "%s\n", status)
}
```

- [x] **Step 6: Run focused tests and verify they pass**

Run:

```bash
gofmt -w packages/core/internal/exporter/progress.go packages/core/internal/exporter/progress_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter
```

Expected: pass.

- [x] **Step 7: Commit**

Run:

```bash
git add packages/core/internal/exporter/progress.go packages/core/internal/exporter/progress_test.go
git commit --no-gpg-sign -m "feat: report export progress"
```

---

### Task 3: Wire Progress into Export and CLI TTY Detection

**Files:**
- Modify: `packages/core/internal/exporter/export.go`
- Modify: `packages/core/internal/cli/export.go`
- Create: `packages/core/internal/cli/export_test.go`

- [x] **Step 1: Add failing CLI TTY detection test**

Create `packages/core/internal/cli/export_test.go` with:

```go
package cli

import (
	"bytes"
	"testing"
)

func TestIsTerminalWriterReturnsFalseForPlainWriter(t *testing.T) {
	var writer bytes.Buffer

	if isTerminalWriter(&writer) {
		t.Fatal("isTerminalWriter returned true for bytes.Buffer")
	}
}
```

- [x] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail with undefined `isTerminalWriter`.

- [x] **Step 3: Implement TTY detection in CLI export**

Modify `packages/core/internal/cli/export.go` to:

```go
package cli

import (
	"context"
	"io"

	"github.com/jass/mojify/packages/core/internal/exporter"
	xterm "golang.org/x/term"
)

type fdWriter interface {
	Fd() uintptr
}

func RunExport(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions) error {
	return exporter.ExportMP4(ctx, inputPath, outputPath, stderr, exporter.Options{
		Width:               options.Width,
		FPS:                 options.FPS,
		Bitrate:             options.Bitrate,
		Overwrite:           options.Overwrite,
		ProgressInteractive: isTerminalWriter(stderr),
	})
}

func isTerminalWriter(writer io.Writer) bool {
	fd, ok := writer.(fdWriter)
	if !ok {
		return false
	}
	return xterm.IsTerminal(int(fd.Fd()))
}
```

- [x] **Step 4: Wire progress into `ExportMP4`**

Modify `packages/core/internal/exporter/export.go`.

Change the function signature to use a named return:

```go
func ExportMP4(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options) (err error) {
```

After layout resolution, replace the current status prints:

```go
	if stderr != nil {
		fmt.Fprintf(stderr, "export: %s -> %s\n", inputPath, outputPath)
		fmt.Fprintf(stderr, "output: %dx%d @ %.3f fps\n", layout.OutputWidth, layout.OutputHeight, layout.FPS)
	}
```

with:

```go
	progress := newProgressReporter(stderr, progressReporterOptions{
		Interactive: options.ProgressInteractive,
		TotalFrames: estimateExportFrameTotal(InputProgressInfo{
			SourceFPS:       info.FPS,
			FrameCount:      info.FrameCount,
			DurationSeconds: info.DurationSeconds,
		}, layout, options),
		Now: options.ProgressClock,
	})
	progress.Start(inputPath, outputPath, layout)
	defer func() {
		if err != nil {
			progress.ErrorLine()
		}
	}()
```

Before the render loop, add:

```go
	renderedFrames := 0
```

Inside the loop, immediately after a successful encoder write, add:

```go
		renderedFrames++
		progress.Frame(renderedFrames)
```

After `decodeCleaned = true`, check `decodeErr` before any completion output:

```go
	if decodeErr != nil {
		return fmt.Errorf("decoder failed: %w", decodeErr)
	}
```

Then close the encoder pipe. Only after `encodePipe.Close()` succeeds, add:

```go
	progress.AllFramesWritten(renderedFrames)
	progress.Finalizing()
```

This review correction prevents export progress from printing `100%` or `finalizing mp4...` when the decoder failed or the encoder rejected the frame stream during close.

Code-quality review also found that FFmpeg child error output could collide with an active interactive progress line. The implementation should route encoder stderr through a progress line-safe writer that opens a clean line before child error text is written.

Replace the existing completion print:

```go
	if stderr != nil {
		fmt.Fprintf(stderr, "export complete: %s\n", outputPath)
	}
```

with:

```go
	progress.Complete(outputPath)
```

- [x] **Step 5: Remove now-unused imports**

If `packages/core/internal/exporter/export.go` no longer uses `fmt`, keep it only if other errors still require it. The current file still uses `fmt.Errorf`, so it remains needed.

- [x] **Step 6: Run focused tests**

Run:

```bash
gofmt -w packages/core/internal/exporter/export.go packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go packages/core/internal/exporter/progress_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/exporter ./packages/core/internal/cli
```

Expected: pass.

- [x] **Step 7: Commit**

Run:

```bash
git add packages/core/internal/exporter/export.go packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go packages/core/internal/exporter/progress_test.go
git commit --no-gpg-sign -m "feat: wire export progress status"
```

---

### Task 4: Update Export QA Docs

**Files:**
- Modify: `docs/qa/export.md`
- Modify: `docs/superpowers/plans/2026-06-02-mojify-export-progress.md`

- [x] **Step 1: Update manual QA expectations**

In `docs/qa/export.md`, after the `Manual Synthetic Smoke` command block, add:

```md
Expected progress output:

- Interactive stderr updates one progress line while rendering frames.
- Known-total exports show rendered export-frame progress such as `exporting video: 120/240 frames 50%`.
- Progress reaches `100%` only after visual frames have been rendered and written to the encoder.
- After `100%`, status switches to `finalizing mp4...`.
- Export ends with `export complete: <output>`.
- No ETA or time-remaining text is printed.
```

- [x] **Step 2: Update checklist**

In `docs/qa/export.md`, append these checklist items:

```md
- Known-total export progress reaches `100%` before `finalizing mp4...`.
- Export progress does not print an ETA or time remaining.
- Non-TTY export logs remain sparse and readable.
```

- [x] **Step 3: Commit docs**

Run:

```bash
git add docs/qa/export.md docs/superpowers/plans/2026-06-02-mojify-export-progress.md
git commit --no-gpg-sign -m "docs: add export progress qa"
```

---

### Task 5: Final Verification

**Files:**
- Read: `git status --short --branch`
- Read: `git log --oneline --decorate -5`

- [x] **Step 1: Run formatting checks**

Run:

```bash
bun run fmt:check
```

Expected: pass.

- [x] **Step 2: Run Go module tidy diff**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go mod tidy -diff
```

Expected: no output and exit code 0.

- [x] **Step 3: Run tests**

Run:

```bash
bun run test
```

Expected: pass. If the sandbox blocks the default Go build cache, rerun with elevated permission rather than changing repository files.

- [x] **Step 4: Run typecheck and build**

Run:

```bash
bun run typecheck
bun run build
```

Expected: both pass.

- [x] **Step 5: Run export QA**

Run:

```bash
bun run qa:clips
bun run qa:export
```

Expected:

- Synthetic export succeeds.
- Export progress prints no ETA.
- Synthetic output exists at `dist/qa/export/low-motion-bars-export.mp4`.
- `ffprobe` finds the output video stream.
- Optional real-sample audio QA passes when a top-level `dist/` sample with audio exists.

- [x] **Step 6: Inspect final diff**

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

- Spec coverage: The plan covers known-total percent, unknown-total frame count, no ETA, TTY vs non-TTY output, `100%` semantics, finalization phase, clean error line, QA docs, and verification.
- Placeholder scan: No unresolved placeholders or unbound "write tests for this" steps remain.
- Type consistency: `Options.ProgressInteractive`, `Options.ProgressClock`, `InputProgressInfo`, `progressReporterOptions`, and `newProgressReporter` are defined before use.
- ADR check: No ADR is included because the decision is reversible product behavior, not a hard architecture decision.
