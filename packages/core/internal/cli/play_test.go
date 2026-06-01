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
	printStats(&out, PlayOptions{Stats: true}, metrics)
	if !strings.Contains(out.String(), "playback stats") {
		t.Fatalf("stats output missing summary:\n%s", out.String())
	}

	out.Reset()
	printStats(&out, PlayOptions{Stats: false}, metrics)
	if out.Len() != 0 {
		t.Fatalf("stats disabled wrote %q", out.String())
	}
}
