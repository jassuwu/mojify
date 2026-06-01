package terminal

import (
	"fmt"
	"strings"

	"github.com/jass/mojify/packages/core/internal/render"
)

const (
	EnterAltScreen = "\x1b[?1049h"
	ExitAltScreen  = "\x1b[?1049l"
	HideCursor     = "\x1b[?25l"
	ShowCursor     = "\x1b[?25h"
	CursorHome     = "\x1b[H"
	ClearToEnd     = "\x1b[J"
	Reset          = "\x1b[0m"
)

func SerializeFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	b.WriteString(CursorHome)
	b.WriteString(ClearToEnd)
	var lastR, lastG, lastB uint8
	hasColor := false

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			cell := frame.Cells[y*frame.Width+x]
			if !hasColor || cell.R != lastR || cell.G != lastG || cell.B != lastB {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", cell.R, cell.G, cell.B)
				lastR, lastG, lastB = cell.R, cell.G, cell.B
				hasColor = true
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
