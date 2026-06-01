package render

import "math"

const densityRamp = " .;coPO?#@"

type DefaultRenderer struct{}

func (DefaultRenderer) Render(frame RGBFrame, grid Grid) CharacterFrame {
	cells := make([]Cell, grid.Cols*grid.Rows)
	for gy := 0; gy < grid.Rows; gy++ {
		for gx := 0; gx < grid.Cols; gx++ {
			sx := gx * frame.Width / grid.Cols
			sy := gy * frame.Height / grid.Rows
			r, g, b := frame.RGBAt(sx, sy)
			luma := luminance(r, g, b)
			ch := densityChar(luma)
			if edge, ok := edgeGlyph(frame, sx, sy); ok {
				ch = edge
			}
			cells[gy*grid.Cols+gx] = Cell{Ch: ch, R: r, G: g, B: b}
		}
	}
	return CharacterFrame{Width: grid.Cols, Height: grid.Rows, Cells: cells}
}

func luminance(r uint8, g uint8, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

func densityChar(luma float64) rune {
	// Bias mid-tones toward clearer terminal contrast while preserving the golden ramp anchors.
	normalized := luma / 255.0
	if normalized < 0.5 {
		normalized = 0.5 * math.Pow(normalized*2, 4)
	} else {
		normalized = 1 - 0.5*math.Pow((1-normalized)*2, 4)
	}
	index := int(math.Round(normalized * float64(len([]rune(densityRamp))-1)))
	if index < 0 {
		index = 0
	}
	runes := []rune(densityRamp)
	if index >= len(runes) {
		index = len(runes) - 1
	}
	return runes[index]
}

func edgeGlyph(frame RGBFrame, x int, y int) (rune, bool) {
	if frame.Width < 3 || frame.Height < 3 {
		return 0, false
	}
	gx := sobelX(frame, x, y)
	gy := sobelY(frame, x, y)
	mag := math.Sqrt(gx*gx + gy*gy)
	if mag < 180 {
		return 0, false
	}
	angle := math.Atan2(gy, gx) * 180 / math.Pi
	if angle < 0 {
		angle += 180
	}
	switch {
	case angle < 22.5 || angle >= 157.5:
		return '|', true
	case angle < 67.5:
		return '/', true
	case angle < 112.5:
		return '-', true
	default:
		return '\\', true
	}
}

func grayAt(frame RGBFrame, x int, y int) float64 {
	r, g, b := frame.RGBAt(x, y)
	return luminance(r, g, b)
}

func sobelX(frame RGBFrame, x int, y int) float64 {
	return -grayAt(frame, x-1, y-1) + grayAt(frame, x+1, y-1) -
		2*grayAt(frame, x-1, y) + 2*grayAt(frame, x+1, y) -
		grayAt(frame, x-1, y+1) + grayAt(frame, x+1, y+1)
}

func sobelY(frame RGBFrame, x int, y int) float64 {
	return -grayAt(frame, x-1, y-1) - 2*grayAt(frame, x, y-1) - grayAt(frame, x+1, y-1) +
		grayAt(frame, x-1, y+1) + 2*grayAt(frame, x, y+1) + grayAt(frame, x+1, y+1)
}
