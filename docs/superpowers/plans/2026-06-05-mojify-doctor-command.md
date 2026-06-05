# Mojify Doctor Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `mojify doctor` as a lightweight runtime dependency checker for Mojify's external media tools.

**Architecture:** Implement a small `packages/core/internal/doctor` package that checks tool availability through an injectable command runner, classifies required versus optional tools, formats a stable report, and avoids real host dependency assumptions in tests. Add thin CLI parse/run/main wiring so `doctor` does not touch source resolution, terminal setup, media probing, decoding, or export.

**Tech Stack:** Go 1.23, existing CLI parser and command dispatch, `os/exec`, `context` timeouts, Bun/Turbo verification scripts, Markdown docs/ADR.

---

## File Structure

- Create `packages/core/internal/doctor/doctor.go`
  - Owns tool check definitions, injectable runner contract, timeout handling, version parsing, status classification, report summary, and report writing.
- Create `packages/core/internal/doctor/doctor_test.go`
  - Unit tests with fake runners; no test depends on real `ffmpeg`, `ffprobe`, `ffplay`, or `yt-dlp`.
- Create `packages/core/internal/cli/doctor.go`
  - Thin CLI runner for `mojify doctor`, returning a non-zero error only when required tools fail.
- Create `packages/core/internal/cli/doctor_test.go`
  - CLI runner tests with injected doctor runner options.
- Modify `packages/core/internal/cli/cli.go`
  - Add `DoctorCommand`, parser support, no-flag rejection, and help text.
- Modify `packages/core/internal/cli/cli_test.go`
  - Parse/help coverage for the new command.
- Modify `packages/core/cmd/mojify/main.go`
  - Dispatch `DoctorCommand`.
- Create `docs/adr/0030-add-runtime-doctor-command.md`
  - Record that the previous doctor deferral is now superseded by CLI polish.
- Modify `CONTEXT.md`
  - Add the `Runtime doctor` glossary term.
- Modify `README.md`
  - Mention `mojify doctor`, add it to capabilities, and remove npm/npx from the near-term roadmap.
- Modify `docs/release.md`
  - Include `mojify doctor` in local/release smoke checks.
- Modify this plan file as steps are completed.

## Task 1: Doctor Package Tests

**Files:**
- Create: `packages/core/internal/doctor/doctor_test.go`

- [ ] **Step 1: Write failing doctor package tests**

Create `packages/core/internal/doctor/doctor_test.go`:

```go
package doctor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type fakeResponse struct {
	stdout         string
	stderr         string
	err            error
	waitForContext bool
}

type fakeRunner map[string]fakeResponse

func (runner fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	response, ok := runner[name]
	if !ok {
		return nil, nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	if response.waitForContext {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	return []byte(response.stdout), []byte(response.stderr), response.err
}

func TestRunReportsVersionsAndOptionalWarnings(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {stdout: "ffmpeg version 8.0.1 Copyright\n"},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if !report.OK() {
		t.Fatalf("report.OK() = false, want true: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusOK, "8.0.1", "")
	assertResult(t, report, "ffprobe", StatusOK, "8.0.1", "")
	assertResult(t, report, "ffplay", StatusWarn, "", "missing; install ffplay and try again")
	assertResult(t, report, "yt-dlp", StatusOK, "2026.05.22", "")

	wantSummary := "Mojify can play and export local media. Live audio needs ffplay."
	if got := report.Summary(); got != wantSummary {
		t.Fatalf("Summary() = %q, want %q", got, wantSummary)
	}
}

func TestRunFailsWhenRequiredToolIsMissing(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg": {stdout: "ffmpeg version 8.0.1 Copyright\n"},
			"ffplay": {stdout: "ffplay version 8.0.1 Copyright\n"},
			"yt-dlp": {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffprobe", StatusError, "", "missing; install ffprobe and try again")

	wantSummary := "Mojify cannot play or export local media until required tools are installed."
	if got := report.Summary(); got != wantSummary {
		t.Fatalf("Summary() = %q, want %q", got, wantSummary)
	}
}

func TestRunClassifiesFailuresBySeverity(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {stderr: "broken ffmpeg\nextra detail", err: errors.New("exit status 1")},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"ffplay":  {stderr: "audio unavailable\n", err: errors.New("exit status 1")},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusError, "", "failed: broken ffmpeg")
	assertResult(t, report, "ffplay", StatusWarn, "", "failed: audio unavailable")
}

func TestRunReportsTimeouts(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {waitForContext: true},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"ffplay":  {stdout: "ffplay version 8.0.1 Copyright\n"},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: 5 * time.Millisecond,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusError, "", "timed out while checking version")
}

func TestWriteReport(t *testing.T) {
	report := Report{Results: []Result{
		{Name: "ffmpeg", Status: StatusOK, Version: "8.0.1", Required: true},
		{Name: "ffprobe", Status: StatusOK, Version: "8.0.1", Required: true},
		{Name: "ffplay", Status: StatusWarn, Detail: "missing; install ffplay and try again"},
		{Name: "yt-dlp", Status: StatusWarn, Detail: "missing; install yt-dlp and try again"},
	}}

	var out bytes.Buffer
	Write(&out, report)
	got := out.String()
	for _, want := range []string{
		"mojify doctor\n",
		"ok    ffmpeg",
		"ok    ffprobe",
		"warn  ffplay",
		"warn  yt-dlp",
		"Mojify can play and export local media. Live audio needs ffplay; platform URL input needs yt-dlp.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("report output missing %q in:\n%s", want, got)
		}
	}
}

func assertResult(t *testing.T, report Report, name string, status Status, version string, detail string) {
	t.Helper()
	for _, result := range report.Results {
		if result.Name != name {
			continue
		}
		if result.Status != status || result.Version != version || result.Detail != detail {
			t.Fatalf("%s result = %#v, want status=%q version=%q detail=%q", name, result, status, version, detail)
		}
		return
	}
	t.Fatalf("missing result for %s in %#v", name, report.Results)
}
```

- [ ] **Step 2: Run doctor package tests to verify they fail**

Run:

```bash
go test ./packages/core/internal/doctor
```

Expected: FAIL because `packages/core/internal/doctor` has no implementation package yet.

## Task 2: Doctor Package Implementation

**Files:**
- Create: `packages/core/internal/doctor/doctor.go`
- Test: `packages/core/internal/doctor/doctor_test.go`

- [ ] **Step 1: Implement doctor package**

Create `packages/core/internal/doctor/doctor.go`:

```go
package doctor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 2 * time.Second

type Status string

const (
	StatusOK    Status = "ok"
	StatusWarn  Status = "warn"
	StatusError Status = "error"
)

type Check struct {
	Name     string
	Args     []string
	Required bool
}

type Result struct {
	Name     string
	Status   Status
	Version  string
	Detail   string
	Required bool
}

type Report struct {
	Results []Result
}

type Runner func(ctx context.Context, name string, args ...string) ([]byte, []byte, error)

type Options struct {
	Runner  Runner
	Timeout time.Duration
	Checks  []Check
}

func DefaultChecks() []Check {
	return []Check{
		{Name: "ffmpeg", Args: []string{"-version"}, Required: true},
		{Name: "ffprobe", Args: []string{"-version"}, Required: true},
		{Name: "ffplay", Args: []string{"-version"}, Required: false},
		{Name: "yt-dlp", Args: []string{"--version"}, Required: false},
	}
}

func Run(ctx context.Context, options Options) Report {
	runner := options.Runner
	if runner == nil {
		runner = execRunner
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	checks := options.Checks
	if len(checks) == 0 {
		checks = DefaultChecks()
	}

	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		results = append(results, runCheck(ctx, check, runner, timeout))
	}
	return Report{Results: results}
}

func (report Report) OK() bool {
	for _, result := range report.Results {
		if result.Required && result.Status == StatusError {
			return false
		}
	}
	return true
}

func (report Report) Summary() string {
	if !report.OK() {
		return "Mojify cannot play or export local media until required tools are installed."
	}

	ffplayWarn := hasNonOK(report, "ffplay")
	ytdlpWarn := hasNonOK(report, "yt-dlp")
	switch {
	case ffplayWarn && ytdlpWarn:
		return "Mojify can play and export local media. Live audio needs ffplay; platform URL input needs yt-dlp."
	case ffplayWarn:
		return "Mojify can play and export local media. Live audio needs ffplay."
	case ytdlpWarn:
		return "Mojify can play and export local media. Platform URL input needs yt-dlp."
	default:
		return "Mojify can play and export local media, play live audio, and resolve platform URLs."
	}
}

func Write(w io.Writer, report Report) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, "mojify doctor")
	fmt.Fprintln(w)
	for _, result := range report.Results {
		value := result.Version
		if value == "" {
			value = result.Detail
		}
		fmt.Fprintf(w, "%-5s %-8s %s\n", result.Status, result.Name, value)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, report.Summary())
}

func runCheck(ctx context.Context, check Check, runner Runner, timeout time.Duration) Result {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout, stderr, err := runner(checkCtx, check.Name, check.Args...)
	if err != nil {
		return failedResult(check, detailForFailure(checkCtx, check.Name, stderr, err))
	}

	output := string(stdout)
	if strings.TrimSpace(output) == "" {
		output = string(stderr)
	}
	version := parseVersion(check.Name, output)
	if version == "" {
		version = "available"
	}
	return Result{Name: check.Name, Status: StatusOK, Version: version, Required: check.Required}
}

func failedResult(check Check, detail string) Result {
	status := StatusWarn
	if check.Required {
		status = StatusError
	}
	return Result{Name: check.Name, Status: status, Detail: detail, Required: check.Required}
}

func detailForFailure(ctx context.Context, name string, stderr []byte, err error) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return "timed out while checking version"
	}
	if isMissingExecutable(err) {
		return fmt.Sprintf("missing; install %s and try again", name)
	}
	if line := firstLine(string(stderr)); line != "" {
		return fmt.Sprintf("failed: %s", line)
	}
	return fmt.Sprintf("failed: %v", err)
}

func isMissingExecutable(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var execErr *exec.Error
	return errors.As(err, &execErr)
}

func parseVersion(name string, output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if name == "yt-dlp" {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
			return line
		}
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "version" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
		return line
	}
	return ""
}

func firstLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func hasNonOK(report Report, name string) bool {
	for _, result := range report.Results {
		if result.Name == name {
			return result.Status != StatusOK
		}
	}
	return false
}

func execRunner(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
```

- [ ] **Step 2: Run doctor package tests**

Run:

```bash
go test ./packages/core/internal/doctor
```

Expected: PASS.

- [ ] **Step 3: Commit doctor package**

```bash
git add packages/core/internal/doctor/doctor.go packages/core/internal/doctor/doctor_test.go
git commit --no-gpg-sign -m "feat: add runtime doctor checks"
```

## Task 3: CLI Parser and Help

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Add failing parser/help tests**

In `packages/core/internal/cli/cli_test.go`, add these tests after `TestParseVersionCommands`:

```go
func TestParseDoctorCommand(t *testing.T) {
	cmd, err := Parse([]string{"doctor"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != DoctorCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, DoctorCommand)
	}
}

func TestParseDoctorRejectsArgumentsAndOptions(t *testing.T) {
	for _, args := range [][]string{
		{"doctor", "--json"},
		{"doctor", "--help"},
		{"doctor", "ffmpeg"},
	} {
		_, err := Parse(args)
		if err == nil {
			t.Fatalf("Parse(%v) returned nil error", args)
		}
		if !strings.Contains(err.Error(), "doctor accepts no arguments or options") {
			t.Fatalf("Parse(%v) error = %q, want doctor rejection", args, err.Error())
		}
	}
}
```

In `TestHelpTextMentionsCommands`, add these expected substrings:

```go
"mojify doctor",
"Check runtime dependency health",
```

- [ ] **Step 2: Run parser tests to verify they fail**

Run:

```bash
go test ./packages/core/internal/cli -run 'TestParseDoctor|TestHelpTextMentionsCommands'
```

Expected: FAIL because `DoctorCommand` and parser/help wiring do not exist.

- [ ] **Step 3: Implement parser and help support**

In `packages/core/internal/cli/cli.go`, add `DoctorCommand` to the command enum:

```go
const (
	HelpCommand CommandKind = iota
	VersionCommand
	DoctorCommand
	PlayCommand
	ProbeCommand
	ExportCommand
)
```

In `Parse`, add the `doctor` case before media commands:

```go
	case "doctor":
		if len(args) != 1 {
			return Command{}, fmt.Errorf("doctor accepts no arguments or options")
		}
		return Command{Kind: DoctorCommand}, nil
```

In `HelpText`, add the doctor usage row between version/export and help:

```text
  mojify doctor                                       Check runtime dependency health
```

Keep the requirements section unchanged in this task.

- [ ] **Step 4: Run parser/help tests**

Run:

```bash
go test ./packages/core/internal/cli -run 'TestParseDoctor|TestHelpTextMentionsCommands'
```

Expected: PASS.

- [ ] **Step 5: Commit parser/help support**

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
git commit --no-gpg-sign -m "feat: parse doctor command"
```

## Task 4: CLI Doctor Runner and Main Dispatch

**Files:**
- Create: `packages/core/internal/cli/doctor.go`
- Create: `packages/core/internal/cli/doctor_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Write failing CLI runner tests**

Create `packages/core/internal/cli/doctor_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/doctor"
)

func TestRunDoctorSucceedsWithOptionalWarnings(t *testing.T) {
	var stdout bytes.Buffer
	err := runDoctorWithOptions(context.Background(), &stdout, doctor.Options{
		Runner: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			switch name {
			case "ffmpeg":
				return []byte("ffmpeg version 8.0.1 Copyright\n"), nil, nil
			case "ffprobe":
				return []byte("ffprobe version 8.0.1 Copyright\n"), nil, nil
			default:
				return nil, nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
			}
		},
		Timeout: time.Second,
	})

	if err != nil {
		t.Fatalf("runDoctorWithOptions returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"mojify doctor",
		"ok    ffmpeg",
		"ok    ffprobe",
		"warn  ffplay",
		"warn  yt-dlp",
		"Mojify can play and export local media.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor output missing %q in:\n%s", want, got)
		}
	}
}

func TestRunDoctorFailsForRequiredErrors(t *testing.T) {
	var stdout bytes.Buffer
	err := runDoctorWithOptions(context.Background(), &stdout, doctor.Options{
		Runner: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			return nil, nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
		},
		Timeout: time.Second,
	})

	if err == nil {
		t.Fatal("runDoctorWithOptions returned nil error for missing required tools")
	}
	if !strings.Contains(err.Error(), "required runtime tools are missing or unhealthy") {
		t.Fatalf("error = %q, want required runtime wording", err.Error())
	}
	got := stdout.String()
	for _, want := range []string{
		"error ffmpeg",
		"error ffprobe",
		"Mojify cannot play or export local media until required tools are installed.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor output missing %q in:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run CLI doctor tests to verify they fail**

Run:

```bash
go test ./packages/core/internal/cli -run TestRunDoctor
```

Expected: FAIL because `runDoctorWithOptions` does not exist.

- [ ] **Step 3: Implement CLI doctor runner**

Create `packages/core/internal/cli/doctor.go`:

```go
package cli

import (
	"context"
	"errors"
	"io"

	"github.com/jass/mojify/packages/core/internal/doctor"
)

func RunDoctor(ctx context.Context, stdout io.Writer) error {
	return runDoctorWithOptions(ctx, stdout, doctor.Options{})
}

func runDoctorWithOptions(ctx context.Context, stdout io.Writer, options doctor.Options) error {
	report := doctor.Run(ctx, options)
	doctor.Write(stdout, report)
	if !report.OK() {
		return errors.New("required runtime tools are missing or unhealthy")
	}
	return nil
}
```

- [ ] **Step 4: Wire main dispatch**

In `packages/core/cmd/mojify/main.go`, add this switch case after `VersionCommand`:

```go
	case cli.DoctorCommand:
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := cli.RunDoctor(ctx, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
			os.Exit(1)
		}
```

- [ ] **Step 5: Run CLI and command tests**

Run:

```bash
go test ./packages/core/internal/cli ./packages/core/cmd/mojify
```

Expected: PASS.

- [ ] **Step 6: Commit doctor runner and dispatch**

```bash
git add packages/core/internal/cli/doctor.go packages/core/internal/cli/doctor_test.go packages/core/cmd/mojify/main.go
git commit --no-gpg-sign -m "feat: run doctor command"
```

## Task 5: Documentation and ADR

**Files:**
- Create: `docs/adr/0030-add-runtime-doctor-command.md`
- Modify: `CONTEXT.md`
- Modify: `README.md`
- Modify: `docs/release.md`

- [ ] **Step 1: Add ADR**

Create `docs/adr/0030-add-runtime-doctor-command.md`:

```md
# Add runtime doctor command

Mojify will add `mojify doctor` as a post-distribution CLI polish stage. This supersedes the earlier installable-distribution deferral of a doctor command because Mojify now has Homebrew bottles, source fallback, and GitHub Release tarballs, and users need one command that explains whether the external runtime tools are available.

The doctor command checks `ffmpeg`, `ffprobe`, `ffplay`, and `yt-dlp` on `PATH`. Missing or unhealthy `ffmpeg` and `ffprobe` are errors because local media playback, probing, and export depend on them. Missing or unhealthy `ffplay` and `yt-dlp` are warnings because visual playback can run with `--no-audio`, and local file workflows do not require platform URL resolution.

Doctor does not install dependencies, run network checks, download sample media, check audio devices, probe terminal capabilities, or produce machine-readable diagnostics in this stage.
```

- [ ] **Step 2: Add glossary entry**

In `CONTEXT.md`, add this entry after `Runtime dependency hint`:

```md
**Runtime doctor**:
The `mojify doctor` CLI command that checks whether external runtime tools are available on `PATH`, treating `ffmpeg` and `ffprobe` as required and `ffplay` and `yt-dlp` as optional capability warnings.
_Avoid_: Dependency installer, Homebrew repair command, network smoke test
```

- [ ] **Step 3: Update README**

In `README.md`, after the tarball install paragraph, add:

```md
Check runtime tools:

```bash
mojify doctor
```
```

In the current capabilities list, add:

```md
- Runtime dependency check with `mojify doctor`
```

In the `Requirements` section, replace:

```md
Homebrew installs declare `ffmpeg` and `yt-dlp`. Tarball installs require the runtime tools to be installed separately.
```

with:

```md
Homebrew installs declare `ffmpeg` and `yt-dlp`. Tarball installs require the runtime tools to be installed separately. Run `mojify doctor` to check the tools visible to the installed binary.
```

In the `Roadmap` list, remove:

```md
- npm/npx wrapper around the native binary
```

- [ ] **Step 4: Update release runbook**

In `docs/release.md`, under `Local Snapshot QA`, after `bun run build`, add:

```bash
./bin/mojify doctor
```

Under `Release Smoke Test`, after `mojify --version`, add:

```bash
mojify doctor
```

- [ ] **Step 5: Run docs diff check**

Run:

```bash
git diff --check
```

Expected: PASS.

- [ ] **Step 6: Commit docs**

```bash
git add docs/adr/0030-add-runtime-doctor-command.md CONTEXT.md README.md docs/release.md
git commit --no-gpg-sign -m "docs: document doctor command"
```

## Task 6: Full Verification

**Files:**
- All files changed in previous tasks
- Modify: `docs/superpowers/plans/2026-06-05-mojify-doctor-command.md`

- [ ] **Step 1: Run formatting**

Run:

```bash
bun run fmt:check
```

Expected: PASS.

- [ ] **Step 2: Run Go tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...
```

Expected: PASS.

- [ ] **Step 3: Run repo tests**

Run:

```bash
bun run test
```

Expected: PASS.

- [ ] **Step 4: Build binary**

Run:

```bash
bun run build
```

Expected: PASS and `bin/mojify` exists.

- [ ] **Step 5: Smoke-test doctor**

Run:

```bash
./bin/mojify doctor
```

Expected: command prints one row each for `ffmpeg`, `ffprobe`, `ffplay`, and `yt-dlp`. Exit code may be non-zero on a host missing required tools, but on the normal development machine it should exit `0`.

- [ ] **Step 6: Run final diff checks**

Run:

```bash
git diff --check
git status -sb
```

Expected: `git diff --check` passes. `git status -sb` shows only intentional modified files and this plan file if steps were checked off.

- [ ] **Step 7: Commit finalized plan state**

If this plan file was updated while executing checkboxes, commit it:

```bash
git add docs/superpowers/plans/2026-06-05-mojify-doctor-command.md
git commit --no-gpg-sign -m "docs: finalize doctor command plan"
```

If the plan file was not changed during execution, skip this commit.

## Self-Review

- Spec coverage: The plan implements `mojify doctor`, checks all four tools, uses required versus optional severities, prints stable rows and summary, exits non-zero for required failures, avoids real host tools in tests, and updates README/release/CONTEXT/ADR docs.
- Scope check: The plan does not add install automation, network checks, sample rendering, audio device checks, terminal capability checks, JSON output, machine-readable diagnostics, or Windows-native dependency discovery.
- Type consistency: `doctor.Options`, `doctor.Run`, `doctor.Write`, `doctor.Report.OK`, `doctor.Report.Summary`, and `cli.runDoctorWithOptions` are introduced before use in later tasks.
