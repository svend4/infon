package glyphset

import (
	"image"
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func TestFontRendersLetters(t *testing.T) {
	if _, ok := fontMask('A'); !ok {
		t.Fatal("A missing from font")
	}
	if _, ok := fontMask('a'); !ok {
		t.Fatal("lowercase a should fold to A")
	}
	f := terminal.NewFrame(3, 1)
	f.DrawText(0, 0, "ABC", color.RGB{R: 255, G: 255, B: 255}, color.RGB{R: 0, G: 0, B: 0})
	img := Rasterize(f, 14).(*image.RGBA)
	white := 0
	b := img.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if img.Pix[(y*b.Dx()+x)*4] > 200 {
				white++
			}
		}
	}
	if white == 0 {
		t.Fatal("no letter pixels were drawn (font not used)")
	}
}
