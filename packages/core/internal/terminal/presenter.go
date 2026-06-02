package terminal

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jass/mojify/packages/core/internal/playback"
	"github.com/jass/mojify/packages/core/internal/render"
)

type Presenter struct {
	Out             io.Writer
	Metrics         *playback.Metrics
	previous        render.CharacterFrame
	hasPrevious     bool
	forceFullRedraw bool
}

func (p *Presenter) Start() error {
	p.resetFrameState()
	_, err := fmt.Fprint(p.Out, EnterAltScreen, HideCursor, CursorHome, ClearToEnd)
	if err != nil {
		return errors.Join(err, p.Stop())
	}
	return err
}

func (p *Presenter) Present(frame render.CharacterFrame) error {
	if err := validateCharacterFrame(frame); err != nil {
		p.forceFullRedraw = true
		return err
	}

	start := time.Now()
	output, err := p.outputFor(frame)
	if err != nil {
		p.forceFullRedraw = true
		return err
	}
	n := 0
	if output != "" {
		n, err = writeSynchronizedFrame(p.Out, output)
		if err != nil {
			p.forceFullRedraw = true
			return err
		}
	}

	p.storePrevious(frame)
	p.forceFullRedraw = false
	if p.Metrics != nil {
		p.Metrics.RecordPresented(n, time.Since(start))
	}
	return nil
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
		return total, errors.Join(frameErr, endErr)
	}
	if endErr != nil {
		return total, endErr
	}
	return total, nil
}

func (p *Presenter) outputFor(frame render.CharacterFrame) (string, error) {
	full := SerializeFrame(frame)
	if p.forceFullRedraw || !p.hasPrevious || dimensionsDiffer(p.previous, frame) {
		return full, nil
	}

	patch, err := SerializeFramePatch(p.previous, frame)
	if err != nil {
		return "", err
	}
	if patch == "" {
		return "", nil
	}
	if len(patch) <= len(full) {
		return patch, nil
	}
	return full, nil
}

func dimensionsDiffer(previous, current render.CharacterFrame) bool {
	return previous.Width != current.Width || previous.Height != current.Height
}

func (p *Presenter) storePrevious(frame render.CharacterFrame) {
	p.previous = render.CharacterFrame{
		Width:  frame.Width,
		Height: frame.Height,
		Cells:  append([]render.Cell(nil), frame.Cells...),
	}
	p.hasPrevious = true
}

func (p *Presenter) resetFrameState() {
	p.previous = render.CharacterFrame{}
	p.hasPrevious = false
	p.forceFullRedraw = false
}

func (p *Presenter) Stop() error {
	p.resetFrameState()
	_, err := fmt.Fprint(p.Out, EndSynchronizedUpdate, Reset, ShowCursor, ExitAltScreen)
	return err
}
