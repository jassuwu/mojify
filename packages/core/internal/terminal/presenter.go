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
	n, err := io.WriteString(p.Out, output)
	if err == nil && p.Metrics != nil {
		p.Metrics.RecordPresented(n, time.Since(start))
	}
	return err
}

func (p Presenter) Stop() error {
	_, err := fmt.Fprint(p.Out, Reset, ShowCursor, ExitAltScreen)
	return err
}
