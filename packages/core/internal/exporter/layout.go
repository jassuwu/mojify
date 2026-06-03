package exporter

import (
	"fmt"
	"math"
	"time"

	"github.com/jass/mojify/packages/core/internal/render"
)

const ExportCellWidth = 8
const ExportCellHeight = 8
const DefaultExportMaxWidth = 1280

type Options struct {
	Width               int
	FPS                 float64
	Bitrate             string
	Overwrite           bool
	ProgressInteractive bool
	ProgressClock       func() time.Time
	Stats               bool
	Workers             int
	MetricsClock        func() time.Time
	InputLabel          string
	HasAt               bool
	AtSeconds           float64
	HasDuration         bool
	DurationSeconds     float64
	Format              OutputFormat
}

type InputInfo struct {
	Width  int
	Height int
	FPS    float64
}

type Layout struct {
	OutputWidth  int
	OutputHeight int
	Grid         render.Grid
	FPS          float64
}

func ResolveLayout(input InputInfo, options Options) (Layout, error) {
	if input.Width <= 0 {
		return Layout{}, fmt.Errorf("input width must be positive, got %d", input.Width)
	}
	if input.Height <= 0 {
		return Layout{}, fmt.Errorf("input height must be positive, got %d", input.Height)
	}
	if options.Width < 0 {
		return Layout{}, fmt.Errorf("export width must be non-negative, got %d", options.Width)
	}
	if options.FPS < 0 {
		return Layout{}, fmt.Errorf("export FPS must be non-negative, got %v", options.FPS)
	}

	fps := input.FPS
	if fps <= 0 {
		fps = 24
	}
	if options.FPS > 0 {
		fps = options.FPS
	}

	if options.Format.Text {
		cols := options.Width
		if cols == 0 {
			cols = min(input.Width, DefaultExportMaxWidth) / ExportCellWidth
		}
		if cols <= 0 {
			cols = 1
		}
		outputWidth := cols * ExportCellWidth
		derivedRows := int(math.Round(float64(cols) * float64(input.Height) / float64(input.Width)))
		rows := max(derivedRows, 1)
		return Layout{
			OutputWidth:  outputWidth,
			OutputHeight: rows * ExportCellHeight,
			Grid: render.Grid{
				Cols: cols,
				Rows: rows,
			},
			FPS: fps,
		}, nil
	}

	width := options.Width
	if width == 0 {
		width = min(input.Width, DefaultExportMaxWidth)
	}
	outputWidth := alignUp(width, ExportCellWidth)
	derivedHeight := int(math.Round(float64(outputWidth) * float64(input.Height) / float64(input.Width)))
	outputHeight := alignUp(max(derivedHeight, 1), ExportCellHeight)

	return Layout{
		OutputWidth:  outputWidth,
		OutputHeight: outputHeight,
		Grid: render.Grid{
			Cols: outputWidth / ExportCellWidth,
			Rows: outputHeight / ExportCellHeight,
		},
		FPS: fps,
	}, nil
}

func alignUp(value int, multiple int) int {
	if value <= 0 {
		return multiple
	}
	remainder := value % multiple
	if remainder == 0 {
		return value
	}
	return value + multiple - remainder
}
