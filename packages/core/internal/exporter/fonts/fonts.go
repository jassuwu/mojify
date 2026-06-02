package fonts

import (
	_ "embed"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed Mx437_IBM_BIOS.ttf
var mx437IBMBIOS []byte

var (
	mx437Once sync.Once
	mx437Font *opentype.Font
	mx437Err  error
)

func DefaultFace() (font.Face, error) {
	mx437Once.Do(func() {
		mx437Font, mx437Err = opentype.Parse(mx437IBMBIOS)
	})
	if mx437Err != nil {
		return nil, mx437Err
	}
	return opentype.NewFace(mx437Font, &opentype.FaceOptions{
		Size:    8,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
