package render

type RGBFrame struct {
	Width  int
	Height int
	Data   []byte
}

type Cell struct {
	Ch       rune
	HasColor bool
	R, G, B  uint8
}

type CharacterFrame struct {
	Width  int
	Height int
	Cells  []Cell
}

func NewRGBFrame(width int, height int, data []byte) RGBFrame {
	return RGBFrame{Width: width, Height: height, Data: data}
}

func (f RGBFrame) RGBAt(x int, y int) (uint8, uint8, uint8) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= f.Width {
		x = f.Width - 1
	}
	if y >= f.Height {
		y = f.Height - 1
	}
	offset := (y*f.Width + x) * 3
	return f.Data[offset], f.Data[offset+1], f.Data[offset+2]
}
