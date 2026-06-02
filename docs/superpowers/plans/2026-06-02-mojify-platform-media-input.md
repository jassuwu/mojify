# Platform Media Input Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let `mojify probe`, `mojify play`, and `mojify export` accept HTTP(S), yt-dlp-compatible platform URLs by resolving each URL into temporary local source media before using the existing path-based media pipeline.

**Architecture:** Keep `media`, `player`, and `exporter` operating on local file paths. Add a CLI-owned source resolver that detects HTTP(S) sources, runs yt-dlp in a fresh temp directory, validates one finite non-playlist video, captures the final downloaded filepath from yt-dlp, and cleans it up after the command exits. Add `cli.RunProbe` so probe shares the same source-resolution boundary as play and export.

**Tech Stack:** Go 1.23, yt-dlp CLI, FFmpeg/ffprobe/ffplay CLI, Bun/Turbo verification scripts, fake executable tests for yt-dlp contract coverage.

---

## Decisions Already Made

- Stage name: **Platform media input**.
- CLI argument name: **Source**.
- A source can be a local video file or an HTTP(S), yt-dlp-compatible platform URL.
- Platform URL support is download-first, not stream-first.
- `probe`, `play`, and `export` accept platform URLs anywhere they currently accept an input source.
- Export output paths remain local `.mp4` paths only.
- Platform URL downloads are temporary per command and cleaned up after command completion, failure, or cancellation.
- No persistent cache, no `--keep-source`, and no new CLI flags.
- Platform URL means only `http://` or `https://`.
- Reject playlist workflow and live streams for this stage.
- yt-dlp is required only when the user passes a platform URL.
- Use `--ignore-config` so user yt-dlp config cannot silently change Mojify's source-resolution contract.
- Ask yt-dlp for one merged playable result, prefer MP4-compatible selections, and treat `--print after_move:filepath` output as authoritative.
- Print simple phase status only; do not parse yt-dlp percentage or ETA.
- Local file inputs stay silent and bypass yt-dlp entirely.
- Automated tests must use a fake yt-dlp executable and must not hit the network.

## File Structure

- Modify: `CONTEXT.md`
  - Already updated during grill with `Source`, `Source media`, and `Platform media input`.
- Create: `docs/adr/0026-resolve-platform-media-input-in-cli.md`
  - Already created during grill; keep committed with this stage.
- Create: `docs/qa/platform-media-input.md`
  - Already created during grill; keep committed with this stage.
- Create: `packages/core/internal/cli/source.go`
  - Own source classification, yt-dlp metadata preflight, temporary download, status messages, and cleanup.
- Create: `packages/core/internal/cli/source_test.go`
  - Lock local bypass, HTTP(S) detection, yt-dlp args, final filepath parsing, failure messages, playlist/live rejection, and cleanup.
- Modify: `packages/core/internal/cli/cli.go`
  - Replace local-only input rejection with HTTP(S)-source acceptance, keep unsupported protocol rejection, update help text to use `<source>`.
- Modify: `packages/core/internal/cli/cli_test.go`
  - Update parser tests for HTTP(S) input acceptance, unsupported protocol rejection, output protocol rejection, and help wording.
- Create: `packages/core/internal/cli/probe.go`
  - Move probe command execution and output formatting out of `main.go`, including source resolution and optional `resolved-source` output.
- Create: `packages/core/internal/cli/probe_test.go`
  - Lock probe output formatting for local and resolved platform sources.
- Modify: `packages/core/internal/cli/play.go`
  - Resolve source media before probing/decoding/starting ffplay audio; local files remain quiet.
- Modify: `packages/core/internal/cli/export.go`
  - Resolve source media before exporter preflight and frame processing; output validation remains parser-owned before resolution.
- Modify: `packages/core/cmd/mojify/main.go`
  - Use signal-aware contexts for probe/play/export and dispatch `cli.RunProbe`, `cli.RunPlay`, and `cli.RunExport`.
- Modify: `README.md`
  - Update requirements, examples, and scope to include platform URL input and `yt-dlp` URL-only dependency.
- Modify: `docs/qa/playback-quality.md`
  - Link to platform media input QA for URL playback cases, without duplicating the whole cross-command checklist.
- Modify: `docs/qa/export.md`
  - Link to platform media input QA for URL export cases, without duplicating the whole cross-command checklist.

---

### Task 1: CLI Source Syntax and Help Text

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Add failing parser tests for HTTP(S) sources**

Append these tests to `packages/core/internal/cli/cli_test.go` near the existing protocol-input tests:

```go
func TestParseAcceptsHTTPSources(t *testing.T) {
	for _, command := range []string{"play", "probe"} {
		for _, input := range []string{
			"https://example.com/watch?v=demo",
			"http://example.com/video",
		} {
			cmd, err := Parse([]string{command, input})
			if err != nil {
				t.Fatalf("Parse(%s %q) returned error: %v", command, input, err)
			}
			if cmd.InputPath != input {
				t.Fatalf("InputPath = %q, want %q", cmd.InputPath, input)
			}
		}
	}
}

func TestParseExportAcceptsHTTPSource(t *testing.T) {
	cmd, err := Parse([]string{"export", "https://example.com/watch?v=demo", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.InputPath != "https://example.com/watch?v=demo" {
		t.Fatalf("InputPath = %q, want URL source", cmd.InputPath)
	}
	if cmd.OutputPath != "clip.mp4" {
		t.Fatalf("OutputPath = %q, want clip.mp4", cmd.OutputPath)
	}
}
```

- [ ] **Step 2: Replace the local-only protocol rejection tests**

Replace `TestParseRejectsProtocolInputs` with:

```go
func TestParseRejectsUnsupportedProtocolInputs(t *testing.T) {
	for _, command := range []string{"play", "probe"} {
		for _, input := range []string{
			"file:///tmp/demo.mp4",
			"pipe:0",
			"concat:part1.mp4|part2.mp4",
			"ytsearch:demo query",
			"-",
		} {
			_, err := Parse([]string{command, input})
			if err == nil {
				t.Fatalf("Parse accepted unsupported %s input %q", command, input)
			}
		}
	}
}
```

Replace `TestParseExportRejectsProtocolInput` with:

```go
func TestParseExportRejectsUnsupportedProtocolInput(t *testing.T) {
	for _, input := range []string{
		"file:///tmp/demo.mp4",
		"pipe:0",
		"concat:part1.mp4|part2.mp4",
		"ytsearch:demo query",
		"-",
	} {
		_, err := Parse([]string{"export", input, "clip.mp4"})
		if err == nil {
			t.Fatalf("Parse returned nil error for unsupported export input %q", input)
		}
	}
}
```

- [ ] **Step 3: Update the help text test**

In `TestHelpTextMentionsCommands`, replace the command/input expectations with:

```go
for _, want := range []string{
	"mojify play [--stats] [--no-audio] <source>",
	"Play source media in the terminal",
	"mojify probe <source>",
	"Print source media and render metadata",
	"mojify export [options] <source> <output.mp4>",
	"Export Mojify visuals to an MP4 file",
	"<source> may be a local video file or an HTTP(S) platform URL",
	"yt-dlp is required for platform URL inputs",
	"--width <px>",
	"--fps <n>",
	"--bitrate <value>",
	"--overwrite",
	"--stats",
	"--no-audio",
	"--workers <n>",
	"FFmpeg and ffprobe",
	"ffplay is required for live playback audio unless --no-audio is used",
} {
	if !contains(help, want) {
		t.Fatalf("HelpText() missing %q in:\n%s", want, help)
	}
}
```

- [ ] **Step 4: Run parser tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestParse|TestHelpText'
```

Expected: fails because HTTP(S) sources are still rejected and help still says `<video>`.

- [ ] **Step 5: Implement HTTP(S) source parsing and help text**

In `packages/core/internal/cli/cli.go`, update `HelpText` to:

```go
func HelpText() string {
	return `mojify

Terminal-first video playback with colored, edge-aware character frames.

Usage:
  mojify play [--stats] [--no-audio] <source>           Play source media in the terminal
  mojify probe <source>                                 Print source media and render metadata
  mojify export [options] <source> <output.mp4>         Export Mojify visuals to an MP4 file
  mojify --help                                         Show this help

Source:
  <source> may be a local video file or an HTTP(S) platform URL.

Play options:
  --stats             Print playback timing stats after completion
  --no-audio          Disable live playback audio for play

Export options:
  --width <px>        Output MP4 width in pixels
  --fps <n>           Output frames per second
  --bitrate <value>   Video bitrate, digits optionally followed by k, K, m, or M
  --overwrite         Replace an existing output file
  --stats             Print export timing stats after completion
  --workers <n>       Render and rasterize with n workers

Requirements:
  FFmpeg and ffprobe must be available on PATH.
  yt-dlp is required for platform URL inputs.
  ffplay is required for live playback audio unless --no-audio is used.
`
}
```

Replace the input protocol rejection in `parseInputCommand`:

```go
if hasUnsupportedSourceProtocol(inputPath) {
	return Command{}, fmt.Errorf("%s accepts local video file paths or HTTP(S) platform URLs only", args[0])
}
```

Replace the export input protocol rejection:

```go
if hasUnsupportedSourceProtocol(paths[0]) {
	return Command{}, fmt.Errorf("export accepts local video file paths or HTTP(S) platform URLs only")
}
```

Add these helpers near `hasProtocolInput`:

```go
func isHTTPSource(input string) bool {
	lower := strings.ToLower(input)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func hasUnsupportedSourceProtocol(input string) bool {
	if isHTTPSource(input) {
		return false
	}
	return hasProtocolInput(input)
}
```

Keep `hasProtocolInput` unchanged for export output validation.

- [ ] **Step 6: Run parser tests and verify they pass**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestParse|TestHelpText'
```

Expected: parser and help tests pass.

- [ ] **Step 7: Commit parser/help changes**

Run:

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go
git commit --no-gpg-sign -m "feat: accept platform sources in cli syntax"
```

---

### Task 2: Source Resolver and yt-dlp Contract

**Files:**
- Create: `packages/core/internal/cli/source.go`
- Create: `packages/core/internal/cli/source_test.go`

- [ ] **Step 1: Create failing source resolver tests**

Create `packages/core/internal/cli/source_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveSourceMediaLocalBypassesYTDLP(t *testing.T) {
	var stderr bytes.Buffer
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "dist/clip.mp4", sourceResolverOptions{
		Stderr:    &stderr,
		YTDLPPath: "missing-yt-dlp-for-local-test",
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	defer resolved.Cleanup()
	if resolved.Path != "dist/clip.mp4" {
		t.Fatalf("Path = %q, want local path", resolved.Path)
	}
	if resolved.Original != "dist/clip.mp4" {
		t.Fatalf("Original = %q, want local source", resolved.Original)
	}
	if resolved.Temporary {
		t.Fatal("Temporary = true, want false for local source")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want no local source status", stderr.String())
	}
}

func TestResolveSourceMediaDownloadsHTTPSSource(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{})
	var stderr bytes.Buffer
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		Stderr:    &stderr,
		YTDLPPath: fake.Path,
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	downloadedPath := resolved.Path
	if !resolved.Temporary {
		t.Fatal("Temporary = false, want true for platform URL")
	}
	if resolved.Original != "https://example.com/watch?v=demo" {
		t.Fatalf("Original = %q, want original URL", resolved.Original)
	}
	if resolved.DisplayName != "Demo_Title [abc123].mp4" {
		t.Fatalf("DisplayName = %q, want final basename", resolved.DisplayName)
	}
	if _, err := os.Stat(downloadedPath); err != nil {
		t.Fatalf("downloaded path does not exist before cleanup: %v", err)
	}
	if err := resolved.Cleanup(); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if _, err := os.Stat(downloadedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("downloaded path still exists after cleanup, stat err = %v", err)
	}
	got := stderr.String()
	for _, want := range []string{
		"resolving source media: https://example.com/watch?v=demo\n",
		"downloading source media...\n",
		"source media ready: Demo_Title [abc123].mp4\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr missing %q in %q", want, got)
		}
	}
}

func TestResolveSourceMediaPassesExpectedYTDLPArgs(t *testing.T) {
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{ArgsPath: argsPath})
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	defer resolved.Cleanup()
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"--ignore-config",
		"--no-playlist",
		"--match-filters",
		"!is_live",
		"--no-progress",
		"--paths",
		"home:",
		"temp:",
		"--restrict-filenames",
		"--trim-filenames",
		"120",
		"--output",
		"%(title).120B [%(id)s].%(ext)s",
		"--print",
		"after_move:filepath",
		"--merge-output-format",
		"mp4",
		"-f",
		"bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/b",
		"https://example.com/watch?v=demo",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("yt-dlp args missing %q in:\n%s", want, got)
		}
	}
}

func TestResolveSourceMediaRejectsPlaylistMetadata(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{MetadataJSON: `{"_type":"playlist","entries":[{"id":"one"}]}`})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/playlist", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "playlists are not supported") {
		t.Fatalf("error = %v, want playlist rejection", err)
	}
}

func TestResolveSourceMediaRejectsLiveMetadata(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{MetadataJSON: `{"id":"live","is_live":true}`})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/live", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "live streams are not supported") {
		t.Fatalf("error = %v, want live stream rejection", err)
	}
}

func TestResolveSourceMediaMissingYTDLPHasConciseError(t *testing.T) {
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: "definitely-missing-yt-dlp",
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required for platform URLs") {
		t.Fatalf("error = %v, want missing yt-dlp message", err)
	}
}

func TestResolveSourceMediaYTDLPFailureIncludesStderr(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{
		FailDownload: true,
		FailureText:  "unsupported url",
	})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported url") {
		t.Fatalf("error = %v, want yt-dlp stderr context", err)
	}
}

func TestResolveSourceMediaRejectsEmptyFinalPath(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{EmptyFinalPath: true})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp did not report a downloaded source path") {
		t.Fatalf("error = %v, want empty final path rejection", err)
	}
}

type fakeYTDLP struct {
	Path string
}

type fakeYTDLPOptions struct {
	ArgsPath      string
	MetadataJSON  string
	FailDownload  bool
	FailureText   string
	EmptyFinalPath bool
}

func writeFakeYTDLP(t *testing.T, options fakeYTDLPOptions) fakeYTDLP {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "yt-dlp")
	if runtime.GOOS == "windows" {
		path += ".bat"
		t.Skip("fake yt-dlp bash script contract tests are Unix-only")
	}
	metadata := options.MetadataJSON
	if metadata == "" {
		metadata = `{"id":"abc123","title":"Demo Title","is_live":false}`
	}
	failure := options.FailureText
	if failure == "" {
		failure = "yt-dlp failed"
	}
	script := "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"args_path=" + shellQuote(options.ArgsPath) + "\n" +
		"if [[ -n \"${args_path}\" ]]; then printf '%s\\n' \"$*\" >> \"${args_path}\"; fi\n" +
		"for arg in \"$@\"; do\n" +
		"  if [[ \"${arg}\" == \"--dump-single-json\" ]]; then printf '%s\\n' " + shellQuote(metadata) + "; exit 0; fi\n" +
		"done\n" +
		"home=\"\"\n" +
		"prev=\"\"\n" +
		"for arg in \"$@\"; do\n" +
		"  if [[ \"${prev}\" == \"--paths\" && \"${arg}\" == home:* ]]; then home=\"${arg#home:}\"; fi\n" +
		"  prev=\"${arg}\"\n" +
		"done\n" +
		"if [[ -z \"${home}\" ]]; then echo 'missing home path' >&2; exit 7; fi\n" +
		"if [[ " + shellBool(options.FailDownload) + " == true ]]; then echo " + shellQuote(failure) + " >&2; exit 9; fi\n" +
		"out=\"${home}/Demo_Title [abc123].mp4\"\n" +
		"mkdir -p \"${home}\"\n" +
		"printf 'fake media' > \"${out}\"\n" +
		"if [[ " + shellBool(options.EmptyFinalPath) + " == false ]]; then printf '%s\\n' \"${out}\"; fi\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}
	return fakeYTDLP{Path: path}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func shellBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
```

- [ ] **Step 2: Run source tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestResolveSourceMedia'
```

Expected: fails because source resolver types and functions do not exist.

- [ ] **Step 3: Implement source resolver**

Create `packages/core/internal/cli/source.go`:

```go
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultYTDLPPath          = "yt-dlp"
	sourceFilenameTemplate   = "%(title).120B [%(id)s].%(ext)s"
	platformDownloadFormat   = "bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/b"
	platformMergeOutput      = "mp4"
	sourceTempDirectoryPrefix = "mojify-source-"
)

type resolvedSourceMedia struct {
	Original    string
	Path        string
	DisplayName string
	Temporary   bool
	Cleanup     func() error
}

type sourceResolverOptions struct {
	Stderr    io.Writer
	YTDLPPath string
}

type sourceMetadata struct {
	Type       string            `json:"_type"`
	IsLive     bool              `json:"is_live"`
	LiveStatus string            `json:"live_status"`
	Entries    []json.RawMessage `json:"entries"`
}

func resolveSourceMedia(ctx context.Context, source string, stderr io.Writer) (resolvedSourceMedia, error) {
	return resolveSourceMediaWithOptions(ctx, source, sourceResolverOptions{Stderr: stderr})
}

func resolveSourceMediaWithOptions(ctx context.Context, source string, options sourceResolverOptions) (resolvedSourceMedia, error) {
	if !isHTTPSource(source) {
		return resolvedSourceMedia{
			Original:    source,
			Path:        source,
			DisplayName: filepath.Base(source),
			Cleanup:     func() error { return nil },
		}, nil
	}

	ytdlpPath := options.YTDLPPath
	if ytdlpPath == "" {
		ytdlpPath = defaultYTDLPPath
	}
	status := options.Stderr
	if status != nil {
		fmt.Fprintf(status, "resolving source media: %s\n", source)
	}

	if err := preflightPlatformSource(ctx, ytdlpPath, source); err != nil {
		return resolvedSourceMedia{}, err
	}

	tempDir, err := os.MkdirTemp("", sourceTempDirectoryPrefix)
	if err != nil {
		return resolvedSourceMedia{}, fmt.Errorf("create source temp directory: %w", err)
	}
	cleanup := func() error {
		return os.RemoveAll(tempDir)
	}
	cleaned := false
	defer func() {
		if !cleaned {
			_ = cleanup()
		}
	}()

	if status != nil {
		fmt.Fprintln(status, "downloading source media...")
	}
	finalPath, err := downloadPlatformSource(ctx, ytdlpPath, tempDir, source)
	if err != nil {
		return resolvedSourceMedia{}, err
	}
	finalPath = strings.TrimSpace(finalPath)
	if finalPath == "" {
		return resolvedSourceMedia{}, fmt.Errorf("resolve source media: yt-dlp did not report a downloaded source path")
	}
	displayName := filepath.Base(finalPath)
	if status != nil {
		fmt.Fprintf(status, "source media ready: %s\n", displayName)
	}

	cleaned = true
	return resolvedSourceMedia{
		Original:    source,
		Path:        finalPath,
		DisplayName: displayName,
		Temporary:   true,
		Cleanup:     cleanup,
	}, nil
}

func preflightPlatformSource(ctx context.Context, ytdlpPath string, source string) error {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-progress",
		"--dump-single-json",
		"--skip-download",
		source,
	}
	output, stderr, err := runYTDLP(ctx, ytdlpPath, args)
	if err != nil {
		return formatYTDLPError("resolve source media", err, stderr)
	}
	var metadata sourceMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return fmt.Errorf("resolve source media: parse yt-dlp metadata: %w", err)
	}
	if metadata.Type == "playlist" || len(metadata.Entries) > 0 {
		return fmt.Errorf("resolve source media: playlists are not supported")
	}
	if metadata.IsLive || strings.Contains(strings.ToLower(metadata.LiveStatus), "live") {
		return fmt.Errorf("resolve source media: live streams are not supported")
	}
	return nil
}

func downloadPlatformSource(ctx context.Context, ytdlpPath string, tempDir string, source string) (string, error) {
	args := platformDownloadArgs(tempDir, source)
	output, stderr, err := runYTDLP(ctx, ytdlpPath, args)
	if err != nil {
		return "", formatYTDLPError("download source media", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

func platformDownloadArgs(tempDir string, source string) []string {
	return []string{
		"--ignore-config",
		"--no-playlist",
		"--match-filters", "!is_live",
		"--no-progress",
		"--paths", "home:" + tempDir,
		"--paths", "temp:" + tempDir,
		"--restrict-filenames",
		"--trim-filenames", "120",
		"--output", sourceFilenameTemplate,
		"--print", "after_move:filepath",
		"--merge-output-format", platformMergeOutput,
		"-f", platformDownloadFormat,
		source,
	}
}

func runYTDLP(ctx context.Context, ytdlpPath string, args []string) ([]byte, string, error) {
	cmd := exec.CommandContext(ctx, ytdlpPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	return output, strings.TrimSpace(stderr.String()), err
}

func formatYTDLPError(prefix string, err error, stderr string) error {
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return fmt.Errorf("%s: yt-dlp is required for platform URLs; install yt-dlp and try again", prefix)
	}
	if stderr != "" {
		return fmt.Errorf("%s: yt-dlp failed: %s", prefix, stderr)
	}
	return fmt.Errorf("%s: yt-dlp failed: %w", prefix, err)
}
```

- [ ] **Step 4: Run source resolver tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestResolveSourceMedia'
```

Expected: source resolver tests pass.

- [ ] **Step 5: Run all CLI tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli
```

Expected: CLI tests pass.

- [ ] **Step 6: Commit source resolver**

Run:

```bash
git add packages/core/internal/cli/source.go packages/core/internal/cli/source_test.go
git commit --no-gpg-sign -m "feat: resolve platform media sources"
```

---

### Task 3: Probe Runner with Source Resolution

**Files:**
- Create: `packages/core/internal/cli/probe.go`
- Create: `packages/core/internal/cli/probe_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Add failing probe runner tests**

Create `packages/core/internal/cli/probe_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/media"
)

func TestPrintProbeInfoForLocalSource(t *testing.T) {
	var out bytes.Buffer
	printProbeInfo(&out, probeOutput{
		OriginalSource: "clip.mp4",
		Width:          1920,
		Height:         1080,
		FPS:            60,
		FrameCount:     120,
		DurationSeconds: 2,
		HasAudio:       true,
		RenderCols:     120,
		RenderRows:     33,
	})
	got := out.String()
	for _, want := range []string{
		"input: clip.mp4\n",
		"video: 1920x1080\n",
		"fps: 60.000\n",
		"frames: 120\n",
		"duration: 2.000s\n",
		"audio: yes\n",
		"render-grid: 120x33 (sample terminal 120x40)\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("probe output missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "resolved-source:") {
		t.Fatalf("local probe printed resolved-source: %q", got)
	}
}

func TestPrintProbeInfoForResolvedPlatformSource(t *testing.T) {
	var out bytes.Buffer
	printProbeInfo(&out, probeOutput{
		OriginalSource:       "https://example.com/watch?v=demo",
		ResolvedDisplayName:  "Demo_Title [abc123].mp4",
		Width:                1280,
		Height:               720,
		FPS:                  30,
		FrameCount:           90,
		DurationSeconds:      3,
		HasAudio:             false,
		RenderCols:           120,
		RenderRows:           33,
	})
	got := out.String()
	for _, want := range []string{
		"input: https://example.com/watch?v=demo\n",
		"resolved-source: Demo_Title [abc123].mp4\n",
		"audio: no\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("probe output missing %q in:\n%s", want, got)
		}
	}
}

func TestRunProbeResolvesPlatformSource(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runProbeWithOptions(context.Background(), "https://example.com/watch?v=demo", &stdout, &stderr, probeRunnerOptions{
		YTDLPPath: fake.Path,
		Probe: func(ctx context.Context, path string) (media.Info, error) {
			if !strings.HasSuffix(path, "Demo_Title [abc123].mp4") {
				t.Fatalf("probe path = %q, want resolved downloaded file", path)
			}
			return media.Info{
				Width:           1280,
				Height:          720,
				FPS:             30,
				FrameCount:      90,
				DurationSeconds: 3,
				HasAudio:        true,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("runProbeWithOptions returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "source media ready: Demo_Title [abc123].mp4") {
		t.Fatalf("stderr missing source ready status: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "input: https://example.com/watch?v=demo\n") {
		t.Fatalf("stdout missing original input: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "resolved-source: Demo_Title [abc123].mp4\n") {
		t.Fatalf("stdout missing resolved source: %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run probe tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestPrintProbe|TestRunProbe'
```

Expected: fails because `RunProbe`, `printProbeInfo`, and supporting types do not exist.

- [ ] **Step 3: Implement `cli.RunProbe`**

Create `packages/core/internal/cli/probe.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/jass/mojify/packages/core/internal/media"
	"github.com/jass/mojify/packages/core/internal/render"
)

type probeOutput struct {
	OriginalSource      string
	ResolvedDisplayName string
	Width               int
	Height              int
	FPS                 float64
	FrameCount          int
	DurationSeconds     float64
	HasAudio            bool
	RenderCols          int
	RenderRows          int
}

type probeRunnerOptions struct {
	YTDLPPath string
	Probe     func(context.Context, string) (media.Info, error)
}

func RunProbe(ctx context.Context, source string, stdout io.Writer, stderr io.Writer) error {
	return runProbeWithOptions(ctx, source, stdout, stderr, probeRunnerOptions{})
}

func runProbeWithOptions(ctx context.Context, source string, stdout io.Writer, stderr io.Writer, options probeRunnerOptions) error {
	resolved, err := resolveSourceMediaWithOptions(ctx, source, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: options.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()

	probe := options.Probe
	if probe == nil {
		probe = media.ProbeContext
	}
	info, err := probe(ctx, resolved.Path)
	if err != nil {
		return fmt.Errorf("probe input: %w", err)
	}
	grid := render.FitGrid(
		render.InputSize{Width: info.Width, Height: info.Height},
		render.TerminalSize{Cols: 120, Rows: 40},
	)
	output := probeOutput{
		OriginalSource:      resolved.Original,
		ResolvedDisplayName: resolvedDisplayName(resolved),
		Width:               info.Width,
		Height:              info.Height,
		FPS:                 info.FPS,
		FrameCount:          info.FrameCount,
		DurationSeconds:     info.DurationSeconds,
		HasAudio:            info.HasAudio,
		RenderCols:          grid.Cols,
		RenderRows:          grid.Rows,
	}
	printProbeInfo(stdout, output)
	return nil
}

func resolvedDisplayName(source resolvedSourceMedia) string {
	if !source.Temporary {
		return ""
	}
	return source.DisplayName
}

func printProbeInfo(w io.Writer, output probeOutput) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "input: %s\n", output.OriginalSource)
	if output.ResolvedDisplayName != "" {
		fmt.Fprintf(w, "resolved-source: %s\n", output.ResolvedDisplayName)
	}
	fmt.Fprintf(w, "video: %dx%d\n", output.Width, output.Height)
	fmt.Fprintf(w, "fps: %.3f\n", output.FPS)
	fmt.Fprintf(w, "frames: %d\n", output.FrameCount)
	fmt.Fprintf(w, "duration: %.3fs\n", output.DurationSeconds)
	if output.HasAudio {
		fmt.Fprintln(w, "audio: yes")
	} else {
		fmt.Fprintln(w, "audio: no")
	}
	fmt.Fprintf(w, "render-grid: %dx%d (sample terminal 120x40)\n", output.RenderCols, output.RenderRows)
}
```

- [ ] **Step 4: Update `main.go` probe dispatch**

In `packages/core/cmd/mojify/main.go`, remove the `media` and `render` imports. Replace the probe case with signal-aware `cli.RunProbe`:

```go
	case cli.ProbeCommand:
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := cli.RunProbe(ctx, cmd.InputPath, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "probe failed: %v\n", err)
			os.Exit(1)
		}
```

- [ ] **Step 5: Run probe tests and command package tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli ./packages/core/cmd/mojify
```

Expected: tests pass.

- [ ] **Step 6: Commit probe runner**

Run:

```bash
git add packages/core/internal/cli/probe.go packages/core/internal/cli/probe_test.go packages/core/cmd/mojify/main.go
git commit --no-gpg-sign -m "feat: run probe through source resolver"
```

---

### Task 4: Wire Source Resolution into Play and Export

**Files:**
- Modify: `packages/core/internal/cli/play.go`
- Modify: `packages/core/internal/cli/export.go`
- Modify: `packages/core/internal/cli/play_test.go`
- Create or modify: `packages/core/internal/cli/export_test.go`

- [ ] **Step 1: Add source-resolution status tests for play/export helpers**

Append to `packages/core/internal/cli/export_test.go`. Create the file if it does not exist:

```go
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunExportRejectsMissingYTDLPForPlatformURL(t *testing.T) {
	var stderr bytes.Buffer
	err := runExportWithOptions(context.Background(), "https://example.com/watch?v=demo", "out.mp4", &stderr, ExportOptions{}, exportRunnerOptions{
		YTDLPPath: "definitely-missing-yt-dlp",
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required for platform URLs") {
		t.Fatalf("error = %v, want missing yt-dlp message", err)
	}
}
```

Append to `packages/core/internal/cli/play_test.go`:

```go
func TestRunPlayRejectsMissingYTDLPForPlatformURL(t *testing.T) {
	err := runPlayWithOptions(context.Background(), "https://example.com/watch?v=demo", os.Stdin, io.Discard, io.Discard, PlayOptions{}, playRunnerOptions{
		YTDLPPath: "definitely-missing-yt-dlp",
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required for platform URLs") {
		t.Fatalf("error = %v, want missing yt-dlp message", err)
	}
}
```

If `play_test.go` does not already import `context`, `io`, `os`, and `strings`, add them to its import block.

- [ ] **Step 2: Run new play/export tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestRunExportRejectsMissingYTDLP|TestRunPlayRejectsMissingYTDLP'
```

Expected: fails because `runExportWithOptions`, `runPlayWithOptions`, and runner option types do not exist.

- [ ] **Step 3: Add play runner options and source resolution**

In `packages/core/internal/cli/play.go`, add:

```go
type playRunnerOptions struct {
	YTDLPPath string
}
```

Change `RunPlay` to delegate:

```go
func RunPlay(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer, stderr io.Writer, options PlayOptions) error {
	return runPlayWithOptions(ctx, inputPath, stdin, stdout, stderr, options, playRunnerOptions{})
}
```

Add a new `runPlayWithOptions` function containing the current `RunPlay` body. At the start of the function, immediately after creating the cancellable context, resolve the source:

```go
	resolved, err := resolveSourceMediaWithOptions(ctx, inputPath, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: runnerOptions.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()
	inputPath = resolved.Path
```

The function signature should be:

```go
func runPlayWithOptions(ctx context.Context, inputPath string, stdin *os.File, stdout io.Writer, stderr io.Writer, options PlayOptions, runnerOptions playRunnerOptions) error
```

Keep all existing playback logic after source resolution unchanged.

- [ ] **Step 4: Add export runner options and source resolution**

In `packages/core/internal/cli/export.go`, add:

```go
type exportRunnerOptions struct {
	YTDLPPath string
}
```

Change `RunExport` to delegate:

```go
func RunExport(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions) error {
	return runExportWithOptions(ctx, inputPath, outputPath, stderr, options, exportRunnerOptions{})
}
```

Add:

```go
func runExportWithOptions(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options ExportOptions, runnerOptions exportRunnerOptions) error {
	resolved, err := resolveSourceMediaWithOptions(ctx, inputPath, sourceResolverOptions{
		Stderr:    stderr,
		YTDLPPath: runnerOptions.YTDLPPath,
	})
	if err != nil {
		return err
	}
	defer resolved.Cleanup()

	return exporter.ExportMP4(ctx, resolved.Path, outputPath, stderr, exporter.Options{
		Width:               options.Width,
		FPS:                 options.FPS,
		Bitrate:             options.Bitrate,
		Overwrite:           options.Overwrite,
		ProgressInteractive: isTerminalWriter(stderr),
		Stats:               options.Stats,
		Workers:             options.Workers,
	})
}
```

- [ ] **Step 5: Run play/export focused tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli -run 'TestRunExportRejectsMissingYTDLP|TestRunPlayRejectsMissingYTDLP|TestBridgeTerminalControls|TestPlaybackResult'
```

Expected: tests pass.

- [ ] **Step 6: Run all CLI tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./packages/core/internal/cli
```

Expected: CLI tests pass.

- [ ] **Step 7: Commit play/export source wiring**

Run:

```bash
git add packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go packages/core/internal/cli/export.go packages/core/internal/cli/export_test.go
git commit --no-gpg-sign -m "feat: resolve platform sources for play and export"
```

---

### Task 5: README and QA Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/qa/playback-quality.md`
- Modify: `docs/qa/export.md`
- Add or modify: `docs/qa/platform-media-input.md`
- Add or modify: `CONTEXT.md`
- Add or modify: `docs/adr/0026-resolve-platform-media-input-in-cli.md`

- [ ] **Step 1: Update README language**

In `README.md`, update the opening sentence to:

```md
Mojify is a terminal-first video player that transforms local video files and yt-dlp-compatible platform URLs into colored, edge-aware character frames.
```

Update requirements to:

```md
- FFmpeg and ffprobe on `PATH`
- yt-dlp on `PATH` for platform URL inputs
- ffplay on `PATH` for live playback audio
```

Update run examples to include:

```md
./bin/mojify probe ./demo.mp4
./bin/mojify play ./demo.mp4
./bin/mojify probe "https://www.youtube.com/watch?v=<id>"
./bin/mojify play "https://www.youtube.com/watch?v=<id>"
./bin/mojify export --overwrite --width 320 ./demo.mp4 dist/demo-export.mp4
./bin/mojify export --overwrite --width 320 "https://www.youtube.com/watch?v=<id>" dist/demo-url-export.mp4
```

Move URL input from `Deferred` to `Included now`:

```md
- Local video files
- yt-dlp-compatible HTTP(S) platform URLs
- Visual terminal playback
- Live terminal audio playback
- MP4 export with source audio content when available
- Truecolor ANSI output
- Edge-aware character rendering
- `play`, `probe`, and `export` commands
```

Keep distribution, GIF/PNG, plugins, custom recipes, playlist workflow, and live streams deferred.

- [ ] **Step 2: Add cross-links from playback and export QA**

Append this short paragraph near the top of `docs/qa/playback-quality.md`:

```md
Platform URL playback is covered by the cross-command checklist in `docs/qa/platform-media-input.md`.
```

Append this short paragraph near the top of `docs/qa/export.md`:

```md
Platform URL export is covered by the cross-command checklist in `docs/qa/platform-media-input.md`.
```

- [ ] **Step 3: Verify platform QA doc content**

Ensure `docs/qa/platform-media-input.md` contains all of these phrases:

```text
fake `yt-dlp`
HTTP(S)
--ignore-config
--no-playlist
!is_live
--print after_move:filepath
probe "$URL"
play --stats "$URL"
export --overwrite --width 320 "$URL"
No downloaded source media is left behind by default.
```

- [ ] **Step 4: Run docs grep checks**

Run:

```bash
rg -n "Platform media input|Source media|Source|yt-dlp-compatible|resolved-source|--ignore-config|after_move:filepath|platform URL inputs" CONTEXT.md README.md docs/adr/0026-resolve-platform-media-input-in-cli.md docs/qa/platform-media-input.md docs/qa/playback-quality.md docs/qa/export.md
```

Expected: output includes the updated glossary, ADR, QA, and README terms.

- [ ] **Step 5: Commit docs**

Run:

```bash
git add CONTEXT.md README.md docs/adr/0026-resolve-platform-media-input-in-cli.md docs/qa/platform-media-input.md docs/qa/playback-quality.md docs/qa/export.md
git commit --no-gpg-sign -m "docs: describe platform media input"
```

---

### Task 6: Final Verification and Manual URL Smoke

**Files:**
- Read: `git status --short --branch`
- Read: `docs/qa/platform-media-input.md`

- [ ] **Step 1: Run formatting check**

Run:

```bash
bun run fmt:check
```

Expected: exit code 0.

- [ ] **Step 2: Run module tidy diff**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go mod tidy -diff
```

Expected: no output and exit code 0.

- [ ] **Step 3: Run full tests**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...
bun run test
```

Expected: both commands exit 0.

- [ ] **Step 4: Run typecheck and build**

Run:

```bash
bun run typecheck
bun run build
```

Expected: both commands exit 0. If the sandbox blocks Go build cache writes, rerun the same command with approval rather than changing the project.

- [ ] **Step 5: Run existing QA clips and export QA**

Run:

```bash
bun run qa:clips
bun run qa:export
```

Expected: generated clip QA and export QA pass.

- [ ] **Step 6: Run local source smoke checks**

Run:

```bash
./bin/mojify probe dist/qa/low-motion-bars.mp4
./bin/mojify play --no-audio --stats dist/qa/low-motion-bars.mp4 >/private/tmp/mojify-platform-local-play.out 2>/private/tmp/mojify-platform-local-play.err
```

Expected:

- Probe output includes `input: dist/qa/low-motion-bars.mp4`.
- Probe output does not include `resolved-source:`.
- Play stats include `audio: disabled`, `audio stream: no`, and `audio warnings: 0`.
- Stderr for the local play smoke does not include `resolving source media`.

- [ ] **Step 7: Run optional real URL smoke when network and a public URL are available**

Set a public finite URL that yt-dlp can resolve without cookies:

```bash
URL="<yt-dlp-compatible-http-url>"
mkdir -p dist/qa/export
./bin/mojify probe "$URL"
./bin/mojify export --overwrite --width 320 "$URL" dist/qa/export/platform-url-export.mp4
```

Expected:

- URL resolution prints `resolving source media`, `downloading source media...`, and `source media ready`.
- Probe stdout includes `input: <original-url>` and `resolved-source: <downloaded-basename>`.
- Export writes `dist/qa/export/platform-url-export.mp4`.
- `ffprobe` finds a video stream in the exported MP4.

If running interactive playback is appropriate, run:

```bash
./bin/mojify play --stats "$URL"
```

Expected:

- Playback starts after source resolution completes.
- Audio, pause/resume, `q`, and Ctrl-C behavior match local resolved source media.

- [ ] **Step 8: Run final diff checks**

Run:

```bash
git diff --check
git status --short --branch
git log --oneline --decorate -5
```

Expected:

- `git diff --check` exits 0.
- Working tree is clean after commits.
- Recent commits include platform media input work.

---

## Self-Review

- Spec coverage: This plan covers HTTP(S) yt-dlp-compatible inputs for probe/play/export, download-first source resolution, temp cleanup, one finite video, playlist/live rejection, yt-dlp URL-only dependency, `--ignore-config`, MP4 preference, phase-only status, local path silence, parser/help updates, `RunProbe`, README/QA/docs, and final verification.
- Placeholder scan: No unresolved placeholder instructions remain. The optional real URL smoke uses an explicit operator-provided `URL` variable because automated tests must not depend on the network.
- Type consistency: `resolvedSourceMedia`, `sourceResolverOptions`, `probeOutput`, `probeRunnerOptions`, `playRunnerOptions`, and `exportRunnerOptions` are introduced before later use.
- Scope check: Persistent cache, streaming into FFmpeg, playlists, live streams, auth/cookies, custom yt-dlp args, and distribution packaging are kept out of scope.
