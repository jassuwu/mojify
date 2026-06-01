package terminal

import (
	"context"
	"io"
)

type Control int

const (
	Quit Control = iota
	TogglePause
)

func ReadControls(ctx context.Context, r io.Reader, out chan<- Control) {
	defer close(out)
	buf := make([]byte, 1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := r.Read(buf)
		if n > 0 {
			switch buf[0] {
			case 'q', 'Q':
				sendControl(ctx, out, Quit)
				return
			case ' ':
				if !sendControl(ctx, out, TogglePause) {
					return
				}
			}
		}
		if err != nil || n == 0 {
			return
		}
	}
}

func sendControl(ctx context.Context, out chan<- Control, control Control) bool {
	select {
	case out <- control:
		return true
	case <-ctx.Done():
		return false
	}
}
