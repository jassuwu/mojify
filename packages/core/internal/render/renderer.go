package render

import (
	"fmt"
	"math"
	"strings"
)

const (
	defaultDensityRamp = " .;coPO?#@"
	asciiDensityRamp   = " .:-=+*#%@"
	blocksDensityRamp  = " ░▒▓█"
)

type RampMode string

const (
	RampModeDefault RampMode = "default"
	RampModeASCII   RampMode = "ascii"
	RampModeBlocks  RampMode = "blocks"
)

type ColorMode string

const (
	ColorModeSource ColorMode = "source"
	ColorModeNone   ColorMode = "none"
)

type EdgeMode string

const (
	EdgeModeDefault EdgeMode = "default"
	EdgeModeNone    EdgeMode = "none"
)

type Recipe struct {
	Name      string
	RampMode  RampMode
	ColorMode ColorMode
	EdgeMode  EdgeMode
}

var recipePresets = []Recipe{
	{Name: "default", RampMode: RampModeDefault, ColorMode: ColorModeSource, EdgeMode: EdgeModeDefault},
	{Name: "mono", RampMode: RampModeDefault, ColorMode: ColorModeNone, EdgeMode: EdgeModeDefault},
	{Name: "ascii", RampMode: RampModeASCII, ColorMode: ColorModeNone, EdgeMode: EdgeModeNone},
	{Name: "blocks", RampMode: RampModeBlocks, ColorMode: ColorModeSource, EdgeMode: EdgeModeNone},
}

func DefaultRecipe() Recipe {
	return recipePresets[0]
}

func RecipePresetNames() []string {
	names := make([]string, 0, len(recipePresets))
	for _, recipe := range recipePresets {
		names = append(names, recipe.Name)
	}
	return names
}

func RecipeByName(name string) (Recipe, error) {
	for _, recipe := range recipePresets {
		if recipe.Name == name {
			return recipe, nil
		}
	}
	return Recipe{}, fmt.Errorf("unsupported recipe %q; supported recipes: %s", name, strings.Join(RecipePresetNames(), ", "))
}

func MustRecipeByName(name string) Recipe {
	recipe, err := RecipeByName(name)
	if err != nil {
		panic(err)
	}
	return recipe
}

type Renderer struct {
	Recipe Recipe
}

type DefaultRenderer struct{}

func (DefaultRenderer) Render(frame RGBFrame, grid Grid) CharacterFrame {
	return NewRenderer(DefaultRecipe()).Render(frame, grid)
}

func NewRenderer(recipe Recipe) Renderer {
	if recipe.Name == "" {
		recipe = DefaultRecipe()
	}
	return Renderer{Recipe: recipe}
}

func (r Renderer) Render(frame RGBFrame, grid Grid) CharacterFrame {
	cells := make([]Cell, grid.Cols*grid.Rows)
	for gy := 0; gy < grid.Rows; gy++ {
		for gx := 0; gx < grid.Cols; gx++ {
			sx := gx * frame.Width / grid.Cols
			sy := gy * frame.Height / grid.Rows
			red, green, blue := frame.RGBAt(sx, sy)
			luma := luminance(red, green, blue)
			ch := densityCharForRamp(luma, rampForMode(r.Recipe.RampMode))
			if r.Recipe.EdgeMode == EdgeModeDefault {
				if edge, ok := edgeGlyph(frame, sx, sy); ok {
					ch = edge
				}
			}
			cell := Cell{Ch: ch}
			if r.Recipe.ColorMode == ColorModeSource {
				cell.HasColor = true
				cell.R = red
				cell.G = green
				cell.B = blue
			}
			cells[gy*grid.Cols+gx] = cell
		}
	}
	return CharacterFrame{Width: grid.Cols, Height: grid.Rows, Cells: cells}
}

func luminance(r uint8, g uint8, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

func densityChar(luma float64) rune {
	return densityCharForRamp(luma, defaultDensityRamp)
}

func densityCharForRamp(luma float64, ramp string) rune {
	// Bias mid-tones toward clearer terminal contrast while preserving the golden ramp anchors.
	normalized := luma / 255.0
	if normalized < 0.5 {
		normalized = 0.5 * math.Pow(normalized*2, 4)
	} else {
		normalized = 1 - 0.5*math.Pow((1-normalized)*2, 4)
	}
	runes := []rune(ramp)
	index := int(math.Round(normalized * float64(len(runes)-1)))
	if index < 0 {
		index = 0
	}
	if index >= len(runes) {
		index = len(runes) - 1
	}
	return runes[index]
}

func rampForMode(mode RampMode) string {
	switch mode {
	case RampModeASCII:
		return asciiDensityRamp
	case RampModeBlocks:
		return blocksDensityRamp
	default:
		return defaultDensityRamp
	}
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
