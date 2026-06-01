package cli

import (
	"context"
	"errors"
	"testing"

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
