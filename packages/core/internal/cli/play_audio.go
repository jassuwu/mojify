package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
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
	mu           sync.Mutex
	process      playbackAudioProcess
	status       playbackAudioStatus
	warnings     []string
	warningCount int
	backend      playbackAudioBackend
	stopping     bool
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
	if !ok {
		return
	}

	a.mu.Lock()
	stopping := a.stopping
	if a.process == process {
		a.process = nil
	}
	a.mu.Unlock()
	if err == nil || (stopping && isExpectedStoppedAudioProcessError(err)) {
		return
	}

	a.recordWarning(fmt.Sprintf("audio warning: ffplay exited unexpectedly: %v", err))
}

func isExpectedStoppedAudioProcessError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "signal: killed") || strings.Contains(message, "signal: terminated")
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
