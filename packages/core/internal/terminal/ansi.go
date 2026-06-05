package terminal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jass/mojify/packages/core/internal/render"
)

const (
	EnterAltScreen          = "\x1b[?1049h"
	ExitAltScreen           = "\x1b[?1049l"
	HideCursor              = "\x1b[?25l"
	ShowCursor              = "\x1b[?25h"
	CursorHome              = "\x1b[H"
	ClearToEnd              = "\x1b[J"
	BeginSynchronizedUpdate = "\x1b[?2026h"
	EndSynchronizedUpdate   = "\x1b[?2026l"
	Reset                   = "\x1b[0m"
)

var ErrInvalidCharacterFrame = errors.New("invalid character frame")

func CursorPosition(row int, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

func SerializeFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	b.WriteString(CursorHome)
	b.WriteString(ClearToEnd)
	var lastR, lastG, lastB uint8
	hasColor := false

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			cell := frame.Cells[y*frame.Width+x]
			if cell.HasColor {
				if !hasColor || cell.R != lastR || cell.G != lastG || cell.B != lastB {
					fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
					lastR, lastG, lastB = cell.R, cell.G, cell.B
					hasColor = true
				}
			} else if hasColor {
				b.WriteString("\x1b[39m")
				hasColor = false
			}
			b.WriteRune(cell.Ch)
		}
		if y != frame.Height-1 {
			b.WriteString("\r\n")
		}
	}

	b.WriteString(Reset)
	return b.String()
}

func SerializeFramePatch(previous, current render.CharacterFrame) (string, error) {
	if err := validateCharacterFrame(previous); err != nil {
		return "", err
	}
	if err := validateCharacterFrame(current); err != nil {
		return "", err
	}
	if previous.Width != current.Width || previous.Height != current.Height {
		return "", fmt.Errorf("frame dimensions differ: previous=%dx%d current=%dx%d", previous.Width, previous.Height, current.Width, current.Height)
	}

	var b strings.Builder
	var foreground foregroundState
	for y := 0; y < current.Height; y++ {
		x := 0
		for x < current.Width {
			index := y*current.Width + x
			if sameCell(previous.Cells[index], current.Cells[index]) {
				x++
				continue
			}

			startX := x
			for x < current.Width {
				index = y*current.Width + x
				if sameCell(previous.Cells[index], current.Cells[index]) {
					break
				}
				x++
			}

			b.WriteString(CursorPosition(y+1, startX+1))
			writeColoredRun(&b, current.Cells[y*current.Width+startX:y*current.Width+x], &foreground)
		}
	}

	if b.Len() == 0 {
		return "", nil
	}
	b.WriteString(Reset)
	return b.String(), nil
}

func validateCharacterFrame(frame render.CharacterFrame) error {
	if frame.Width <= 0 || frame.Height <= 0 || len(frame.Cells) != frame.Width*frame.Height {
		return fmt.Errorf("%w: width=%d height=%d cells=%d", ErrInvalidCharacterFrame, frame.Width, frame.Height, len(frame.Cells))
	}
	return nil
}

func sameCell(a, b render.Cell) bool {
	if a.Ch != b.Ch || a.HasColor != b.HasColor {
		return false
	}
	if !a.HasColor {
		return true
	}
	return a.R == b.R && a.G == b.G && a.B == b.B
}

type foregroundState struct {
	hasColor     bool
	lastR, lastG uint8
	lastB        uint8
}

func writeColoredRun(b *strings.Builder, cells []render.Cell, foreground *foregroundState) {
	for _, cell := range cells {
		if cell.HasColor {
			if !foreground.hasColor || cell.R != foreground.lastR || cell.G != foreground.lastG || cell.B != foreground.lastB {
				fmt.Fprintf(b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
				foreground.lastR, foreground.lastG, foreground.lastB = cell.R, cell.G, cell.B
				foreground.hasColor = true
			}
		} else if foreground.hasColor {
			b.WriteString("\x1b[39m")
			foreground.hasColor = false
		}
		b.WriteRune(cell.Ch)
	}
}
