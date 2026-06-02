# Playback Audio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add default-on live terminal audio to `mojify play`, with `--no-audio` as the opt-out, best-effort fallback, pause/resume control, cleanup, probe visibility, stats, and QA.

**Architecture:** Keep visual playback scheduling in `player` and orchestrate audio in `cli.RunPlay`. Extend `media.ProbeContext` to detect source audio, add a small `ffplay` process boundary for live audio, and drive audio pause/resume from the same terminal controls that drive visual playback. Audio is best-effort: missing source audio, missing `ffplay`, or audio-device startup failure must not block visual terminal playback.

**Tech Stack:** Go, FFmpeg CLI, ffprobe, ffplay, existing Mojify CLI/player/media packages, Bun/Turbo verification scripts.

---

## Implementation Result

Implemented on `feat/playback-audio` on 2026-06-02.

Automated verification run:

- `GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...`
- `bun run fmt:check`
- `bun run test`
- `bun run typecheck`
- `bun run build`
- `GOCACHE=/private/tmp/mojify-gocache go test -race -count=1 ./packages/core/internal/cli`
- `GOCACHE=/private/tmp/mojify-gocache go mod tidy -diff`
- `bun run qa:clips`
- `git diff --check`

Local smoke verification run:

- `./bin/mojify probe dist/qa/low-motion-bars.mp4` reported `audio: no`.
- `./bin/mojify probe dist/iris.mp4` reported `audio: yes`.
- `./bin/mojify play --no-audio --stats dist/qa/low-motion-bars.mp4` completed the 120-frame silent QA clip with `audio: disabled`, `audio stream: no`, `audio started: no`, and `audio warnings: 0`.

Manual TTY QA still required before treating playback audio as user-validated:

- Audible default playback on a real audio sample.
- Space pause/resume for both visuals and ffplay audio.
- `q` and Ctrl-C cleanup of the ffplay process.
- `--no-audio` silence on a real audio sample.

---

## Decisions Already Made

- The stage term is **Playback audio**.
- `mojify play` attempts live terminal audio by default when the source has an audio stream.
- `mojify play --no-audio <video>` disables live terminal audio.
- `--no-audio` is valid only on `play`, not `export`.
- Audio is best-effort. Visual playback continues when source audio is absent, `ffplay` is missing, or audio cannot start.
- Mojify probes for audio first and starts `ffplay` only when the source has an audio stream.
- `mojify probe` prints `audio: yes` or `audio: no`.
- Pause/resume controls both visual playback and live terminal audio.
- `q` and Ctrl-C must stop audio reliably.
- Rough wall-clock sync is enough for this stage; no drift correction, seeking, or resync loop.
- Start audio immediately before entering visual playback after decoder/render pipeline, presenter, and controls are initialized.
- Prefer ffplay stdin `p` for pause/resume. If manual QA proves it unreliable in `-nodisp`, use process suspend/resume as the fallback.
- Audio lifecycle is orchestrated in `cli.RunPlay`, not in `player`.
- Startup audio warnings may print before visual playback begins.
- Runtime audio failures print once after playback exits, even without `--stats`.
- `play --stats` includes sparse audio facts.
- Audio may start when stdin is not a TTY.
- Mute, volume controls, audio stream selection, URL input, distribution work, and export audio changes are out of scope.

## File Structure

- Modify: `CONTEXT.md`
  - Already updated during grill: playback includes live audio, controls include audio, and `Playback audio` is defined.
- Create: `docs/adr/0024-use-ffplay-for-live-terminal-audio.md`
  - Already created during grill.
- Create: `docs/adr/0025-enable-live-playback-audio-by-default.md`
  - Already created during grill.
- Modify: `packages/core/internal/cli/cli.go`
  - Parse `play --no-audio`, reject duplicate play flags, update help text and requirements.
- Modify: `packages/core/internal/cli/cli_test.go`
  - Lock `--no-audio` parsing, duplicate rejection, export rejection, and help text.
- Modify: `packages/core/cmd/mojify/main.go`
  - Pass play audio options and print probe audio metadata.
- Modify: `packages/core/internal/media/probe.go`
  - Add `Info.HasAudio` and parse audio streams.
- Modify: `packages/core/internal/media/probe_test.go`
  - Lock audio stream detection and missing-audio behavior.
- Create: `packages/core/internal/media/audio.go`
  - Build ffplay args and start an ffplay live-audio process.
- Create: `packages/core/internal/media/audio_test.go`
  - Lock ffplay args and input validation.
- Create: `packages/core/internal/cli/play_audio.go`
  - Own playback audio status, backend abstraction, ffplay adapter, pause/resume, stop, runtime warnings, and stats formatting.
- Create: `packages/core/internal/cli/play_audio_test.go`
  - Lock best-effort start, disabled behavior, no-source-audio behavior, pause toggling, stop cleanup, runtime warning drain, and stats text.
- Modify: `packages/core/internal/cli/play.go`
  - Wire audio into `RunPlay`, fan out terminal controls, print warnings and audio stats.
- Modify: `packages/core/internal/cli/play_test.go`
  - Lock fan-out and audio stats helpers.
- Modify: `docs/qa/playback-quality.md`
  - Add playback audio QA section.
- Modify: `docs/superpowers/plans/2026-06-02-mojify-playback-audio.md`
  - Track implementation status.

---

### Task 1: CLI Audio Option and Help Text

**Files:**
- Modify: `packages/core/internal/cli/cli.go`
- Modify: `packages/core/internal/cli/cli_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Add failing CLI tests for `play --no-audio`**

Append to `packages/core/internal/cli/cli_test.go`:

```go
func TestParsePlayNoAudioBeforeInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "--no-audio", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != PlayCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, PlayCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if !cmd.NoAudio {
		t.Fatal("NoAudio = false, want true")
	}
}

func TestParsePlayNoAudioAfterInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "clip.mp4", "--no-audio"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !cmd.NoAudio {
		t.Fatal("NoAudio = false, want true")
	}
}

func TestParsePlayRejectsDuplicateNoAudio(t *testing.T) {
	_, err := Parse([]string{"play", "--no-audio", "clip.mp4", "--no-audio"})
	if err == nil {
		t.Fatal("Parse returned nil error for duplicate --no-audio")
	}
}

func TestParseRejectsNoAudioForProbe(t *testing.T) {
	_, err := Parse([]string{"probe", "--no-audio", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for probe --no-audio")
	}
}

func TestParseExportRejectsNoAudio(t *testing.T) {
	_, err := Parse([]string{"export", "--no-audio", "clip.mov", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for export --no-audio")
	}
}
```

In `TestHelpTextMentionsCommands`, replace the current play usage expectation:

```go
		"mojify play [--stats] <video>",
```

with:

```go
		"mojify play [--stats] [--no-audio] <video>",
```

Add these expectations to the same list:

```go
		"--no-audio",
		"ffplay is required for live playback audio unless --no-audio is used",
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail because `Command.NoAudio`, `--no-audio` parsing, and help text are missing.

- [ ] **Step 3: Add the play audio parse field**

Modify `packages/core/internal/cli/cli.go` so `Command` includes:

```go
type Command struct {
	Kind       CommandKind
	InputPath  string
	OutputPath string
	Stats      bool
	NoAudio    bool
	Export     ExportOptions
}
```

- [ ] **Step 4: Parse `play --no-audio` only for play**

Replace `parseInputCommand` in `packages/core/internal/cli/cli.go` with:

```go
func parseInputCommand(kind CommandKind, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%s requires a video input", args[0])
	}

	var inputPath string
	stats := false
	noAudio := false
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
		case "--no-audio":
			if kind != PlayCommand {
				return Command{}, fmt.Errorf("%s does not accept --no-audio", args[0])
			}
			if noAudio {
				return Command{}, fmt.Errorf("%s accepts --no-audio only once", args[0])
			}
			noAudio = true
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
	return Command{Kind: kind, InputPath: inputPath, Stats: stats, NoAudio: noAudio}, nil
}
```

- [ ] **Step 5: Update help text**

In `HelpText()` in `packages/core/internal/cli/cli.go`, replace:

```text
  mojify play [--stats] <video>                         Play a local video file in the terminal
```

with:

```text
  mojify play [--stats] [--no-audio] <video>            Play a local video file in the terminal
```

Add this line near the other play/export options:

```text
  --no-audio         Disable live playback audio for play
```

Replace the requirements section with:

```text
Requirements:
  FFmpeg and ffprobe must be available on PATH.
  ffplay is required for live playback audio unless --no-audio is used.
```

- [ ] **Step 6: Pass the CLI option into `RunPlay`**

In `packages/core/cmd/mojify/main.go`, replace:

```go
cli.PlayOptions{Stats: cmd.Stats},
```

with:

```go
cli.PlayOptions{Stats: cmd.Stats, NoAudio: cmd.NoAudio},
```

- [ ] **Step 7: Run CLI tests**

Run:

```bash
gofmt -w packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go packages/core/cmd/mojify/main.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: pass.

- [ ] **Step 8: Commit CLI option**

Run:

```bash
git add packages/core/internal/cli/cli.go packages/core/internal/cli/cli_test.go packages/core/cmd/mojify/main.go
git commit --no-gpg-sign -m "feat: add playback audio opt-out"
```

---

### Task 2: Probe Audio Metadata

**Files:**
- Modify: `packages/core/internal/media/probe.go`
- Modify: `packages/core/internal/media/probe_test.go`
- Modify: `packages/core/cmd/mojify/main.go`

- [ ] **Step 1: Add failing probe tests for audio presence**

Append to `packages/core/internal/media/probe_test.go`:

```go
func TestParseProbeJSONDetectsAudioStream(t *testing.T) {
	const input = `{
	  "streams": [
	    {
	      "codec_type": "audio",
	      "codec_name": "aac"
	    },
	    {
	      "codec_type": "video",
	      "width": 1280,
	      "height": 720,
	      "avg_frame_rate": "24/1",
	      "nb_frames": "48"
	    }
	  ],
	  "format": { "duration": "2.000000" }
	}`

	info, err := ParseProbeJSON([]byte(input))
	if err != nil {
		t.Fatalf("ParseProbeJSON returned error: %v", err)
	}
	if !info.HasAudio {
		t.Fatal("HasAudio = false, want true")
	}
}

func TestParseProbeJSONReportsNoAudioStream(t *testing.T) {
	const input = `{
	  "streams": [
	    {
	      "codec_type": "video",
	      "width": 1280,
	      "height": 720,
	      "avg_frame_rate": "24/1",
	      "nb_frames": "48"
	    }
	  ],
	  "format": { "duration": "2.000000" }
	}`

	info, err := ParseProbeJSON([]byte(input))
	if err != nil {
		t.Fatalf("ParseProbeJSON returned error: %v", err)
	}
	if info.HasAudio {
		t.Fatal("HasAudio = true, want false")
	}
}
```

- [ ] **Step 2: Run media tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: fail because `Info.HasAudio` is missing.

- [ ] **Step 3: Add `HasAudio` to media info**

Modify `packages/core/internal/media/probe.go`:

```go
type Info struct {
	Width           int
	Height          int
	FPS             float64
	FrameCount      int
	DurationSeconds float64
	HasAudio        bool
}
```

- [ ] **Step 4: Parse audio stream presence**

Replace `ParseProbeJSON` in `packages/core/internal/media/probe.go` with:

```go
func ParseProbeJSON(data []byte) (Info, error) {
	var probe probeJSON
	if err := json.Unmarshal(data, &probe); err != nil {
		return Info{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	hasAudio := false
	for _, stream := range probe.Streams {
		if stream.CodecType == "audio" {
			hasAudio = true
			break
		}
	}

	for _, stream := range probe.Streams {
		if stream.CodecType != "video" {
			continue
		}
		if stream.Width <= 0 || stream.Height <= 0 {
			return Info{}, fmt.Errorf("invalid video dimensions %dx%d", stream.Width, stream.Height)
		}

		fps, err := parseRate(stream.AvgFrameRate)
		if err != nil {
			return Info{}, fmt.Errorf("parse avg_frame_rate: %w", err)
		}

		frameCount, err := parseOptionalInt(stream.NBFrames)
		if err != nil {
			return Info{}, fmt.Errorf("parse nb_frames: %w", err)
		}

		duration, err := parseOptionalFloat(probe.Format.Duration)
		if err != nil {
			return Info{}, fmt.Errorf("parse duration: %w", err)
		}

		return Info{
			Width:           stream.Width,
			Height:          stream.Height,
			FPS:             fps,
			FrameCount:      frameCount,
			DurationSeconds: duration,
			HasAudio:        hasAudio,
		}, nil
	}

	return Info{}, fmt.Errorf("missing video stream")
}
```

- [ ] **Step 5: Print audio metadata in probe command**

In `packages/core/cmd/mojify/main.go`, after:

```go
fmt.Printf("duration: %.3fs\n", info.DurationSeconds)
```

add:

```go
if info.HasAudio {
	fmt.Println("audio: yes")
} else {
	fmt.Println("audio: no")
}
```

- [ ] **Step 6: Run focused tests**

Run:

```bash
gofmt -w packages/core/internal/media/probe.go packages/core/internal/media/probe_test.go packages/core/cmd/mojify/main.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: pass.

- [ ] **Step 7: Commit probe metadata**

Run:

```bash
git add packages/core/internal/media/probe.go packages/core/internal/media/probe_test.go packages/core/cmd/mojify/main.go
git commit --no-gpg-sign -m "feat: report source audio in probe"
```

---

### Task 3: ffplay Audio Process Boundary

**Files:**
- Create: `packages/core/internal/media/audio.go`
- Create: `packages/core/internal/media/audio_test.go`

- [ ] **Step 1: Add failing ffplay argument tests**

Create `packages/core/internal/media/audio_test.go`:

```go
package media

import (
	"reflect"
	"testing"
)

func TestFFplayAudioArgs(t *testing.T) {
	args, err := FFplayAudioArgs("clip.mp4")
	if err != nil {
		t.Fatalf("FFplayAudioArgs returned error: %v", err)
	}
	want := []string{
		"-nodisp",
		"-autoexit",
		"-loglevel", "error",
		"-vn",
		"clip.mp4",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestFFplayAudioArgsRejectsMissingInput(t *testing.T) {
	_, err := FFplayAudioArgs(" ")
	if err == nil {
		t.Fatal("FFplayAudioArgs returned nil error for missing input")
	}
}
```

- [ ] **Step 2: Run media tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: fail because `FFplayAudioArgs` is missing.

- [ ] **Step 3: Add ffplay args and process starter**

Create `packages/core/internal/media/audio.go`:

```go
package media

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func FFplayAudioArgs(inputPath string) ([]string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return nil, fmt.Errorf("input path is required")
	}
	return []string{
		"-nodisp",
		"-autoexit",
		"-loglevel", "error",
		"-vn",
		inputPath,
	}, nil
}

func StartFFplayAudioContext(ctx context.Context, inputPath string, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	args, err := FFplayAudioArgs(inputPath)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.CommandContext(ctx, "ffplay", args...)
	if stderr != nil {
		cmd.Stderr = stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdin, nil
}
```

- [ ] **Step 4: Run focused tests**

Run:

```bash
gofmt -w packages/core/internal/media/audio.go packages/core/internal/media/audio_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/media
```

Expected: pass.

- [ ] **Step 5: Commit audio process boundary**

Run:

```bash
git add packages/core/internal/media/audio.go packages/core/internal/media/audio_test.go
git commit --no-gpg-sign -m "feat: add ffplay audio process boundary"
```

---

### Task 4: Playback Audio Controller

**Files:**
- Create: `packages/core/internal/cli/play_audio.go`
- Create: `packages/core/internal/cli/play_audio_test.go`

- [ ] **Step 1: Add failing controller tests**

Create `packages/core/internal/cli/play_audio_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPlaybackAudioDisabledDoesNotStart(t *testing.T) {
	backend := &fakeAudioBackend{}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   false,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	if audio.Status().Enabled {
		t.Fatal("Enabled = true, want false")
	}
	if backend.starts != 0 {
		t.Fatalf("backend starts = %d, want 0", backend.starts)
	}
}

func TestPlaybackAudioSkipsMissingAudioStream(t *testing.T) {
	backend := &fakeAudioBackend{}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: false,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	status := audio.Status()
	if !status.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if status.HasStream {
		t.Fatal("HasStream = true, want false")
	}
	if status.Started {
		t.Fatal("Started = true, want false")
	}
	if backend.starts != 0 {
		t.Fatalf("backend starts = %d, want 0", backend.starts)
	}
}

func TestPlaybackAudioRecordsStartupWarning(t *testing.T) {
	backend := &fakeAudioBackend{startErr: errors.New("ffplay missing")}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	status := audio.Status()
	if status.Started {
		t.Fatal("Started = true, want false")
	}
	if status.WarningCount != 1 {
		t.Fatalf("WarningCount = %d, want 1", status.WarningCount)
	}
	warnings := audio.DrainWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "ffplay missing") {
		t.Fatalf("warnings = %#v, want ffplay missing warning", warnings)
	}
}

func TestPlaybackAudioTogglePause(t *testing.T) {
	process := &fakeAudioProcess{}
	backend := &fakeAudioBackend{process: process}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	audio.TogglePause()
	if process.toggles != 1 {
		t.Fatalf("toggles = %d, want 1", process.toggles)
	}
}

func TestPlaybackAudioStop(t *testing.T) {
	process := &fakeAudioProcess{}
	backend := &fakeAudioBackend{process: process}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	audio.Stop()
	if process.stops != 1 {
		t.Fatalf("stops = %d, want 1", process.stops)
	}
}

func TestPlaybackAudioStats(t *testing.T) {
	status := playbackAudioStatus{
		Enabled:      true,
		HasStream:    true,
		Started:      true,
		WarningCount: 2,
	}
	var out bytes.Buffer
	printAudioStats(&out, status)
	got := out.String()
	for _, want := range []string{
		"audio: enabled\n",
		"audio stream: yes\n",
		"audio started: yes\n",
		"audio warnings: 2\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("audio stats missing %q in:\n%s", want, got)
		}
	}
}

func TestPlaybackAudioDrainWarningsKeepsCumulativeStatusCount(t *testing.T) {
	backend := &fakeAudioBackend{startErr: errors.New("ffplay missing")}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	if got := len(audio.DrainWarnings()); got != 1 {
		t.Fatalf("drained warnings = %d, want 1", got)
	}
	if got := audio.Status().WarningCount; got != 1 {
		t.Fatalf("WarningCount = %d, want cumulative count 1", got)
	}
}

func TestPlaybackAudioStopSuppressesExpectedProcessExitWarning(t *testing.T) {
	process := &fakeAudioProcess{done: make(chan error, 1)}
	backend := &fakeAudioBackend{process: process}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	audio.Stop()
	process.done <- errors.New("signal: killed")
	time.Sleep(25 * time.Millisecond)
	if got := audio.Status().WarningCount; got != 0 {
		t.Fatalf("WarningCount = %d, want 0", got)
	}
}

type fakeAudioBackend struct {
	starts   int
	startErr error
	process  *fakeAudioProcess
}

func (b *fakeAudioBackend) Start(ctx context.Context, inputPath string) (playbackAudioProcess, error) {
	b.starts++
	if b.startErr != nil {
		return nil, b.startErr
	}
	if b.process == nil {
		b.process = &fakeAudioProcess{}
	}
	return b.process, nil
}

type fakeAudioProcess struct {
	toggles int
	stops   int
	done    chan error
}

func (p *fakeAudioProcess) TogglePause() error {
	p.toggles++
	return nil
}

func (p *fakeAudioProcess) Stop() error {
	p.stops++
	return nil
}

func (p *fakeAudioProcess) Done() <-chan error {
	if p.done == nil {
		p.done = make(chan error)
	}
	return p.done
}

```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail because playback audio types and helpers are missing.

- [ ] **Step 3: Add playback audio controller**

Create `packages/core/internal/cli/play_audio.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/jass/mojify/packages/core/internal/media"
)

type playbackAudioOptions struct {
	Enabled   bool
	HasStream bool
	Backend   playbackAudioBackend
}

type playbackAudioBackend interface {
	Start(ctx context.Context, inputPath string) (playbackAudioProcess, error)
}

type playbackAudioProcess interface {
	TogglePause() error
	Stop() error
	Done() <-chan error
}

type playbackAudioStatus struct {
	Enabled      bool
	HasStream    bool
	Started      bool
	WarningCount int
}

type playbackAudio struct {
	mu       sync.Mutex
	process  playbackAudioProcess
	status   playbackAudioStatus
	warnings []string
	warningCount int
	backend  playbackAudioBackend
	stopping bool
}

func newPlaybackAudio(options playbackAudioOptions) *playbackAudio {
	backend := options.Backend
	if backend == nil {
		backend = ffplayAudioBackend{}
	}
	return &playbackAudio{
		status: playbackAudioStatus{
			Enabled:   options.Enabled,
			HasStream: options.HasStream,
		},
		backend: backend,
	}
}

func (a *playbackAudio) Start(ctx context.Context, inputPath string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	enabled := a.status.Enabled
	hasStream := a.status.HasStream
	backend := a.backend
	a.mu.Unlock()
	if !enabled || !hasStream {
		return
	}
	process, err := backend.Start(ctx, inputPath)
	if err != nil {
		a.recordWarning(fmt.Sprintf("audio warning: %v", err))
		return
	}

	a.mu.Lock()
	a.process = process
	a.status.Started = true
	a.stopping = false
	a.mu.Unlock()

	go a.watchProcess(process)
}

func startPlaybackAudio(ctx context.Context, inputPath string, options playbackAudioOptions) *playbackAudio {
	audio := newPlaybackAudio(options)
	audio.Start(ctx, inputPath)
	return audio
}

func (a *playbackAudio) TogglePause() {
	if a == nil {
		return
	}
	a.mu.Lock()
	process := a.process
	a.mu.Unlock()
	if process == nil {
		return
	}
	if err := process.TogglePause(); err != nil {
		a.recordWarning(fmt.Sprintf("audio warning: pause failed: %v", err))
	}
}

func (a *playbackAudio) Stop() {
	if a == nil {
		return
	}
	a.mu.Lock()
	process := a.process
	a.process = nil
	a.stopping = true
	a.mu.Unlock()
	if process == nil {
		return
	}
	if err := process.Stop(); err != nil {
		a.recordWarning(fmt.Sprintf("audio warning: stop failed: %v", err))
	}
}

func (a *playbackAudio) Status() playbackAudioStatus {
	if a == nil {
		return playbackAudioStatus{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	status := a.status
	status.WarningCount = a.warningCount
	return status
}

func (a *playbackAudio) DrainWarnings() []string {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	warnings := append([]string(nil), a.warnings...)
	a.warnings = nil
	return warnings
}

func (a *playbackAudio) watchProcess(process playbackAudioProcess) {
	err, ok := <-process.Done()
	if !ok || err == nil {
		return
	}
	a.mu.Lock()
	stopping := a.stopping
	if a.process == process {
		a.process = nil
	}
	a.mu.Unlock()
	if stopping {
		return
	}
	a.recordWarning(fmt.Sprintf("audio warning: ffplay exited unexpectedly: %v", err))
}

func (a *playbackAudio) recordWarning(warning string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.warnings = append(a.warnings, warning)
	a.warningCount++
	a.status.WarningCount = a.warningCount
}

func printAudioStats(w io.Writer, status playbackAudioStatus) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "audio: %s\n", enabledText(status.Enabled))
	fmt.Fprintf(w, "audio stream: %s\n", yesNo(status.HasStream))
	fmt.Fprintf(w, "audio started: %s\n", yesNo(status.Started))
	fmt.Fprintf(w, "audio warnings: %d\n", status.WarningCount)
}

func enabledText(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

type ffplayAudioBackend struct{}

func (ffplayAudioBackend) Start(ctx context.Context, inputPath string) (playbackAudioProcess, error) {
	cmd, stdin, err := media.StartFFplayAudioContext(ctx, inputPath, nil)
	if err != nil {
		return nil, err
	}
	process := &ffplayAudioProcess{
		cmd:   cmd,
		stdin: stdin,
		done:  make(chan error, 1),
	}
	go func() {
		process.done <- cmd.Wait()
		close(process.done)
	}()
	return process, nil
}

type ffplayAudioProcess struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	done   chan error
	closed bool
}

func (p *ffplayAudioProcess) TogglePause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.stdin == nil {
		return nil
	}
	_, err := io.WriteString(p.stdin, "p")
	return err
}

func (p *ffplayAudioProcess) Stop() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	stdin := p.stdin
	cmd := p.cmd
	p.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}

func (p *ffplayAudioProcess) Done() <-chan error {
	return p.done
}
```

- [ ] **Step 4: Run focused CLI tests**

Run:

```bash
gofmt -w packages/core/internal/cli/play_audio.go packages/core/internal/cli/play_audio_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: pass.

- [ ] **Step 5: Commit playback audio controller**

Run:

```bash
git add packages/core/internal/cli/play_audio.go packages/core/internal/cli/play_audio_test.go
git commit --no-gpg-sign -m "feat: add playback audio controller"
```

---

### Task 5: Wire Audio Into Playback Orchestration

**Files:**
- Modify: `packages/core/internal/cli/play.go`
- Modify: `packages/core/internal/cli/play_test.go`

- [ ] **Step 1: Add failing tests for control fan-out**

Append to `packages/core/internal/cli/play_test.go`:

```go
func TestBridgeTerminalControlsTogglesAudioPause(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan terminal.Control, 1)
	out := make(chan player.Control, 1)
	audio := &fakePauseAudio{}

	in <- terminal.TogglePause
	go bridgeTerminalControls(ctx, cancel, in, out, audio)

	select {
	case got := <-out:
		if got != player.TogglePause {
			t.Fatalf("control = %v, want TogglePause", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for player control")
	}
	if audio.toggles != 1 {
		t.Fatalf("audio toggles = %d, want 1", audio.toggles)
	}
}

type fakePauseAudio struct {
	toggles int
}

func (a *fakePauseAudio) TogglePause() {
	a.toggles++
}
```

Add these imports if they are not already present in `packages/core/internal/cli/play_test.go`:

```go
import (
	"context"
	"time"

	"github.com/jass/mojify/packages/core/internal/player"
	"github.com/jass/mojify/packages/core/internal/terminal"
)
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: fail because `bridgeTerminalControls` does not accept an audio pause target.

- [ ] **Step 3: Add `NoAudio` to play options**

Modify `packages/core/internal/cli/play.go`:

```go
type PlayOptions struct {
	Stats   bool
	NoAudio bool
}
```

- [ ] **Step 4: Add a pause target interface**

In `packages/core/internal/cli/play.go`, near `startPlaybackControls`, add:

```go
type audioPauseTarget interface {
	TogglePause()
}
```

- [ ] **Step 5: Create the audio controller before controls**

In `RunPlay`, before `startPlaybackControls`, add:

```go
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   !options.NoAudio,
		HasStream: info.HasAudio,
	})
```

Replace:

```go
controls, stopControls, err := startPlaybackControls(ctx, cancel, stdin)
```

with:

```go
controls, stopControls, err := startPlaybackControls(ctx, cancel, stdin, audio)
```

- [ ] **Step 6: Start audio after controls are initialized**

After the `startPlaybackControls` error check and before `PlayWithControls`, add:

```go
	audio.Start(ctx, inputPath)
	for _, warning := range audio.DrainWarnings() {
		fmt.Fprintln(stderr, warning)
	}
	defer audio.Stop()
```

- [ ] **Step 7: Fan out terminal pause to audio**

Change the `startPlaybackControls` signature in `packages/core/internal/cli/play.go`:

```go
func startPlaybackControls(ctx context.Context, cancel context.CancelFunc, stdin *os.File, audio audioPauseTarget) (<-chan player.Control, func(), error)
```

Inside it, replace:

```go
go bridgeTerminalControls(controlCtx, cancel, terminalControls, playerControls)
```

with:

```go
go bridgeTerminalControls(controlCtx, cancel, terminalControls, playerControls, audio)
```

Change the `bridgeTerminalControls` signature:

```go
func bridgeTerminalControls(ctx context.Context, cancel context.CancelFunc, in <-chan terminal.Control, out chan<- player.Control, audio audioPauseTarget)
```

Inside the `terminal.TogglePause` case, before sending to the player control channel, add:

```go
if audio != nil {
	audio.TogglePause()
}
```

- [ ] **Step 8: Print deferred runtime warnings and audio stats**

In `RunPlay`, after `presenter.Stop()` succeeds and before returning `resultErr`, add:

```go
for _, warning := range audio.DrainWarnings() {
	fmt.Fprintln(stderr, warning)
}
printStats(stderr, options, metrics, audio.Status())
return resultErr
```

Replace the existing call:

```go
printStats(stderr, options, metrics)
return resultErr
```

with the new block above.

Update `printStats`:

```go
func printStats(w io.Writer, options PlayOptions, metrics *playback.Metrics, audioStatus playbackAudioStatus) {
	if !options.Stats || metrics == nil {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, metrics.Summary())
	printAudioStats(w, audioStatus)
}
```

- [ ] **Step 9: Run focused CLI tests**

Run:

```bash
gofmt -w packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go
GOCACHE=/private/tmp/mojify-gocache go test ./packages/core/internal/cli
```

Expected: pass.

- [ ] **Step 10: Commit playback orchestration**

Run:

```bash
git add packages/core/internal/cli/play.go packages/core/internal/cli/play_test.go
git commit --no-gpg-sign -m "feat: wire live audio into playback"
```

---

### Task 6: Playback Audio QA Docs

**Files:**
- Modify: `docs/qa/playback-quality.md`
- Modify: `docs/superpowers/plans/2026-06-02-mojify-playback-audio.md`

Status note (Worker F, 2026-06-02): Playback Audio QA docs were added to `docs/qa/playback-quality.md`; the requested `rg` verification matched the new QA section and checklist terms. Commit step intentionally skipped because this worker was asked not to commit.

- [ ] **Step 1: Add playback audio QA section**

In `docs/qa/playback-quality.md`, after `Manual Runs`, add:

````md
## Playback Audio QA

Use a local ignored real sample with an audio stream, such as `dist/iris.mp4`, for live audio checks.

```bash
bun run build
./bin/mojify probe dist/iris.mp4
./bin/mojify play --stats dist/iris.mp4
./bin/mojify play --no-audio --stats dist/iris.mp4
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

Expected:

- `probe` prints `audio: yes` for the real sample.
- Default playback has audible source audio when `ffplay` and an audio device are available.
- Space pauses and resumes both terminal frames and audio.
- `q` stops terminal playback and audio.
- Ctrl-C restores the terminal and stops audio.
- `--no-audio` plays the same visual content silently.
- Silent generated QA clips do not print audio warnings.
- `play --stats` reports audio enabled/disabled, source audio presence, whether audio started, and warning count.
- If `ffplay` is unavailable or the audio device cannot open, visual playback continues and one concise audio warning is printed.
````

- [ ] **Step 2: Add playback audio checklist items**

In `docs/qa/playback-quality.md`, under `Visual Checklist`, append:

```md
- Default playback starts live audio for source media with audio when `ffplay` and an audio device are available.
- `--no-audio` keeps playback silent.
- Space pauses and resumes both visuals and live audio.
- `q` and Ctrl-C stop live audio.
- Runtime audio warnings are printed after playback exits, not during terminal frame presentation.
```

Under `Notes To Record`, append:

```md
- Whether the source has audio according to `mojify probe`.
- Whether `ffplay` is available on PATH.
- Whether audio started, paused, resumed, and stopped as expected.
- Whether any audio warnings appeared.
```

- [ ] **Step 3: Run a docs grep check**

Run:

```bash
rg -n "Playback Audio QA|--no-audio|audio: yes|ffplay|pauses and resumes both" docs/qa/playback-quality.md
```

Expected: output includes the new playback audio QA section and checklist terms.

- [ ] **Step 4: Commit QA docs**

Run:

```bash
git add docs/qa/playback-quality.md docs/superpowers/plans/2026-06-02-mojify-playback-audio.md
git commit --no-gpg-sign -m "docs: add playback audio qa"
```

---

### Task 7: Final Verification and Manual Audio Smoke

**Files:**
- Read: `CONTEXT.md`
- Read: `docs/adr/0024-use-ffplay-for-live-terminal-audio.md`
- Read: `docs/adr/0025-enable-live-playback-audio-by-default.md`
- Read: `docs/qa/playback-quality.md`
- Read: `git status --short --branch`

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
bun run test
```

Expected: exit code 0.

- [ ] **Step 4: Run typecheck and build**

Run:

```bash
bun run typecheck
bun run build
```

Expected: both commands exit 0.

- [ ] **Step 5: Generate QA clips**

Run:

```bash
bun run qa:clips
```

Expected: generated clips exist under `dist/qa/`.

- [ ] **Step 6: Check probe audio output**

Run:

```bash
./bin/mojify probe dist/qa/low-motion-bars.mp4
```

Expected: output includes `audio: no`.

If `dist/iris.mp4` exists, run:

```bash
./bin/mojify probe dist/iris.mp4
```

Expected: output includes `audio: yes`.

- [ ] **Step 7: Run visual playback silent-source smoke**

Run:

```bash
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

Expected:

- Visual playback works.
- No audio warning is printed for the silent generated clip.
- Stats include `audio: enabled`, `audio stream: no`, `audio started: no`, and `audio warnings: 0`.

- [ ] **Step 8: Run manual real-audio smoke when local sample exists**

If `dist/iris.mp4` exists, run:

```bash
./bin/mojify play --stats dist/iris.mp4
```

Expected:

- Visual playback works.
- Audio is audible when `ffplay` and an audio device are available.
- Space pauses and resumes both audio and visuals.
- `q` exits and stops audio.
- Stats include `audio: enabled`, `audio stream: yes`, and `audio started: yes` when audio starts.

- [ ] **Step 9: Run manual opt-out smoke when local sample exists**

If `dist/iris.mp4` exists, run:

```bash
./bin/mojify play --no-audio --stats dist/iris.mp4
```

Expected:

- Visual playback works.
- No live audio is audible.
- Stats include `audio: disabled`, `audio stream: yes`, `audio started: no`, and `audio warnings: 0`.

- [ ] **Step 10: Validate ffplay pause control**

During the real-audio smoke, press Space twice.

Expected:

- First Space pauses visual playback and audio promptly.
- Second Space resumes visual playback and audio.
- If audio does not pause through ffplay stdin control, update the implementation to use process suspend/resume for the ffplay process on Unix, then rerun Steps 8 and 9.

- [ ] **Step 11: Run final diff checks**

Run:

```bash
git diff --check
git status --short --branch
git log --oneline --decorate -5
```

Expected:

- `git diff --check` exits 0.
- Working tree is clean after commits.
- Recent commits include playback audio work.

---

## Self-Review

- Spec coverage: The plan covers default-on audio, `--no-audio`, probe audio metadata, ffplay backend, best-effort fallback, pause/resume fan-out, quit/Ctrl-C cleanup, deferred runtime warnings, sparse stats, QA docs, and final verification.
- Placeholder scan: No unresolved placeholder tokens or unspecified test-writing steps remain.
- Type consistency: `Command.NoAudio`, `PlayOptions.NoAudio`, `Info.HasAudio`, `playbackAudioStatus`, `playbackAudioBackend`, and `playbackAudioProcess` are introduced before later use.
- Scope check: URL input, distribution, mute, volume, stream selection, active drift correction, export audio changes, and fatal audio fallback are kept out of scope.
