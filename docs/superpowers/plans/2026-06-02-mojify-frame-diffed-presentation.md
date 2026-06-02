# Frame-Diffed Presentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce Ghostty-visible terminal repainting by presenting only changed regions between consecutive character frames while preserving Mojify's renderer output, scheduler, controls, and CLI shape.

**Architecture:** Make `terminal.Presenter` stateful so it owns a private copy of the last successfully presented `render.CharacterFrame`. Keep `SerializeFrame` as the canonical full-redraw serializer for first frames, dimension changes, and fallback; add a separate cell-diff patch serializer that emits cursor-addressed contiguous changed runs per row. For changed frames, choose the diff patch when its inner payload length is less than or equal to the full-redraw payload length; identical frames write nothing and still count as presented.

**Tech Stack:** Go terminal package, ANSI escape sequences, existing playback metrics, Bun/Turbo scripts, FFmpeg-generated QA clips, manual Ghostty QA.

---

## Decisions From Grill

- Canonical term: **Frame-diffed presentation**.
- Primary acceptance gate: visible playback improvement in Ghostty, not only better metrics.
- Diff unit: `render.Cell`, comparing `Ch`, `R`, `G`, and `B`.
- Serialization unit: contiguous changed runs per row.
- Identical consecutive frames: successful no-op write, recorded as a presented frame with `0` emitted bytes.
- Synchronized presentation remains around non-empty frame payloads only.
- Full redraw and diff patch serialization stay separate.
- `terminal.Presenter` becomes stateful with private previous-frame state and pointer receiver methods.
- `player.Presenter` interface stays unchanged; the CLI passes `&presenter` explicitly.
- Diff/full selection compares inner payload lengths and chooses diff when `len(diffPatch) <= len(fullRedraw)`.
- Diff patch color state is self-contained and ends with `Reset`.
- Shape-invalid frames return an error, do not write, do not record metrics, do not update previous-frame state, and force the next valid frame to full redraw.
- Write errors do not update previous-frame state and force the next valid frame to full redraw.
- No CLI flag, terminal probing, scheduler change, buffer change, decode-size change, render-grid change, renderer recipe change, fidelity reduction, or `--stats-json` surface is included.

## Files

- Modify: `CONTEXT.md` only if the implementation reveals a missing domain term.
- Modify: `docs/qa/playback-quality.md` to document frame-diffed presentation QA.
- Modify: `packages/core/internal/terminal/ansi.go` to add cursor positioning, frame validation, and diff patch serialization.
- Modify: `packages/core/internal/terminal/ansi_test.go` to lock diff patch serialization behavior.
- Modify: `packages/core/internal/terminal/presenter.go` to make `Presenter` stateful and choose full redraw vs diff patch vs no-op.
- Modify: `packages/core/internal/terminal/presenter_test.go` to lock presenter state, metrics, fallback, and recovery behavior.
- Modify: `packages/core/internal/cli/play.go` to pass `&presenter` into `player.PlayWithControls`.

---

### Task 1: Capture Current `main` Ghostty Baseline

**Files:**
- Read: `docs/qa/playback-quality.md`
- Read/write ignored local notes: `dist/qa/frame-diffed-presentation-baseline.md`

- [ ] **Step 1: Confirm the branch and clean state**

Run:

```bash
git status --short --branch
```

Expected output:

```text
## main
```

- [ ] **Step 2: Build the current baseline binary**

Run:

```bash
bun run build
```

Expected: command exits `0` and refreshes `bin/mojify`.

- [ ] **Step 3: Ensure generated QA clips exist**

Run:

```bash
bun run qa:clips
```

Expected generated files:

```text
dist/qa/low-motion-bars.mp4
dist/qa/high-motion-testsrc.mp4
dist/qa/high-contrast-grid.mp4
```

- [ ] **Step 4: Record the Ghostty baseline note template**

Create or update ignored local file `dist/qa/frame-diffed-presentation-baseline.md` with this structure:

```md
# Frame-Diffed Presentation Baseline

Baseline commit:
Terminal:
Terminal size:
Date:

## Generated Clips

### low-motion-bars
Command: ./bin/mojify play --stats dist/qa/low-motion-bars.mp4
Visual notes:
Stats summary:

### high-motion-testsrc
Command: ./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4
Visual notes:
Stats summary:

### high-contrast-grid
Command: ./bin/mojify play --stats dist/qa/high-contrast-grid.mp4
Visual notes:
Stats summary:

## Real Local Clips

### Call of The Night opening
Command: ./bin/mojify play --stats "dist/Call of The Night - Opening ｜ 4K ｜ 60FPS ｜ Creditless ｜ [L96VbQ9ytWk].webm"
Visual notes:
Stats summary:

### IRIS OUT
Command: ./bin/mojify play --stats "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm"
Visual notes:
Stats summary:
```

The angle-bracket fields are local QA note fields. They must be filled before implementation starts.

- [ ] **Step 5: Run generated clips in Ghostty with stats**

Run these commands manually inside Ghostty at the same terminal size:

```bash
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4
./bin/mojify play --stats dist/qa/high-contrast-grid.mp4
```

Expected: each clip plays, exits with `q`, restores the terminal, and prints stats after exit. Fill the generated clip sections in `dist/qa/frame-diffed-presentation-baseline.md`.

- [ ] **Step 6: Run real local clips in Ghostty with stats**

Run these commands manually inside Ghostty at the same terminal size:

```bash
./bin/mojify play --stats "dist/Call of The Night - Opening ｜ 4K ｜ 60FPS ｜ Creditless ｜ [L96VbQ9ytWk].webm"
./bin/mojify play --stats "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm"
```

Expected: each clip plays, exits with `q`, restores the terminal, and prints stats after exit. Fill the real clip sections in `dist/qa/frame-diffed-presentation-baseline.md`.

- [ ] **Step 7: Do not commit ignored baseline notes**

Run:

```bash
git status --short
```

Expected: no tracked changes from `dist/qa/frame-diffed-presentation-baseline.md`, because `dist/` is ignored.

---

### Task 2: Add Failing Diff Patch Serializer Tests

**Files:**
- Modify: `packages/core/internal/terminal/ansi_test.go`
- Test: `go test ./packages/core/internal/terminal`

- [ ] **Step 1: Add cursor-position and diff patch tests**

Append these tests to `packages/core/internal/terminal/ansi_test.go`:

```go
func TestCursorPositionUsesOneBasedCoordinates(t *testing.T) {
	if got, want := CursorPosition(2, 3), "\x1b[2;3H"; got != want {
		t.Fatalf("CursorPosition(2, 3) = %q, want %q", got, want)
	}
}

func TestSerializeFramePatchWritesChangedRuns(t *testing.T) {
	previous := characterFrame(4, 2,
		render.Cell{Ch: 'A', R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', R: 1, G: 1, B: 1},
		render.Cell{Ch: 'A', R: 1, G: 1, B: 1},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
	)
	current := characterFrame(4, 2,
		render.Cell{Ch: 'A', R: 1, G: 1, B: 1},
		render.Cell{Ch: 'B', R: 9, G: 9, B: 9},
		render.Cell{Ch: 'C', R: 9, G: 9, B: 9},
		render.Cell{Ch: 'A', R: 1, G: 1, B: 1},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
		render.Cell{Ch: 'X', R: 2, G: 2, B: 2},
		render.Cell{Ch: 'Z', R: 3, G: 4, B: 5},
	)

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch returned error: %v", err)
	}
	want := CursorPosition(1, 2) + "\x1b[38;2;9;9;9mBC" +
		CursorPosition(2, 4) + "\x1b[38;2;3;4;5mZ" + Reset
	if got != want {
		t.Fatalf("SerializeFramePatch() = %q, want %q", got, want)
	}
	if strings.Contains(got, ClearToEnd) {
		t.Fatalf("SerializeFramePatch() included ClearToEnd in %q", got)
	}
}

func TestSerializeFramePatchReturnsEmptyForIdenticalFrames(t *testing.T) {
	frame := characterFrame(2, 1,
		render.Cell{Ch: 'A', R: 1, G: 2, B: 3},
		render.Cell{Ch: 'B', R: 4, G: 5, B: 6},
	)

	got, err := SerializeFramePatch(frame, frame)
	if err != nil {
		t.Fatalf("SerializeFramePatch returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("SerializeFramePatch identical frames = %q, want empty patch", got)
	}
}

func TestSerializeFramePatchTreatsColorOnlyChangeAsChanged(t *testing.T) {
	previous := characterFrame(1, 1, render.Cell{Ch: 'A', R: 1, G: 1, B: 1})
	current := characterFrame(1, 1, render.Cell{Ch: 'A', R: 2, G: 3, B: 4})

	got, err := SerializeFramePatch(previous, current)
	if err != nil {
		t.Fatalf("SerializeFramePatch returned error: %v", err)
	}
	want := CursorPosition(1, 1) + "\x1b[38;2;2;3;4mA" + Reset
	if got != want {
		t.Fatalf("SerializeFramePatch color-only change = %q, want %q", got, want)
	}
}

func TestSerializeFramePatchRejectsMismatchedDimensions(t *testing.T) {
	previous := characterFrame(1, 1, render.Cell{Ch: 'A'})
	current := characterFrame(2, 1, render.Cell{Ch: 'A'}, render.Cell{Ch: 'B'})

	_, err := SerializeFramePatch(previous, current)
	if err == nil {
		t.Fatal("SerializeFramePatch returned nil error for mismatched dimensions")
	}
	if !strings.Contains(err.Error(), "frame dimensions differ") {
		t.Fatalf("SerializeFramePatch error = %q, want frame dimensions differ", err.Error())
	}
}

func characterFrame(width int, height int, cells ...render.Cell) render.CharacterFrame {
	return render.CharacterFrame{Width: width, Height: height, Cells: cells}
}
```

- [ ] **Step 2: Run the focused terminal tests and verify they fail**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected: FAIL because `CursorPosition` and `SerializeFramePatch` are undefined.

---

### Task 3: Implement Diff Patch Serialization

**Files:**
- Modify: `packages/core/internal/terminal/ansi.go`
- Test: `go test ./packages/core/internal/terminal`

- [ ] **Step 1: Update imports in `ansi.go`**

Change the import block in `packages/core/internal/terminal/ansi.go` to:

```go
import (
	"errors"
	"fmt"
	"strings"

	"github.com/jass/mojify/packages/core/internal/render"
)
```

- [ ] **Step 2: Add validation and cursor helpers**

Add this code after the ANSI constants:

```go
var ErrInvalidCharacterFrame = errors.New("invalid character frame")

func CursorPosition(row int, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

func validateCharacterFrame(frame render.CharacterFrame) error {
	if frame.Width <= 0 || frame.Height <= 0 || len(frame.Cells) != frame.Width*frame.Height {
		return fmt.Errorf("%w: width=%d height=%d cells=%d", ErrInvalidCharacterFrame, frame.Width, frame.Height, len(frame.Cells))
	}
	return nil
}
```

- [ ] **Step 3: Add cell comparison and colored-run serialization helpers**

Add this code after `validateCharacterFrame`:

```go
func sameCell(a render.Cell, b render.Cell) bool {
	return a.Ch == b.Ch && a.R == b.R && a.G == b.G && a.B == b.B
}

func writeColoredRun(b *strings.Builder, cells []render.Cell) {
	var lastR, lastG, lastB uint8
	hasColor := false

	for _, cell := range cells {
		if !hasColor || cell.R != lastR || cell.G != lastG || cell.B != lastB {
			fmt.Fprintf(b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
			lastR, lastG, lastB = cell.R, cell.G, cell.B
			hasColor = true
		}
		b.WriteRune(cell.Ch)
	}
}
```

- [ ] **Step 4: Add `SerializeFramePatch`**

Add this code after `SerializeFrame`:

```go
func SerializeFramePatch(previous render.CharacterFrame, current render.CharacterFrame) (string, error) {
	if err := validateCharacterFrame(previous); err != nil {
		return "", err
	}
	if err := validateCharacterFrame(current); err != nil {
		return "", err
	}
	if previous.Width != current.Width || previous.Height != current.Height {
		return "", fmt.Errorf("frame dimensions differ: previous=%dx%d current=%dx%d", previous.Width, previous.Height, current.Width, current.Height)
	}

	var b strings.Builder
	hasPatch := false

	for y := 0; y < current.Height; y++ {
		x := 0
		for x < current.Width {
			index := y*current.Width + x
			if sameCell(previous.Cells[index], current.Cells[index]) {
				x++
				continue
			}

			startX := x
			for x < current.Width {
				index := y*current.Width + x
				if sameCell(previous.Cells[index], current.Cells[index]) {
					break
				}
				x++
			}

			hasPatch = true
			b.WriteString(CursorPosition(y+1, startX+1))
			writeColoredRun(&b, current.Cells[y*current.Width+startX:y*current.Width+x])
		}
	}

	if !hasPatch {
		return "", nil
	}
	b.WriteString(Reset)
	return b.String(), nil
}
```

- [ ] **Step 5: Run focused terminal tests and verify they pass**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected: PASS.

- [ ] **Step 6: Run formatting**

Run:

```bash
gofmt -w packages/core/internal/terminal/ansi.go packages/core/internal/terminal/ansi_test.go
```

- [ ] **Step 7: Commit serializer work**

Run:

```bash
git add packages/core/internal/terminal/ansi.go packages/core/internal/terminal/ansi_test.go
git commit -m "feat: add frame diff patch serialization"
```

Expected commit includes only `ansi.go` and `ansi_test.go`.

---

### Task 4: Add Failing Stateful Presenter Tests

**Files:**
- Modify: `packages/core/internal/terminal/presenter_test.go`
- Test: `go test ./packages/core/internal/terminal`

- [ ] **Step 1: Add `strings` to presenter test imports**

Change the import block in `packages/core/internal/terminal/presenter_test.go` to:

```go
import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)
```

- [ ] **Step 2: Add test for diff output after first frame**

Append this test before the helper writer types:

```go
func TestPresenterUsesDiffPatchAfterFirstFrame(t *testing.T) {
	var out bytes.Buffer
	presenter := Presenter{Out: &out}
	first := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 4, G: 5, B: 6},
		},
	}
	second := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'C', R: 7, G: 8, B: 9},
		},
	}

	if err := presenter.Present(first); err != nil {
		t.Fatalf("first Present returned error: %v", err)
	}
	out.Reset()

	if err := presenter.Present(second); err != nil {
		t.Fatalf("second Present returned error: %v", err)
	}

	patch, err := SerializeFramePatch(first, second)
	if err != nil {
		t.Fatalf("SerializeFramePatch returned error: %v", err)
	}
	want := BeginSynchronizedUpdate + patch + EndSynchronizedUpdate
	if got := out.String(); got != want {
		t.Fatalf("second Present wrote %q, want %q", got, want)
	}
	if strings.Contains(out.String(), ClearToEnd) {
		t.Fatalf("diff Present included ClearToEnd in %q", out.String())
	}
}
```

- [ ] **Step 3: Add test for identical-frame no-op metrics**

Append this test before the helper writer types:

```go
func TestPresenterNoopsIdenticalFrameAndRecordsZeroBytes(t *testing.T) {
	metrics := playback.NewMetrics(1, 1)
	var out bytes.Buffer
	presenter := Presenter{Out: &out, Metrics: metrics}
	frame := render.CharacterFrame{
		Width:  1,
		Height: 1,
		Cells:  []render.Cell{{Ch: 'A', R: 1, G: 2, B: 3}},
	}

	if err := presenter.Present(frame); err != nil {
		t.Fatalf("first Present returned error: %v", err)
	}
	firstBytes := out.Len()
	out.Reset()

	if err := presenter.Present(frame); err != nil {
		t.Fatalf("second Present returned error: %v", err)
	}
	if got := out.String(); got != "" {
		t.Fatalf("identical Present wrote %q, want no output", got)
	}

	snapshot := metrics.Snapshot()
	if snapshot.PresentedFrames != 2 {
		t.Fatalf("PresentedFrames = %d, want 2", snapshot.PresentedFrames)
	}
	if snapshot.AverageBytesPerFrame != firstBytes/2 {
		t.Fatalf("AverageBytesPerFrame = %d, want %d", snapshot.AverageBytesPerFrame, firstBytes/2)
	}
}
```

- [ ] **Step 4: Add test for full redraw fallback when patch is larger**

Append this test before the helper writer types:

```go
func TestPresenterFallsBackToFullRedrawWhenPatchIsLarger(t *testing.T) {
	var out bytes.Buffer
	presenter := Presenter{Out: &out}
	first := render.CharacterFrame{
		Width:  2,
		Height: 2,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'A', R: 1, G: 2, B: 3},
		},
	}
	second := render.CharacterFrame{
		Width:  2,
		Height: 2,
		Cells: []render.Cell{
			{Ch: 'B', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 1, G: 2, B: 3},
		},
	}

	if err := presenter.Present(first); err != nil {
		t.Fatalf("first Present returned error: %v", err)
	}
	out.Reset()

	if err := presenter.Present(second); err != nil {
		t.Fatalf("second Present returned error: %v", err)
	}

	want := BeginSynchronizedUpdate + SerializeFrame(second) + EndSynchronizedUpdate
	if got := out.String(); got != want {
		t.Fatalf("fallback Present wrote %q, want %q", got, want)
	}
}
```

- [ ] **Step 5: Add test for invalid frame recovery**

Append this test before the helper writer types:

```go
func TestPresenterRejectsInvalidFrameAndForcesNextFullRedraw(t *testing.T) {
	metrics := playback.NewMetrics(2, 1)
	var out bytes.Buffer
	presenter := Presenter{Out: &out, Metrics: metrics}
	first := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 4, G: 5, B: 6},
		},
	}
	invalid := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells:  []render.Cell{{Ch: 'X', R: 9, G: 9, B: 9}},
	}
	next := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'C', R: 7, G: 8, B: 9},
		},
	}

	if err := presenter.Present(first); err != nil {
		t.Fatalf("first Present returned error: %v", err)
	}
	out.Reset()

	if err := presenter.Present(invalid); err == nil {
		t.Fatal("invalid Present returned nil error")
	}
	if got := out.String(); got != "" {
		t.Fatalf("invalid Present wrote %q, want no output", got)
	}
	if snapshot := metrics.Snapshot(); snapshot.PresentedFrames != 1 {
		t.Fatalf("PresentedFrames after invalid frame = %d, want 1", snapshot.PresentedFrames)
	}
	out.Reset()

	if err := presenter.Present(next); err != nil {
		t.Fatalf("recovery Present returned error: %v", err)
	}
	want := BeginSynchronizedUpdate + SerializeFrame(next) + EndSynchronizedUpdate
	if got := out.String(); got != want {
		t.Fatalf("recovery Present wrote %q, want %q", got, want)
	}
}
```

- [ ] **Step 6: Add test for write-error recovery**

Append this test before the helper writer types:

```go
func TestPresenterWriteErrorForcesNextFullRedraw(t *testing.T) {
	var out bytes.Buffer
	presenter := Presenter{Out: &out}
	first := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'B', R: 4, G: 5, B: 6},
		},
	}
	second := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'A', R: 1, G: 2, B: 3},
			{Ch: 'C', R: 7, G: 8, B: 9},
		},
	}
	third := render.CharacterFrame{
		Width:  2,
		Height: 1,
		Cells: []render.Cell{
			{Ch: 'D', R: 9, G: 8, B: 7},
			{Ch: 'E', R: 6, G: 5, B: 4},
		},
	}

	if err := presenter.Present(first); err != nil {
		t.Fatalf("first Present returned error: %v", err)
	}

	failing := newFailOnceAfterBytesWriter(len(BeginSynchronizedUpdate) + 2)
	presenter.Out = failing
	if err := presenter.Present(second); err == nil {
		t.Fatal("failing Present returned nil error")
	}

	var recovery bytes.Buffer
	presenter.Out = &recovery
	if err := presenter.Present(third); err != nil {
		t.Fatalf("recovery Present returned error: %v", err)
	}

	want := BeginSynchronizedUpdate + SerializeFrame(third) + EndSynchronizedUpdate
	if got := recovery.String(); got != want {
		t.Fatalf("recovery Present wrote %q, want %q", got, want)
	}
}
```

- [ ] **Step 7: Run focused terminal tests and verify they fail**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected: FAIL because `Presenter` is still stateless, no-op frames still write a full frame, invalid frames are not validated, and write-error recovery is not implemented.

---

### Task 5: Implement Stateful Presenter Diffing

**Files:**
- Modify: `packages/core/internal/terminal/presenter.go`
- Modify: `packages/core/internal/cli/play.go`
- Test: `go test ./packages/core/internal/terminal`
- Test: `go test ./packages/core/internal/cli ./packages/core/internal/player`

- [ ] **Step 1: Change presenter state and method receivers**

Replace the `Presenter` type and `Start` method in `packages/core/internal/terminal/presenter.go` with:

```go
type Presenter struct {
	Out     io.Writer
	Metrics *playback.Metrics

	previous        render.CharacterFrame
	hasPrevious     bool
	forceFullRedraw bool
}

func (p *Presenter) Start() error {
	p.resetFrameState()
	_, err := fmt.Fprint(p.Out, EnterAltScreen, HideCursor, CursorHome, ClearToEnd)
	if err != nil {
		return errors.Join(err, p.Stop())
	}
	return err
}
```

- [ ] **Step 2: Replace `Present` with diff-aware logic**

Replace `Present` in `packages/core/internal/terminal/presenter.go` with:

```go
func (p *Presenter) Present(frame render.CharacterFrame) error {
	if err := validateCharacterFrame(frame); err != nil {
		p.forceFullRedraw = true
		return err
	}

	start := time.Now()
	output, err := p.outputFor(frame)
	if err != nil {
		p.forceFullRedraw = true
		return err
	}

	written := 0
	if output != "" {
		written, err = writeSynchronizedFrame(p.Out, output)
		if err != nil {
			p.forceFullRedraw = true
			return err
		}
	}

	p.storePrevious(frame)
	p.forceFullRedraw = false
	if p.Metrics != nil {
		p.Metrics.RecordPresented(written, time.Since(start))
	}
	return nil
}
```

- [ ] **Step 3: Add presenter output selection helpers**

Add this code below `Present`:

```go
func (p *Presenter) outputFor(frame render.CharacterFrame) (string, error) {
	if !p.hasPrevious || p.forceFullRedraw || dimensionsDiffer(p.previous, frame) {
		return SerializeFrame(frame), nil
	}

	patch, err := SerializeFramePatch(p.previous, frame)
	if err != nil {
		return "", err
	}
	if patch == "" {
		return "", nil
	}

	full := SerializeFrame(frame)
	if len(patch) <= len(full) {
		return patch, nil
	}
	return full, nil
}

func dimensionsDiffer(previous render.CharacterFrame, current render.CharacterFrame) bool {
	return previous.Width != current.Width || previous.Height != current.Height
}

func (p *Presenter) storePrevious(frame render.CharacterFrame) {
	p.previous = render.CharacterFrame{
		Width:  frame.Width,
		Height: frame.Height,
		Cells:  append([]render.Cell(nil), frame.Cells...),
	}
	p.hasPrevious = true
}

func (p *Presenter) resetFrameState() {
	p.previous = render.CharacterFrame{}
	p.hasPrevious = false
	p.forceFullRedraw = false
}
```

- [ ] **Step 4: Change `Stop` to clear presenter state**

Replace `Stop` in `packages/core/internal/terminal/presenter.go` with:

```go
func (p *Presenter) Stop() error {
	p.resetFrameState()
	_, err := fmt.Fprint(p.Out, EndSynchronizedUpdate, Reset, ShowCursor, ExitAltScreen)
	return err
}
```

- [ ] **Step 5: Pass the presenter by pointer in the CLI**

In `packages/core/internal/cli/play.go`, change:

```go
playErr := player.PlayWithControls(ctx, frames, presenter, info.FPS, controls, metrics)
```

to:

```go
playErr := player.PlayWithControls(ctx, frames, &presenter, info.FPS, controls, metrics)
```

- [ ] **Step 6: Update lifecycle test expectation if required by pointer receivers**

If `go test` reports that `Presenter` no longer satisfies an interface by value, update test call sites to pass `&presenter` only where an interface value is required. Direct calls such as `presenter.Present(frame)` should continue to compile because the local variable is addressable.

- [ ] **Step 7: Run focused terminal tests and verify they pass**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected: PASS.

- [ ] **Step 8: Run CLI and player package tests**

Run:

```bash
go test ./packages/core/internal/cli ./packages/core/internal/player
```

Expected: PASS.

- [ ] **Step 9: Format changed Go files**

Run:

```bash
gofmt -w packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go packages/core/internal/cli/play.go
```

- [ ] **Step 10: Commit presenter work**

Run:

```bash
git add packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go packages/core/internal/cli/play.go
git commit -m "feat: present frame diffs"
```

Expected commit includes presenter state, presenter tests, and the CLI pointer call.

---

### Task 6: Update Playback Quality QA Documentation

**Files:**
- Modify: `docs/qa/playback-quality.md`

- [ ] **Step 1: Add frame-diffed presentation checklist items**

In `docs/qa/playback-quality.md`, add these bullets under `## Visual Checklist` after the synchronized presentation bullet:

```md
- Frame-diffed presentation does not leave stale characters or stale colors.
- Frame-diffed presentation does not show cursor-positioning artifacts, bottom-row glitches, or visible patch trails.
- In Ghostty, frame-diffed presentation is visibly less distracting than the current `main` baseline at the same terminal size.
```

- [ ] **Step 2: Add frame-diffed notes to record**

Under `## Notes To Record`, add these bullets:

```md
- Current `main` baseline commit used for comparison.
- Whether frame-diffed presentation visibly improves Ghostty playback against that baseline.
- Whether full-screen clears or obvious repaint waves remain noticeable.
- Average bytes per frame before and after frame-diffed presentation.
```

- [ ] **Step 3: Replace synchronized-only regression language with stage-specific sections**

Replace the `## Regression Guardrails` section with:

```md
## Regression Guardrails

For synchronized presentation, visual QA is the acceptance gate. Metrics are guardrails:

- Effective FPS should not materially regress against the previous `--stats` baseline for the same clip, terminal app, and terminal size.
- Presented frames should not materially regress against the previous `--stats` baseline for the same clip, terminal app, and terminal size.
- If no prior baseline exists for that clip, terminal app, and terminal size, record the current stats as the comparison point and do not claim a metrics improvement.
- Average bytes per frame may increase slightly because synchronized-update markers add terminal control bytes.

For frame-diffed presentation, Ghostty-visible improvement is required:

- Compare against the current `main` baseline at the same Ghostty version and terminal size.
- Low-motion generated clips should show a material average-bytes-per-frame reduction.
- High-motion generated clips may show a smaller byte reduction because more cells genuinely change.
- Effective FPS and presented frames should not materially regress against the current `main` baseline.
- Real ignored `dist/` videos should be included as manual acceptance references.
- Do not call the stage successful if Ghostty playback still looks like full-screen clear/repaint, even when unit tests pass.
```

- [ ] **Step 4: Commit QA docs**

Run:

```bash
git add docs/qa/playback-quality.md
git commit -m "docs: add frame diff playback qa"
```

Expected commit includes only `docs/qa/playback-quality.md`.

---

### Task 7: Full Verification and Ghostty After-Comparison

**Files:**
- Read: `dist/qa/frame-diffed-presentation-baseline.md`
- Read/write ignored local notes: `dist/qa/frame-diffed-presentation-after.md`
- Read: `docs/qa/playback-quality.md`

- [ ] **Step 1: Run repository formatting check**

Run:

```bash
bun run fmt:check
```

Expected: PASS.

- [ ] **Step 2: Run repository tests**

Run:

```bash
bun run test
```

Expected: PASS.

- [ ] **Step 3: Run typecheck**

Run:

```bash
bun run typecheck
```

Expected: PASS.

- [ ] **Step 4: Run production build**

Run:

```bash
bun run build
```

Expected: PASS and `bin/mojify` exists.

- [ ] **Step 5: Verify Go module tidiness**

Run:

```bash
go mod tidy -diff
```

Expected: no diff output and exit `0`.

- [ ] **Step 6: Run race tests for core packages**

Run:

```bash
go test -race ./packages/core/internal/... ./packages/core/cmd/...
```

Expected: PASS.

- [ ] **Step 7: Regenerate QA clips**

Run:

```bash
bun run qa:clips
```

Expected generated files:

```text
dist/qa/low-motion-bars.mp4
dist/qa/high-motion-testsrc.mp4
dist/qa/high-contrast-grid.mp4
```

- [ ] **Step 8: Record after-comparison note template**

Create or update ignored local file `dist/qa/frame-diffed-presentation-after.md` with this structure:

```md
# Frame-Diffed Presentation After Comparison

Implementation commit:
Baseline note: dist/qa/frame-diffed-presentation-baseline.md
Terminal:
Terminal size:
Date:

## Generated Clips

### low-motion-bars
Command: ./bin/mojify play --stats dist/qa/low-motion-bars.mp4
Visual comparison against baseline:
Average bytes per frame before:
Average bytes per frame after:
Stats summary:

### high-motion-testsrc
Command: ./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4
Visual comparison against baseline:
Average bytes per frame before:
Average bytes per frame after:
Stats summary:

### high-contrast-grid
Command: ./bin/mojify play --stats dist/qa/high-contrast-grid.mp4
Visual comparison against baseline:
Average bytes per frame before:
Average bytes per frame after:
Stats summary:

## Real Local Clips

### Call of The Night opening
Command: ./bin/mojify play --stats "dist/Call of The Night - Opening ｜ 4K ｜ 60FPS ｜ Creditless ｜ [L96VbQ9ytWk].webm"
Visual comparison against baseline:
Average bytes per frame before:
Average bytes per frame after:
Stats summary:

### IRIS OUT
Command: ./bin/mojify play --stats "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm"
Visual comparison against baseline:
Average bytes per frame before:
Average bytes per frame after:
Stats summary:

## Acceptance

Ghostty-visible improvement:
Blocking artifacts observed:
Low-motion bytes/frame materially reduced:
Effective FPS materially regressed:
Presented frames materially regressed:
```

Fill every field before requesting final review.

- [ ] **Step 9: Run generated clips in Ghostty after implementation**

Run these commands manually inside Ghostty at the same terminal size used for baseline:

```bash
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4
./bin/mojify play --stats dist/qa/high-contrast-grid.mp4
```

Expected: each clip plays, exits with `q`, restores the terminal, and prints stats after exit. Fill generated clip sections in `dist/qa/frame-diffed-presentation-after.md`.

- [ ] **Step 10: Run real local clips in Ghostty after implementation**

Run these commands manually inside Ghostty at the same terminal size used for baseline:

```bash
./bin/mojify play --stats "dist/Call of The Night - Opening ｜ 4K ｜ 60FPS ｜ Creditless ｜ [L96VbQ9ytWk].webm"
./bin/mojify play --stats "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm"
```

Expected: each clip plays, exits with `q`, restores the terminal, and prints stats after exit. Fill real clip sections in `dist/qa/frame-diffed-presentation-after.md`.

- [ ] **Step 11: Reject the stage if Ghostty does not visibly improve**

Use this acceptance rule:

```text
If Ghostty playback still looks like full-screen clear/repaint on generated and real clips, do not mark the stage complete. Keep the implementation branch open and investigate whether the full-redraw fallback is happening too often, whether diff patches are visually artifacting, or whether the terminal write path is bottlenecked elsewhere.
```

- [ ] **Step 12: Commit any final test or docs corrections**

If verification reveals small docs or test corrections, commit them with a focused message:

```bash
git add packages/core/internal/terminal/ansi.go packages/core/internal/terminal/ansi_test.go packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go packages/core/internal/cli/play.go docs/qa/playback-quality.md
git commit -m "fix: harden frame diff presentation"
```

Expected: no ignored `dist/` notes are committed.

---

### Task 8: Scope Review Before Finishing

**Files:**
- Read: `git diff --stat main...HEAD`
- Read: `git diff main...HEAD -- packages/core/internal/terminal packages/core/internal/cli docs/qa CONTEXT.md docs/adr docs/superpowers/plans`

- [ ] **Step 1: Confirm changed files are in scope**

Run:

```bash
git diff --stat main...HEAD
```

Expected: changed files are limited to terminal presenter/ANSI code, terminal tests, CLI pointer passing, QA docs, and plan docs.

- [ ] **Step 2: Scan for forbidden scope**

Run:

```bash
git diff main...HEAD -- packages README.md docs scripts package.json | rg -n "audio|export|url|seek|speed|zoom|256-color|monochrome|decodeWidth|buffer|stats-json|--no-diff|--diff"
```

Expected: no matches from implementation changes. If matches appear in old plan text, inspect manually and confirm they are explicit out-of-scope notes.

- [ ] **Step 3: Summarize implementation for review**

Use this review summary:

```text
Implemented frame-diffed presentation for Mojify terminal playback.

Scope:
- Presenter now keeps private previous-frame state.
- First frame, dimension changes, dirty recovery, and oversized patches use full redraw.
- Steady-state changed frames use cursor-addressed changed row runs when the patch is no larger than full redraw.
- Identical frames are no-op writes and record 0 presented bytes.
- Diff patches are synchronized, color self-contained, and reset at the end.
- Invalid frame shape and write errors do not update previous-frame state and force next valid frame to full redraw.
- CLI passes the presenter by pointer; player interface is unchanged.
- QA docs now require Ghostty-visible improvement and before/after metrics comparison.

Out of scope:
- No CLI flag.
- No terminal probing.
- No scheduler, buffer, decode-size, render-grid, renderer, fidelity, audio, export, URL, or stats JSON changes.

Verification:
- bun run fmt:check
- bun run test
- bun run typecheck
- bun run build
- go mod tidy -diff
- go test -race ./packages/core/internal/... ./packages/core/cmd/...
- bun run qa:clips
- Ghostty before/after QA for generated clips and real ignored dist clips
```

- [ ] **Step 4: Request code review**

Run the review process from the requesting-code-review skill with the summary from Step 3. Critical and important findings must be fixed before finishing the branch. Minor findings can be documented for a follow-up only if they do not undermine Ghostty-visible improvement or terminal correctness.

---

## Self-Review

- Spec coverage: all grill decisions are covered by Tasks 1 through 8.
- Placeholder scan: the plan uses explicit code snippets, commands, expected outcomes, and acceptance rules.
- Type consistency: `CursorPosition`, `SerializeFramePatch`, `validateCharacterFrame`, `sameCell`, `writeColoredRun`, `outputFor`, `dimensionsDiffer`, `storePrevious`, and `resetFrameState` are introduced before use in implementation tasks.
- Scope consistency: implementation is presenter-only except for the required CLI pointer call and QA docs.
