package pseudo

import (
	"image"
	stdcolor "image/color"

	"github.com/svend4/infon/pkg/terminal"
)

// RasterizeFrame turns a terminal.Frame (grid of colored cells) into an RGBA
// image of the requested pixel size: a drawn glyph takes its foreground color,
// blank cells take their background. This lets cell-native formats (glyphs,
// sigils, vector, sketch) flow through the BABE -> terminal video path.
func RasterizeFrame(f *terminal.Frame, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if f == nil || f.Width == 0 || f.Height == 0 {
		return img
	}
	for y := 0; y < h; y++ {
		cy := y * f.Height / h
		for x := 0; x < w; x++ {
			cx := x * f.Width / w
			blk := f.Blocks[cy][cx]
			c := blk.Bg
			if blk.Glyph != ' ' {
				c = blk.Fg
			}
			img.SetRGBA(x, y, stdcolor.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}
	return img
}
