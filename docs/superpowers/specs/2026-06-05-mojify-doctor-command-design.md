# Mojify Doctor Command Design

Approved design for adding a lightweight runtime dependency check command. This spec does not implement the feature.

## Goal

Add `mojify doctor` so users can quickly verify whether the external tools Mojify shells out to are available on `PATH`.

This is a CLI polish and supportability stage. It supersedes the earlier distribution-stage deferral of a doctor command because Mojify now has real Homebrew, bottle, and tarball install paths, and users need a simple way to diagnose local runtime setup.

## User Surface

`mojify doctor` prints a compact status report to stdout:

```text
mojify doctor

ok    ffmpeg   8.0.1
ok    ffprobe  8.0.1
warn  ffplay   missing; live playback audio will be unavailable
ok    yt-dlp   2026.05.22

Mojify can play and export local media.
```

The exact version string may be normalized from each tool's native version output. The command should prefer useful short versions over dumping full banner text.

`mojify doctor` accepts no flags in this stage. Users should use `mojify --help` for command discovery.

## Checks

Doctor checks these tools:

| Tool | Severity if missing | Why |
| --- | --- | --- |
| `ffmpeg` | error | Required for decode and export encoding. |
| `ffprobe` | error | Required for metadata probing and layout decisions. |
| `ffplay` | warning | Required only for live playback audio. Visual playback can still run with `--no-audio`. |
| `yt-dlp` | warning | Required only for platform URL inputs. Local file workflows can still run without it. |

Each check should run a cheap version command with a timeout:

- `ffmpeg -version`
- `ffprobe -version`
- `ffplay -version`
- `yt-dlp --version`

If a command is missing, doctor should report `missing` rather than surfacing raw `exec` internals. If a command exists but fails or times out, doctor should report a warning or error with concise context based on the tool's severity.

## Exit Codes

`mojify doctor` exits:

- `0` when `ffmpeg` and `ffprobe` are available, even if optional tools are missing.
- non-zero when `ffmpeg` or `ffprobe` is missing, fails its version check, or times out.

This keeps local media playback/export as the baseline health contract while still warning about audio and URL-only capability gaps.

## Architecture

Add a CLI command kind for `DoctorCommand`.

The dependency-checking logic should live in a small internal package or module that can be tested without invoking real tools. It should accept an injectable command runner so tests can cover:

- present tool with version output
- missing executable
- command failure with stderr
- timeout
- required versus optional severity

The existing runtime missing-tool formatting in `packages/core/internal/media/tools.go` should remain focused on media command failures. Doctor can reuse concepts, but it should not force media packages to own CLI health reporting.

`Run` should dispatch `DoctorCommand` without requiring source resolution, probing, terminal setup, or media decoding.

## Output Rules

Doctor output should be terminal-friendly and log-friendly:

- One row per tool.
- Stable labels: `ok`, `warn`, `error`.
- No ANSI color required in this stage.
- No spinners, progress bars, or interactive prompts.
- Summary sentence at the end that states the highest useful capability level:
  - local media can play/export when required tools are healthy
  - local media is blocked when required tools fail
  - URL input needs `yt-dlp` when that warning is present
  - live audio needs `ffplay` when that warning is present

## Documentation

Update:

- `README.md` requirements or quickstart section with `mojify doctor`.
- `docs/release.md` tarball smoke test instructions to include `mojify doctor`.
- `CONTEXT.md` with a glossary entry for runtime doctor.
- a new ADR recording that `mojify doctor` is now accepted as a post-distribution CLI polish stage.

## Out Of Scope

- Installing tools automatically.
- Homebrew-specific repair advice beyond generic install hints.
- Network checks.
- yt-dlp platform smoke downloads.
- Rendering a sample video.
- Checking audio devices.
- Checking terminal feature support.
- JSON output.
- Machine-readable diagnostics.
- Windows-native dependency discovery.

## Acceptance Criteria

- `mojify doctor` parses as a first-class command.
- `mojify --help` lists `doctor`.
- Required tool failures produce non-zero exit.
- Missing `ffplay` and missing `yt-dlp` produce warnings but do not fail the command when required tools are healthy.
- Version output is concise and stable enough for tests.
- Tests do not depend on the host machine having or missing real external tools.
- Documentation explains that doctor checks external runtime tools but does not install or download anything.
