package terminal

import (
	"fmt"
	"io"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter struct {
	Out     io.Writer
	Metrics *playback.Metrics
}

func (p Presenter) Start() error {
	_, err := fmt.Fprint(p.Out, EnterAltScreen, HideCursor, CursorHome, ClearToEnd)
	return err
}

func (p Presenter) Present(frame render.CharacterFrame) error {
	start := time.Now()
	output := SerializeFrame(frame)
	n, err := writeSynchronizedFrame(p.Out, output)
	if err == nil && p.Metrics != nil {
		p.Metrics.RecordPresented(n, time.Since(start))
	}
	return err
}

func writeSynchronizedFrame(w io.Writer, output string) (int, error) {
	total, err := io.WriteString(w, BeginSynchronizedUpdate)
	if err != nil {
		return total, err
	}

	n, frameErr := io.WriteString(w, output)
	total += n

	n, endErr := io.WriteString(w, EndSynchronizedUpdate)
	total += n

	if frameErr != nil {
		return total, frameErr
	}
	if endErr != nil {
		return total, endErr
	}
	return total, nil
}

func (p Presenter) Stop() error {
	_, err := fmt.Fprint(p.Out, Reset, ShowCursor, ExitAltScreen)
	return err
}
