package cli

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPlaybackAudioConstructionDoesNotStartBackend(t *testing.T) {
	backend := &fakeAudioBackend{}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	if backend.starts != 0 {
		t.Fatalf("backend starts after construction = %d, want 0", backend.starts)
	}
	audio.Start(context.Background(), "clip.mp4")
	if backend.starts != 1 {
		t.Fatalf("backend starts after Start = %d, want 1", backend.starts)
	}
}

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

func TestFFplayAudioProcessTogglePauseDoesNotUseStdinControl(t *testing.T) {
	stdin := &recordingWriteCloser{}
	process := &ffplayAudioProcess{
		stdin: stdin,
		done:  make(chan error),
	}
	if err := process.TogglePause(); err != nil {
		t.Fatalf("TogglePause returned error: %v", err)
	}
	if stdin.writes != 0 {
		t.Fatalf("stdin writes = %d, want 0; ffplay pause should use process pause/resume instead of stdin control", stdin.writes)
	}
}

func TestFFplayAudioProcessTogglePauseAlternatesProcessPauseAndResume(t *testing.T) {
	var pauses []bool
	process := &ffplayAudioProcess{
		done: make(chan error),
		pauseProcess: func(cmd *exec.Cmd, pause bool) error {
			pauses = append(pauses, pause)
			return nil
		},
	}
	if err := process.TogglePause(); err != nil {
		t.Fatalf("first TogglePause returned error: %v", err)
	}
	if err := process.TogglePause(); err != nil {
		t.Fatalf("second TogglePause returned error: %v", err)
	}
	if len(pauses) != 2 || !pauses[0] || pauses[1] {
		t.Fatalf("pause sequence = %v, want [true false]", pauses)
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

func TestPlaybackAudioDoneRecordsUnexpectedProcessExitWarning(t *testing.T) {
	process := &fakeAudioProcess{done: make(chan error, 1)}
	backend := &fakeAudioBackend{process: process}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	process.done <- errors.New("device lost")
	time.Sleep(25 * time.Millisecond)
	status := audio.Status()
	if status.WarningCount != 1 {
		t.Fatalf("WarningCount = %d, want 1", status.WarningCount)
	}
	warnings := audio.DrainWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "device lost") {
		t.Fatalf("warnings = %#v, want device lost warning", warnings)
	}
}

func TestPlaybackAudioDoneClearsProcessAfterNormalExit(t *testing.T) {
	process := &fakeAudioProcess{done: make(chan error, 1)}
	backend := &fakeAudioBackend{process: process}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	process.done <- nil
	time.Sleep(25 * time.Millisecond)
	audio.TogglePause()
	if process.toggles != 0 {
		t.Fatalf("toggles = %d, want 0 after process exit", process.toggles)
	}
	if got := audio.Status().WarningCount; got != 0 {
		t.Fatalf("WarningCount = %d, want 0", got)
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

func TestPlaybackAudioStopDoesNotSuppressUnexpectedProcessExitWarning(t *testing.T) {
	process := &fakeAudioProcess{done: make(chan error, 1)}
	backend := &fakeAudioBackend{process: process}
	audio := newPlaybackAudio(playbackAudioOptions{
		Enabled:   true,
		HasStream: true,
		Backend:   backend,
	})
	audio.Start(context.Background(), "clip.mp4")
	audio.Stop()
	process.done <- errors.New("device lost")
	time.Sleep(25 * time.Millisecond)
	if got := audio.Status().WarningCount; got != 1 {
		t.Fatalf("WarningCount = %d, want 1", got)
	}
	warnings := audio.DrainWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "device lost") {
		t.Fatalf("warnings = %#v, want device lost warning", warnings)
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

type recordingWriteCloser struct {
	writes int
}

func (w *recordingWriteCloser) Write(data []byte) (int, error) {
	w.writes++
	return len(data), nil
}

func (w *recordingWriteCloser) Close() error {
	return nil
}
