package render

import "math"

const CellAspect = 2.0
const minCols = 10
const minRows = 5

type InputSize struct {
	Width  int
	Height int
}

type TerminalSize struct {
	Cols int
	Rows int
}

type Grid struct {
	Cols int
	Rows int
}

func FitGrid(input InputSize, term TerminalSize) Grid {
	maxCols := max(term.Cols, minCols)
	maxRows := max(term.Rows, minRows)
	if input.Width <= 0 || input.Height <= 0 {
		return Grid{Cols: maxCols, Rows: maxRows}
	}

	aspect := float64(input.Width) / float64(input.Height)
	rowsFromCols := int(math.Floor(float64(maxCols) / (aspect * CellAspect)))
	if rowsFromCols < minRows {
		return Grid{Cols: maxCols, Rows: minRows}
	}
	if rowsFromCols <= maxRows {
		return Grid{Cols: maxCols, Rows: rowsFromCols}
	}

	colsFromRows := int(math.Floor(float64(maxRows) * aspect * CellAspect))
	return Grid{Cols: max(minCols, min(maxCols, colsFromRows)), Rows: maxRows}
}
