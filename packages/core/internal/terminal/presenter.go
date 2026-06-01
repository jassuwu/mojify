package terminal

import (
	"fmt"
	"io"

	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter struct {
	Out io.Writer
}

func (p Presenter) Start() error {
	_, err := fmt.Fprint(p.Out, EnterAltScreen, HideCursor, CursorHome, ClearToEnd)
	return err
}

func (p Presenter) Present(frame render.CharacterFrame) error {
	_, err := fmt.Fprint(p.Out, SerializeFrame(frame))
	return err
}

func (p Presenter) Stop() error {
	_, err := fmt.Fprint(p.Out, Reset, ShowCursor, ExitAltScreen)
	return err
}
