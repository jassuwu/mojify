# Synchronized Presentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce visible terminal repaint during playback by enabling best-effort synchronized frame presentation by default.

**Architecture:** Keep the renderer, scheduler, controls, CLI, and stats surface unchanged. Add terminal synchronized-update escape sequences to the terminal package, then wrap each presented character frame with begin/end synchronization markers. Terminals that ignore the sequences should fall back to today's visual behavior without user configuration.

**Tech Stack:** Go 1.23, ANSI/VT escape sequences, FFmpeg CLI, Bun/Turborepo QA scripts.

---

## Decisions Already Made

- Next stage: terminal output optimization within playback quality hardening.
- Primary acceptance: visual QA improvement first, with playback metrics as regression guards.
- First mechanism: synchronized presentation before frame diffing.
- Fallback: synchronized presentation is best-effort and enabled by default.
- CLI surface: no new user-facing flag in this stage.
- Out of scope: frame diffing, renderer changes, lower fidelity rendering, terminal capability probing, audio, export, URL input, packaging.
- Average bytes per frame may increase slightly because synchronized-update markers add control bytes.

## File Structure

- `packages/core/internal/terminal/ansi.go`: define synchronized-update escape constants.
- `packages/core/internal/terminal/ansi_test.go`: lock the synchronized-update constants.
- `packages/core/internal/terminal/presenter.go`: wrap `Present` output in synchronized-update begin/end markers and keep metrics semantics intact.
- `packages/core/internal/terminal/presenter_test.go`: verify frame output is synchronized, lifecycle output remains unsynchronized, and metrics count the actual presented bytes.
- `docs/qa/playback-quality.md`: add synchronized-presentation visual QA notes and metrics guardrails.
- `docs/superpowers/plans/2026-06-01-mojify-synchronized-presentation.md`: track execution status for this plan.

---

## Task 1: Add Synchronized Update Escape Constants

**Files:**
- Modify: `packages/core/internal/terminal/ansi.go`
- Modify: `packages/core/internal/terminal/ansi_test.go`

- [ ] **Step 1: Add failing terminal constant test**

Add this test to `packages/core/internal/terminal/ansi_test.go` after `TestSerializeFrameUsesDeterministicRowsAndSuppressesRepeatedColor`:

```go
func TestSynchronizedUpdateSequencesAreStable(t *testing.T) {
	if BeginSynchronizedUpdate != "\x1b[?2026h" {
		t.Fatalf("BeginSynchronizedUpdate = %q, want CSI ? 2026 h", BeginSynchronizedUpdate)
	}
	if EndSynchronizedUpdate != "\x1b[?2026l" {
		t.Fatalf("EndSynchronizedUpdate = %q, want CSI ? 2026 l", EndSynchronizedUpdate)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected:

```text
FAIL
packages/core/internal/terminal/ansi_test.go:...: undefined: BeginSynchronizedUpdate
packages/core/internal/terminal/ansi_test.go:...: undefined: EndSynchronizedUpdate
```

- [ ] **Step 3: Define synchronized update constants**

Modify the `const` block in `packages/core/internal/terminal/ansi.go` to include:

```go
const (
	EnterAltScreen          = "\x1b[?1049h"
	ExitAltScreen           = "\x1b[?1049l"
	HideCursor              = "\x1b[?25l"
	ShowCursor              = "\x1b[?25h"
	CursorHome              = "\x1b[H"
	ClearToEnd              = "\x1b[J"
	BeginSynchronizedUpdate = "\x1b[?2026h"
	EndSynchronizedUpdate   = "\x1b[?2026l"
	Reset                   = "\x1b[0m"
)
```

- [ ] **Step 4: Run terminal tests**

Run:

```bash
gofmt -w packages/core/internal/terminal/ansi.go packages/core/internal/terminal/ansi_test.go
go test ./packages/core/internal/terminal
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/terminal
```

- [ ] **Step 5: Commit**

```bash
git add packages/core/internal/terminal/ansi.go packages/core/internal/terminal/ansi_test.go
git commit -m "feat: add synchronized update sequences"
```

---

## Task 2: Wrap Presented Frames In Synchronized Updates

**Files:**
- Modify: `packages/core/internal/terminal/presenter.go`
- Modify: `packages/core/internal/terminal/presenter_test.go`

- [ ] **Step 1: Add failing presenter output test update**

In `packages/core/internal/terminal/presenter_test.go`, update the `Present` assertion in `TestPresenterLifecycleWritesTerminalSequences`:

```go
	if got, want := out.String(), BeginSynchronizedUpdate+SerializeFrame(frame)+EndSynchronizedUpdate; got != want {
		t.Fatalf("Present wrote %q, want %q", got, want)
	}
```

Keep the `Start` assertion as:

```go
	if got, want := out.String(), EnterAltScreen+HideCursor+CursorHome+ClearToEnd; got != want {
		t.Fatalf("Start wrote %q, want %q", got, want)
	}
```

Keep the `Stop` assertion as:

```go
	if got, want := out.String(), Reset+ShowCursor+ExitAltScreen; got != want {
		t.Fatalf("Stop wrote %q, want %q", got, want)
	}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
go test ./packages/core/internal/terminal
```

Expected:

```text
FAIL
Present wrote "\x1b[H..." want "\x1b[?2026h\x1b[H...\x1b[?2026l"
```

- [ ] **Step 3: Implement synchronized frame writing**

Modify `packages/core/internal/terminal/presenter.go` so `Present` calls a helper that writes begin marker, serialized frame, and end marker:

```go
func (p Presenter) Present(frame render.CharacterFrame) error {
	start := time.Now()
	output := SerializeFrame(frame)
	n, err := writeSynchronizedFrame(p.Out, output)
	if err == nil && p.Metrics != nil {
		p.Metrics.RecordPresented(n, time.Since(start))
	}
	return err
}

func writeSynchronizedFrame(w io.Writer, output string) (int, error) {
	total, err := io.WriteString(w, BeginSynchronizedUpdate)
	if err != nil {
		return total, err
	}

	n, frameErr := io.WriteString(w, output)
	total += n

	n, endErr := io.WriteString(w, EndSynchronizedUpdate)
	total += n

	if frameErr != nil {
		return total, frameErr
	}
	if endErr != nil {
		return total, endErr
	}
	return total, nil
}
```

Do not wrap `Start` or `Stop`; only frame presentation should be synchronized.

- [ ] **Step 4: Run presenter tests**

Run:

```bash
gofmt -w packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go
go test ./packages/core/internal/terminal
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/terminal
```

- [ ] **Step 5: Verify metrics count synchronized output bytes**

Run:

```bash
go test ./packages/core/internal/terminal -run TestPresenterRecordsPlaybackMetrics -count=1000
```

Expected:

```text
ok  	github.com/jass/mojify/packages/core/internal/terminal
```

The existing `AverageBytesPerFrame == out.Len()` assertion must continue to pass, which means stats include the synchronized-update marker bytes.

- [ ] **Step 6: Commit**

```bash
git add packages/core/internal/terminal/presenter.go packages/core/internal/terminal/presenter_test.go
git commit -m "feat: synchronize terminal frame presentation"
```

---

## Task 3: Update Playback QA Docs

**Files:**
- Modify: `docs/qa/playback-quality.md`

- [ ] **Step 1: Update visual checklist**

In `docs/qa/playback-quality.md`, update the visual checklist to include synchronized-presentation observations:

```md
## Visual Checklist

For each clip:

- Playback starts in the alternate screen.
- `q` exits and restores the terminal.
- Space pauses and resumes playback.
- Ctrl-C restores the cursor and terminal.
- Playback does not show distracting full-screen flashing.
- Playback does not show obvious top-to-bottom repaint waves at normal terminal size.
- Synchronized presentation does not introduce visible stalling, tearing, or delayed frame bursts.
- The stats summary appears after exit.
- The stats summary includes render grid, rendered frames, presented frames, skipped frames, effective FPS, average render time, average present time, and average bytes per frame.
```

- [ ] **Step 2: Update notes to record**

In `docs/qa/playback-quality.md`, update `Notes To Record` to include terminal synchronization support:

```md
## Notes To Record

Capture these observations when comparing changes:

- Terminal app and version.
- Whether the terminal appears to support synchronized updates.
- Terminal size.
- Clip name.
- Whether repainting is distracting.
- Whether timing feels continuous.
- Stats summary.
```

- [ ] **Step 3: Add regression guardrail note**

Add this section after `Notes To Record`:

```md
## Regression Guardrails

For synchronized presentation, visual QA is the acceptance gate. Metrics are guardrails:

- Effective FPS should not materially regress against the previous `--stats` baseline for the same clip and terminal size.
- Presented frames should not materially regress against the previous `--stats` baseline for the same clip and terminal size.
- Average bytes per frame may increase slightly because synchronized-update markers add terminal control bytes.
```

- [ ] **Step 4: Run docs verification**

Run:

```bash
rg -n "Synchronized presentation|Regression Guardrails|synchronized updates" docs/qa/playback-quality.md
```

Expected:

```text
docs/qa/playback-quality.md:...: Synchronized presentation does not introduce visible stalling, tearing, or delayed frame bursts.
docs/qa/playback-quality.md:...: Whether the terminal appears to support synchronized updates.
docs/qa/playback-quality.md:...: ## Regression Guardrails
```

- [ ] **Step 5: Commit**

```bash
git add docs/qa/playback-quality.md
git commit -m "docs: add synchronized presentation qa notes"
```

---

## Task 4: Final Verification And Review

**Files:**
- Verify all changed files.

- [ ] **Step 1: Run full verification**

Run:

```bash
bun run fmt:check
bun run test
bun run typecheck
bun run build
go mod tidy -diff
bun run qa:clips
./bin/mojify probe dist/qa/low-motion-bars.mp4
./bin/mojify probe dist/qa/high-motion-testsrc.mp4
./bin/mojify probe dist/qa/high-contrast-grid.mp4
./bin/mojify play --stats dist/qa/low-motion-bars.mp4 >/private/tmp/mojify-sync-low.out 2>/private/tmp/mojify-sync-low.err
./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4 >/private/tmp/mojify-sync-high.out 2>/private/tmp/mojify-sync-high.err
./bin/mojify play --stats dist/qa/high-contrast-grid.mp4 >/private/tmp/mojify-sync-grid.out 2>/private/tmp/mojify-sync-grid.err
```

Expected:

```text
all commands pass
each probe prints video metadata
each stderr file contains "playback stats"
stdout files are non-empty
git status shows no generated dist/qa files because dist/ is ignored
```

- [ ] **Step 2: Verify stats stream placement**

Run:

```bash
rg -n "playback stats|play failed" /private/tmp/mojify-sync-low.out /private/tmp/mojify-sync-low.err /private/tmp/mojify-sync-high.out /private/tmp/mojify-sync-high.err /private/tmp/mojify-sync-grid.out /private/tmp/mojify-sync-grid.err
```

Expected:

```text
/private/tmp/mojify-sync-low.err:...:playback stats
/private/tmp/mojify-sync-high.err:...:playback stats
/private/tmp/mojify-sync-grid.err:...:playback stats
```

The command must not report `playback stats` in stdout files and must not report `play failed` in any file.

- [ ] **Step 3: Run interactive smoke**

Run in a real terminal:

```bash
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

During playback:

```text
press Space once to pause
press Space again to resume
press q to quit
```

Expected:

```text
alternate screen exits cleanly
cursor is restored
stats summary prints after exit
no obvious top-to-bottom repaint wave in a terminal that supports synchronized updates
```

- [ ] **Step 4: Review scope**

Run:

```bash
git diff --stat main...HEAD
git diff main...HEAD -- packages/core/internal/terminal README.md docs/qa package.json scripts | rg -n "diff|dirty region|seek|audio|export|url|--no-sync|--sync|renderer recipe|lower resolution|256-color"
```

Expected:

```text
changes are limited to synchronized terminal presentation, QA docs, and planning docs
the rg command prints no matches
```

- [ ] **Step 5: Request code review**

Ask reviewers to inspect:

```text
Synchronized Presentation:
- Frame presentation is wrapped in best-effort synchronized-update escape sequences.
- Start/Stop terminal lifecycle remains unsynchronized and still restores the terminal.
- Metrics count the actual bytes written for presented frames.
- No CLI flag, frame diffing, renderer change, lower fidelity mode, URL input, audio, export, or packaging work is included.
- Visual QA remains the acceptance gate; metrics are regression guards.
```

- [ ] **Step 6: Address review feedback**

If review returns Critical or Important findings, fix them before finishing the branch. Minor findings can be documented for the follow-up frame-diffing plan.

- [ ] **Step 7: Finish branch**

Run the finishing workflow:

```bash
git status --short
git log --oneline --max-count=8
```

Expected:

```text
working tree clean
recent commits include synchronized update constants, synchronized frame presentation, and synchronized QA docs
```

---

## Self-Review Checklist

- Scope coverage:
  - Synchronized presentation constants: Task 1.
  - Best-effort default frame wrapping: Task 2.
  - Existing CLI shape preserved: Task 2 and Task 4 scope scan.
  - Visual QA improvement first: Task 3 and Task 4 interactive smoke.
  - Metrics as regression guards: Task 3 and Task 4 stats runs.
  - No frame diffing or fidelity reduction: Task 4 scope scan.
- Placeholder scan:
  - No placeholder tokens, vague file paths, or unimplemented placeholders are present.
- Type consistency:
  - `BeginSynchronizedUpdate`, `EndSynchronizedUpdate`, and `writeSynchronizedFrame` are introduced before they are consumed.
