package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/player"
	"github.com/jass/mojify/packages/core/internal/terminal"
)

func TestPlaybackResultReturnsPrimaryPlaybackError(t *testing.T) {
	playErr := errors.New("present failed")
	renderErr := errors.New("read failed after teardown")
	decoderErr := errors.New("signal: killed")

	err := playbackResult(nil, playErr, renderErr, decoderErr)
	if !errors.Is(err, playErr) {
		t.Fatalf("err = %v, want playback error", err)
	}
}

func TestPlaybackResultReturnsDecoderFailureAfterCleanFrameEOF(t *testing.T) {
	decoderErr := errors.New("ffmpeg exited 1")

	err := playbackResult(nil, nil, nil, decoderErr)
	if !errors.Is(err, decoderErr) {
		t.Fatalf("err = %v, want decoder error", err)
	}
}

func TestPlaybackResultTreatsCancellationAsClean(t *testing.T) {
	decoderErr := errors.New("signal: killed")

	err := playbackResult(context.Canceled, player.ErrCancelled, nil, decoderErr)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestPlaybackResultTreatsCancellationTeardownReadErrorAsClean(t *testing.T) {
	renderErr := errors.New("read pipe: file already closed")
	decoderErr := errors.New("signal: killed")

	err := playbackResult(context.Canceled, player.ErrCancelled, renderErr, decoderErr)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestPrintStatsWritesSummaryOnlyWhenEnabled(t *testing.T) {
	metrics := playback.NewMetrics(4, 2)
	start := time.Unix(10, 0)
	metrics.Start(start)
	metrics.RecordRendered(time.Millisecond)
	metrics.RecordPresented(100, time.Millisecond)
	metrics.Finish(start.Add(time.Second))

	var out bytes.Buffer
	printStats(&out, PlayOptions{Stats: true}, metrics, playbackAudioStatus{
		Enabled:      true,
		HasStream:    true,
		Started:      true,
		WarningCount: 1,
	})
	if !strings.Contains(out.String(), "playback stats") {
		t.Fatalf("stats output missing summary:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "audio started: yes") {
		t.Fatalf("stats output missing audio summary:\n%s", out.String())
	}

	out.Reset()
	printStats(&out, PlayOptions{Stats: false}, metrics, playbackAudioStatus{})
	if out.Len() != 0 {
		t.Fatalf("stats disabled wrote %q", out.String())
	}
}

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
