package exporter

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/jass/mojify/packages/core/internal/render"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var defaultBackground = color.RGBA{R: 8, G: 8, B: 8, A: 255}

type Rasterizer struct {
	Face       font.Face
	CellWidth  int
	CellHeight int
	Background color.RGBA
}

func NewRasterizer(face font.Face) Rasterizer {
	return Rasterizer{
		Face:       face,
		CellWidth:  ExportCellWidth,
		CellHeight: ExportCellHeight,
		Background: defaultBackground,
	}
}

func (r Rasterizer) Rasterize(frame render.CharacterFrame, layout Layout) ([]byte, error) {
	if err := r.validate(frame, layout); err != nil {
		return nil, err
	}

	bounds := image.Rect(0, 0, layout.OutputWidth, layout.OutputHeight)
	canvas := image.NewRGBA(bounds)
	background := r.Background
	if background.A == 0 {
		background.A = 255
	}
	draw.Draw(canvas, bounds, image.NewUniform(background), image.Point{}, draw.Src)

	metrics := r.Face.Metrics()
	ascent := metrics.Ascent.Ceil()
	if ascent < 1 {
		ascent = r.CellHeight
	}

	for row := 0; row < frame.Height; row++ {
		for col := 0; col < frame.Width; col++ {
			cell := frame.Cells[row*frame.Width+col]
			ch := cell.Ch
			if ch == 0 {
				ch = ' '
			}
			glyph := string(ch)
			x := col * r.CellWidth
			if advance := font.MeasureString(r.Face, glyph).Ceil(); advance < r.CellWidth {
				x += (r.CellWidth - advance) / 2
			}
			y := row*r.CellHeight + ascent
			glyphColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
			if cell.HasColor {
				glyphColor = color.RGBA{R: cell.R, G: cell.G, B: cell.B, A: 255}
			}
			drawer := font.Drawer{
				Dst:  canvas,
				Src:  image.NewUniform(glyphColor),
				Face: r.Face,
				Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
			}
			drawer.DrawString(glyph)
		}
	}

	return packedRGB(canvas, layout.OutputWidth, layout.OutputHeight), nil
}

func (r Rasterizer) validate(frame render.CharacterFrame, layout Layout) error {
	if r.Face == nil {
		return fmt.Errorf("font face is required")
	}
	if r.CellWidth <= 0 || r.CellHeight <= 0 {
		return fmt.Errorf("cell size must be positive, got %dx%d", r.CellWidth, r.CellHeight)
	}
	if layout.OutputWidth <= 0 || layout.OutputHeight <= 0 {
		return fmt.Errorf("output size must be positive, got %dx%d", layout.OutputWidth, layout.OutputHeight)
	}
	if layout.Grid.Cols <= 0 || layout.Grid.Rows <= 0 {
		return fmt.Errorf("layout grid must be positive, got %dx%d", layout.Grid.Cols, layout.Grid.Rows)
	}
	if layout.OutputWidth != layout.Grid.Cols*r.CellWidth || layout.OutputHeight != layout.Grid.Rows*r.CellHeight {
		return fmt.Errorf(
			"layout output size %dx%d does not match grid %dx%d and cell size %dx%d",
			layout.OutputWidth,
			layout.OutputHeight,
			layout.Grid.Cols,
			layout.Grid.Rows,
			r.CellWidth,
			r.CellHeight,
		)
	}
	if frame.Width != layout.Grid.Cols || frame.Height != layout.Grid.Rows {
		return fmt.Errorf("frame size %dx%d does not match layout grid %dx%d", frame.Width, frame.Height, layout.Grid.Cols, layout.Grid.Rows)
	}
	if len(frame.Cells) != frame.Width*frame.Height {
		return fmt.Errorf("frame has %d cells, want %d", len(frame.Cells), frame.Width*frame.Height)
	}
	return nil
}

func packedRGB(canvas *image.RGBA, width int, height int) []byte {
	rgb := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			src := y*canvas.Stride + x*4
			dst := (y*width + x) * 3
			rgb[dst] = canvas.Pix[src]
			rgb[dst+1] = canvas.Pix[src+1]
			rgb[dst+2] = canvas.Pix[src+2]
		}
	}
	return rgb
}
