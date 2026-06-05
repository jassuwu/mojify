package exporter

import (
	"fmt"
	"strings"

	"github.com/jass/mojify/packages/core/internal/render"
)

const ansiReset = "\x1b[0m"

func SerializeTextFrame(frame render.CharacterFrame, format OutputFormat) (string, error) {
	if err := validateTextFrame(frame); err != nil {
		return "", err
	}

	switch format.Extension {
	case ".txt":
		return serializePlainTextFrame(frame), nil
	case ".ansi":
		return serializeANSITextFrame(frame), nil
	default:
		return "", fmt.Errorf("unsupported text format %q", format.Extension)
	}
}

func validateTextFrame(frame render.CharacterFrame) error {
	if frame.Width <= 0 || frame.Height <= 0 {
		return fmt.Errorf("invalid character frame: width=%d height=%d cells=%d", frame.Width, frame.Height, len(frame.Cells))
	}
	if len(frame.Cells) != frame.Width*frame.Height {
		return fmt.Errorf("invalid character frame: width=%d height=%d cells=%d", frame.Width, frame.Height, len(frame.Cells))
	}
	return nil
}

func serializePlainTextFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	for row := 0; row < frame.Height; row++ {
		for col := 0; col < frame.Width; col++ {
			b.WriteRune(textCellRune(frame.Cells[row*frame.Width+col]))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func serializeANSITextFrame(frame render.CharacterFrame) string {
	var b strings.Builder
	var lastR, lastG, lastB uint8
	hasColor := false

	for row := 0; row < frame.Height; row++ {
		for col := 0; col < frame.Width; col++ {
			cell := frame.Cells[row*frame.Width+col]
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
			b.WriteRune(textCellRune(cell))
		}
		if row == frame.Height-1 {
			b.WriteString(ansiReset)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func textCellRune(cell render.Cell) rune {
	if cell.Ch == 0 {
		return ' '
	}
	return cell.Ch
}
