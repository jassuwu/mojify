# Mojify Binary Release Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Mojify installable as a prebuilt CLI through GitHub Release tarballs and `brew install jassuwu/tap/mojify`.

**Architecture:** Keep distribution binary-first and CLI-scoped. GoReleaser builds the `mojify` binary for macOS/Linux, injects the calendar build version, publishes checksummed release archives, and updates the separate Homebrew tap; GitHub Actions provides tag-only publishing and main-branch snapshot release QA. The product command surface stays `play`, `probe`, and `export`, with only version output added.

**Tech Stack:** Go 1.23, Bun/Turbo for existing repo scripts, GoReleaser v2, GitHub Actions, Homebrew tap formula generation, FFmpeg/yt-dlp as external runtime dependencies.

---

## File Structure

- `packages/core/internal/cli/version.go`: new version metadata and user-facing version output helpers.
- `packages/core/internal/cli/cli.go`: add `VersionCommand`, parse `mojify --version` and `mojify version`, and mention version output in help text.
- `packages/core/internal/cli/cli_test.go`: add parser/help/version-output tests.
- `packages/core/cmd/mojify/main.go`: dispatch `VersionCommand`.
- `packages/core/internal/media/tools.go`: new shared runtime dependency hint helpers for external tools.
- `packages/core/internal/media/tools_test.go`: tests for missing-tool formatting.
- `packages/core/internal/media/probe.go`: use runtime dependency hints for missing `ffprobe`.
- `packages/core/internal/media/decode.go`: use runtime dependency hints for missing `ffmpeg`.
- `packages/core/internal/media/audio.go`: use runtime dependency hints for missing `ffplay`.
- `.goreleaser.yaml`: GoReleaser config for macOS/Linux archives, checksums, GitHub Release metadata, and Homebrew tap formula publishing.
- `.github/workflows/ci.yml`: add non-publishing GoReleaser snapshot QA on main/PR.
- `.github/workflows/release.yml`: new tag-only release workflow for `v0.YYYYMMDD.BUILD` tags.
- `docs/release.md`: public-safe operator runbook for tap setup, snapshot QA, tagging, and smoke tests.
- `README.md`: update install instructions and distribution status.

## Task 1: Add User-Facing Version Output

**Files:**
- Create: `packages/core/internal/cli/version.go`
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Write failing version parser and output tests**

Append these tests to `packages/core/internal/cli/cli_test.go`:

```go
func TestParseVersionCommands(t *testing.T) {
	for _, args := range [][]string{
		{"--version"},
		{"version"},
	} {
		cmd, err := Parse(args)
		if err != nil {
			t.Fatalf("Parse(%v) returned error: %v", args, err)
		}
		if cmd.Kind != VersionCommand {
			t.Fatalf("Kind = %v, want %v for args %v", cmd.Kind, VersionCommand, args)
		}
	}
}

func TestVersionTextUsesInjectedVersion(t *testing.T) {
	oldVersion := version
	t.Cleanup(func() {
		version = oldVersion
	})

	version = "v0.20260602.145"

	got := VersionText()
	want := "mojify 0.20260602.145\n"
	if got != want {
		t.Fatalf("VersionText() = %q, want %q", got, want)
	}
}

func TestVersionTextFallsBackForSourceBuild(t *testing.T) {
	oldVersion := version
	t.Cleanup(func() {
		version = oldVersion
	})

	version = ""

	got := VersionText()
	want := "mojify 0.0.0-dev\n"
	if got != want {
		t.Fatalf("VersionText() = %q, want %q", got, want)
	}
}
```

Also add this expected help-text fragment inside `TestHelpTextMentionsCommands`:

```go
"mojify --version",
```

- [ ] **Step 2: Run the version tests and verify they fail**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected: FAIL because `VersionCommand`, `version`, and `VersionText` do not exist yet.

- [ ] **Step 3: Add version metadata helpers**

Create `packages/core/internal/cli/version.go`:

```go
package cli

import "strings"

const fallbackVersion = "0.0.0-dev"

var (
	version = fallbackVersion
	commit  = ""
	date    = ""
)

func Version() string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return fallbackVersion
	}
	return strings.TrimPrefix(trimmed, "v")
}

func VersionText() string {
	return "mojify " + Version() + "\n"
}
```

- [ ] **Step 4: Add `VersionCommand` parsing and help text**

In `packages/core/internal/cli/cli.go`, update the `CommandKind` constants to include `VersionCommand`:

```go
const (
	HelpCommand CommandKind = iota
	VersionCommand
	PlayCommand
	ProbeCommand
	ExportCommand
)
```

In `Parse`, add version handling before the subcommand cases:

```go
	switch args[0] {
	case "-h", "--help", "help":
		return Command{Kind: HelpCommand}, nil
	case "--version", "version":
		return Command{Kind: VersionCommand}, nil
	case "play":
		return parseInputCommand(PlayCommand, args)
	case "probe":
		return parseInputCommand(ProbeCommand, args)
	case "export":
		return parseExportCommand(args)
	default:
		return Command{}, fmt.Errorf("unknown command %q", args[0])
	}
```

In `HelpText`, add the version line under usage:

```text
  mojify --version                                      Print the installed Mojify version
```

- [ ] **Step 5: Dispatch the version command**

In `packages/core/cmd/mojify/main.go`, add this switch branch after `HelpCommand`:

```go
	case cli.VersionCommand:
		fmt.Print(cli.VersionText())
```

- [ ] **Step 6: Run version tests and verify they pass**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected: PASS.

- [ ] **Step 7: Run the local version smoke**

Run:

```bash
go run ./packages/core/cmd/mojify --version
```

Expected:

```text
mojify 0.0.0-dev
```

- [ ] **Step 8: Commit version output**

```bash
git add packages/core/internal/cli/version.go packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go packages/core/cmd/mojify/main.go
git commit -m "feat: add version output"
```

## Task 2: Add Runtime Dependency Hints

**Files:**
- Create: `packages/core/internal/media/tools.go`
- Create: `packages/core/internal/media/tools_test.go`
- Modify: `packages/core/internal/media/probe.go`
- Modify: `packages/core/internal/media/decode.go`
- Modify: `packages/core/internal/media/audio.go`

- [ ] **Step 1: Write failing runtime dependency hint tests**

Create `packages/core/internal/media/tools_test.go`:

```go
package media

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestFormatToolFailureMissingTool(t *testing.T) {
	err := formatToolFailure("ffmpeg", &exec.Error{Name: "ffmpeg", Err: exec.ErrNotFound}, "")
	if err == nil {
		t.Fatal("formatToolFailure returned nil")
	}
	got := err.Error()
	for _, want := range []string{
		"ffmpeg is required",
		"install ffmpeg",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}

func TestFormatToolFailurePreservesStderr(t *testing.T) {
	err := formatToolFailure("ffprobe", errors.New("exit status 1"), "invalid data")
	if err == nil {
		t.Fatal("formatToolFailure returned nil")
	}
	got := err.Error()
	want := "ffprobe failed: invalid data"
	if got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestFormatToolStartErrorMissingTool(t *testing.T) {
	err := formatToolStartError("ffplay", &exec.Error{Name: "ffplay", Err: exec.ErrNotFound})
	if err == nil {
		t.Fatal("formatToolStartError returned nil")
	}
	got := err.Error()
	for _, want := range []string{
		"ffplay is required",
		"install ffplay",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}
```

- [ ] **Step 2: Run the media tests and verify they fail**

Run:

```bash
go test ./packages/core/internal/media
```

Expected: FAIL because `formatToolFailure` and `formatToolStartError` do not exist.

- [ ] **Step 3: Add tool error helpers**

Create `packages/core/internal/media/tools.go`:

```go
package media

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func formatToolFailure(tool string, err error, stderr string) error {
	if isMissingExecutable(err) {
		return missingToolError(tool)
	}
	stderr = strings.TrimSpace(stderr)
	if stderr != "" {
		return fmt.Errorf("%s failed: %s", tool, stderr)
	}
	return fmt.Errorf("%s failed: %w", tool, err)
}

func formatToolStartError(tool string, err error) error {
	if isMissingExecutable(err) {
		return missingToolError(tool)
	}
	return err
}

func missingToolError(tool string) error {
	return fmt.Errorf("%s is required; install %s and try again", tool, tool)
}

func isMissingExecutable(err error) bool {
	var execErr *exec.Error
	return errors.As(err, &execErr)
}
```

- [ ] **Step 4: Use hints in `ffprobe` errors**

In `packages/core/internal/media/probe.go`, replace the current `if err != nil` block in `ProbeContext` with:

```go
	if err != nil {
		return Info{}, formatToolFailure("ffprobe", err, stderr.String())
	}
```

- [ ] **Step 5: Use hints in `ffmpeg` decoder startup**

In `packages/core/internal/media/decode.go`, update both `cmd.Start()` error returns:

```go
	if err := cmd.Start(); err != nil {
		return nil, nil, formatToolStartError("ffmpeg", err)
	}
```

Apply the same replacement in `StartDecoderContext` and `StartExportDecoderContext`.

- [ ] **Step 6: Use hints in `ffplay` audio startup**

In `packages/core/internal/media/audio.go`, update the `cmd.Start()` error return:

```go
	if err := cmd.Start(); err != nil {
		return nil, nil, formatToolStartError("ffplay", err)
	}
```

- [ ] **Step 7: Run media tests and verify they pass**

Run:

```bash
go test ./packages/core/internal/media
```

Expected: PASS.

- [ ] **Step 8: Run all Go tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 9: Commit runtime dependency hints**

```bash
git add packages/core/internal/media/tools.go packages/core/internal/media/tools_test.go packages/core/internal/media/probe.go packages/core/internal/media/decode.go packages/core/internal/media/audio.go
git commit -m "fix: add runtime dependency hints"
```

## Task 3: Add GoReleaser Configuration

**Files:**
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Create GoReleaser config**

Create `.goreleaser.yaml`:

```yaml
version: 2

project_name: mojify

before:
  hooks:
    - go mod tidy

builds:
  - id: mojify
    main: ./packages/core/cmd/mojify
    binary: mojify
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - >-
        -s -w
        -X github.com/jass/mojify/packages/core/internal/cli.version={{ .Version }}
        -X github.com/jass/mojify/packages/core/internal/cli.commit={{ .Commit }}
        -X github.com/jass/mojify/packages/core/internal/cli.date={{ .Date }}

archives:
  - id: mojify
    ids:
      - mojify
    formats:
      - tar.gz
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    files:
      - LICENSE
      - README.md

checksum:
  name_template: checksums.txt
  algorithm: sha256

changelog:
  disable: "{{ .IsSnapshot }}"
  use: github-native

release:
  github:
    owner: jassuwu
    name: mojify
  name_template: "{{ .ProjectName }} {{ .Tag }}"

brews:
  - name: mojify
    ids:
      - mojify
    repository:
      owner: jassuwu
      name: homebrew-tap
      branch: main
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    directory: Formula
    skip_upload: "{{ .IsSnapshot }}"
    homepage: "https://github.com/jassuwu/mojify"
    description: "Terminal-first video playback with colored, edge-aware character frames."
    license: MIT
    dependencies:
      - name: ffmpeg
      - name: yt-dlp
    install: |
      bin.install "mojify"
    test: |
      system "#{bin}/mojify", "--version"
```

- [ ] **Step 2: Validate the config locally**

Run:

```bash
goreleaser check
```

Expected: PASS with no config validation errors.

If the command is unavailable, install GoReleaser locally first:

```bash
brew install goreleaser
```

- [ ] **Step 3: Run token-free snapshot release QA**

Run:

```bash
goreleaser release --snapshot --clean
```

Expected:

```text
dist/checksums.txt
dist/mojify_*.tar.gz
```

No GitHub Release is published and no tap repo is touched.

- [ ] **Step 4: Smoke-test the snapshot binary version**

Run:

```bash
tar -tzf dist/mojify_*_darwin_arm64.tar.gz
tar -xzf dist/mojify_*_darwin_arm64.tar.gz -C /tmp mojify
/tmp/mojify --version
```

Expected:

```text
mojify 0.0.0-SNAPSHOT-...
```

The exact snapshot suffix is generated by GoReleaser; the important behavior is that the command prints a single `mojify VERSION` line.

- [ ] **Step 5: Commit GoReleaser config**

```bash
git add .goreleaser.yaml
git commit -m "build: add goreleaser config"
```

## Task 4: Add Release CI and Tag Publishing Workflow

**Files:**
- Modify: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Add snapshot release QA to CI**

In `.github/workflows/ci.yml`, add this step after the existing `Build` step:

```yaml
      - name: GoReleaser snapshot
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --snapshot --clean
```

- [ ] **Step 2: Create tag-only release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v0.*.*"

permissions:
  contents: write

jobs:
  release:
    name: Publish CLI release
    runs-on: ubuntu-latest
    timeout-minutes: 20
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Validate release tag
        shell: bash
        run: |
          set -euo pipefail
          if [[ ! "$GITHUB_REF_NAME" =~ ^v0\.[0-9]{8}\.[0-9]+$ ]]; then
            echo "Invalid Mojify release tag: $GITHUB_REF_NAME" >&2
            echo "Expected v0.YYYYMMDD.BUILD, for example v0.20260602.145" >&2
            exit 1
          fi

      - name: Install FFmpeg
        run: sudo apt-get update && sudo apt-get install -y ffmpeg

      - name: Test
        run: go test ./...

      - name: Build GoReleaser smoke artifact
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: build --clean --single-target

      - name: Version smoke
        shell: bash
        run: |
          set -euo pipefail
          expected="mojify ${GITHUB_REF_NAME#v}"
          binary="$(find dist -type f -name mojify -perm -111 | head -n 1)"
          if [[ -z "$binary" ]]; then
            echo "GoReleaser smoke build did not produce a mojify binary." >&2
            exit 1
          fi
          actual="$("$binary" --version)"
          if [[ "$actual" != "$expected" ]]; then
            echo "Unexpected version output from $binary" >&2
            echo "Expected: $expected" >&2
            echo "Actual:   $actual" >&2
            exit 1
          fi

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

- [ ] **Step 3: Verify workflow syntax with git diff**

Run:

```bash
git diff -- .github/workflows/ci.yml .github/workflows/release.yml
```

Expected: CI includes one non-publishing snapshot step; release workflow only triggers on `v0.*.*` tags and has no `workflow_dispatch` or `schedule`.

- [ ] **Step 4: Commit workflows**

```bash
git add .github/workflows/ci.yml .github/workflows/release.yml
git commit -m "ci: add binary release workflows"
```

## Task 5: Add Public-Safe Release Runbook and Install Docs

**Files:**
- Create: `docs/release.md`
- Modify: `README.md`

- [ ] **Step 1: Create release runbook**

Create `docs/release.md`:

````markdown
# Release Runbook

Mojify publishes CLI binary releases from stable tags in the shape `v0.YYYYMMDD.BUILD`, for example `v0.20260602.145`.

This document is public-safe. It names repositories, GitHub Actions secret names, and token scope requirements, but never records token values.

## Supported Install Paths

- Homebrew: `brew install jassuwu/tap/mojify`
- GitHub Release tarballs from `jassuwu/mojify`

Not supported in this stage:

- npm or npx
- native Windows binaries
- Linux distro packages
- `go install`
- signed or notarized artifacts

Windows users should use Mojify inside WSL for this stage.

## One-Time Tap Setup

1. Create a public GitHub repository named `jassuwu/homebrew-tap`.
2. Use `main` as the default branch.
3. Do not add token values or private account material to this repository.
4. Create a fine-grained GitHub token that can write contents to `jassuwu/homebrew-tap`.
5. In `jassuwu/mojify`, add the token as a GitHub Actions secret named `TAP_GITHUB_TOKEN`.

The built-in `GITHUB_TOKEN` publishes releases in `jassuwu/mojify`. `TAP_GITHUB_TOKEN` is only for pushing formula updates to `jassuwu/homebrew-tap`.

## Local Snapshot QA

Install local release tooling:

```bash
brew install goreleaser ffmpeg yt-dlp
```

Run the normal checks:

```bash
go test ./...
goreleaser check
goreleaser release --snapshot --clean
```

Inspect the generated artifacts:

```bash
ls dist
cat dist/checksums.txt
tar -tzf dist/mojify_*_darwin_arm64.tar.gz
```

Smoke-test a generated binary:

```bash
tar -xzf dist/mojify_*_darwin_arm64.tar.gz -C /tmp mojify
/tmp/mojify --version
```

Expected shape:

```text
mojify 0.0.0-SNAPSHOT-...
```

Snapshot QA must not require GitHub tokens and must not publish a GitHub Release or update the Homebrew tap.

## Stable Release

Choose a calendar build version:

```bash
git tag v0.YYYYMMDD.BUILD
```

Example:

```bash
git tag v0.20260602.145
git push origin v0.20260602.145
```

The tag workflow publishes:

- macOS arm64 tarball
- macOS amd64 tarball
- Linux arm64 tarball
- Linux amd64 tarball
- `checksums.txt`
- generated GitHub Release notes
- updated Homebrew formula in `jassuwu/homebrew-tap`

## Release Smoke Test

After the workflow passes, test Homebrew installation:

```bash
brew update
brew install jassuwu/tap/mojify
mojify --version
bun run qa:clips
mojify probe ./dist/qa/low-motion-bars.mp4
```

Expected version shape:

```text
mojify 0.YYYYMMDD.BUILD
```

If the formula update fails, check:

- `TAP_GITHUB_TOKEN` exists in `jassuwu/mojify` repository secrets.
- The token can write contents to `jassuwu/homebrew-tap`.
- The tap repository exists and has a `main` branch.

## Runtime Dependencies

Homebrew installs declare:

- `ffmpeg`
- `yt-dlp`

Tarball users must install:

- `ffmpeg`
- `ffprobe`
- `ffplay`
- `yt-dlp` for platform URL inputs

Mojify does not bundle these tools and does not download them at runtime.
````

- [ ] **Step 2: Update README install section**

In `README.md`, replace the current `## Status` section with:

```markdown
## Installation

Mojify is distributed as a prebuilt CLI for macOS and Linux.

### Homebrew

```bash
brew install jassuwu/tap/mojify
```

### GitHub Releases

Download the matching tarball for your platform from the [GitHub Releases](https://github.com/jassuwu/mojify/releases), then place `mojify` on your `PATH`.

Windows support is WSL-only for now. Native Windows binaries are deferred.
```

Keep the existing playback/export/scope content after the new installation section, but split user runtime requirements from source-development requirements:

```markdown
## Requirements

- FFmpeg and ffprobe on `PATH`
- yt-dlp on `PATH` for platform URL inputs
- ffplay on `PATH` for live playback audio

Homebrew installs declare `ffmpeg` and `yt-dlp`. Tarball installs require these tools to be installed separately.

## Development Requirements

- Go 1.23+
- Bun 1.3+

## Run From Source
```

- [ ] **Step 3: Verify docs do not contain secret values**

Run:

```bash
rg -n "ghp_|github_pat_|TAP_GITHUB_TOKEN=.*|BEGIN .*PRIVATE KEY|password|secret value" docs/release.md README.md
```

Expected: no matches for actual token/private material. The plain secret name `TAP_GITHUB_TOKEN` is allowed.

- [ ] **Step 4: Commit release docs**

```bash
git add docs/release.md README.md
git commit -m "docs: add release runbook"
```

## Task 6: Final Verification

**Files:**
- Verify all files changed by Tasks 1-5.

- [ ] **Step 1: Run format check**

Run:

```bash
bun run fmt:check
```

Expected: PASS.

- [ ] **Step 2: Run Go tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run monorepo tests**

Run:

```bash
bun run test
```

Expected: PASS.

- [ ] **Step 4: Run build**

Run:

```bash
bun run build
```

Expected: PASS and `bin/mojify` exists.

- [ ] **Step 5: Run version smoke on the built binary**

Run:

```bash
./bin/mojify --version
```

Expected:

```text
mojify 0.0.0-dev
```

- [ ] **Step 6: Run GoReleaser config check**

Run:

```bash
goreleaser check
```

Expected: PASS.

- [ ] **Step 7: Run token-free snapshot release QA**

Run:

```bash
goreleaser release --snapshot --clean
```

Expected: PASS, with `dist/checksums.txt` and four `dist/mojify_*.tar.gz` archives.

- [ ] **Step 8: Inspect the generated Homebrew formula locally**

Run:

```bash
rg -n "depends_on|bin.install|system" dist
```

Expected generated formula content includes:

```text
depends_on "ffmpeg"
depends_on "yt-dlp"
bin.install "mojify"
system "#{bin}/mojify", "--version"
```

- [ ] **Step 9: Run git status**

Run:

```bash
git status --short
```

Expected: no uncommitted implementation changes other than any intentionally uncommitted plan/ADR docs if the branch policy keeps planning docs separate.

- [ ] **Step 10: Final commit if needed**

If any implementation files remain unstaged after the task commits, commit them:

```bash
git add .
git commit -m "chore: finalize binary release distribution"
```
