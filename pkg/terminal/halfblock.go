package terminal

import (
	"image"

	"github.com/svend4/infon/pkg/color"
)

// HalfBlock renders an image into a cols x rows Frame using the upper-half-block
// glyph '▀': each cell stacks two image rows, the foreground colour painting the
// top pixel and the background the bottom. This doubles the vertical resolution
// and reproduces both colours exactly in 24-bit, which makes it the best mode for
// smooth, photographic content such as a ray-traced frame. The source is sampled
// at cell centres (nearest-neighbour), so any image size maps onto the grid.
func HalfBlock(img image.Image, cols, rows int) *Frame {
	f := NewFrame(cols, rows)
	if cols <= 0 || rows <= 0 {
		return f
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 {
		return f
	}
	at := func(cx, py int) color.RGB {
		sx := b.Min.X + cx*iw/cols
		sy := b.Min.Y + py*ih/(rows*2)
		r, g, bl, _ := img.At(sx, sy).RGBA()
		return color.RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8)}
	}
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			f.SetBlock(x, y, '▀', at(x, y*2), at(x, y*2+1))
		}
	}
	return f
}
