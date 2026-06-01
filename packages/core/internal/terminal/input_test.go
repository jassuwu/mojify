package terminal

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestReadControlsEmitsPauseAndQuit(t *testing.T) {
	out := make(chan Control, 2)
	ReadControls(context.Background(), strings.NewReader(" q"), out)

	if got := <-out; got != TogglePause {
		t.Fatalf("first control = %v, want %v", got, TogglePause)
	}
	if got := <-out; got != Quit {
		t.Fatalf("second control = %v, want %v", got, Quit)
	}
	if _, ok := <-out; ok {
		t.Fatal("ReadControls left output channel open")
	}
}

func TestReadControlsReturnsWhenCancelledBeforeSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan Control)
	done := make(chan struct{})
	reader := cancelAfterRead{
		value:  ' ',
		cancel: cancel,
	}

	go func() {
		defer close(done)
		ReadControls(ctx, &reader, out)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("ReadControls blocked sending after cancellation")
	}
}

type cancelAfterRead struct {
	value  byte
	cancel context.CancelFunc
	read   bool
}

func (r *cancelAfterRead) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.EOF
	}
	r.read = true
	p[0] = r.value
	r.cancel()
	return 1, nil
}
