package fonts

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed Mx437_IBM_BIOS.ttf
var mx437IBMBIOS []byte

func DefaultFace() (font.Face, error) {
	parsed, err := opentype.Parse(mx437IBMBIOS)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    8,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
