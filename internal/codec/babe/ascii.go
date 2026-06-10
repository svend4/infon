package babe

import (
	"image"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// asciiRamp maps brightness (dark -> light) to glyphs of increasing ink
// coverage. This is the classic luminance-ramp ("ASCII art") technique — the
// signature look of monochrome console 3-D renderers — exposed here as a
// first-class render mode. The ramp is ordered so each step adds roughly more
// foreground ink, which is what makes a gradient read as a smooth shade.
var asciiRamp = []rune(" .,:;-=+ox*O0X#%@")

// ImageToFrameASCII renders the image as a luminance ramp: one glyph per cell,
// chosen from asciiRamp by the cell's average brightness. Two things make it
// better than a naive ramp: each cell area-averages its whole source region
// (not a single point sample, so thin features don't flicker or vanish), and the
// glyph is drawn in the cell's average color on black — so it reads in color on a
// capable terminal and as pure ASCII art on a monochrome one. With one byte of
// glyph per cell and no sub-cell color pairs, it is also the lowest-bandwidth and
// most portable mode (works where Unicode half-blocks/sextants do not).
func ImageToFrameASCII(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)
	if targetWidth <= 0 || targetHeight <= 0 {
		return frame
	}
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return frame
	}

	for ty := 0; ty < targetHeight; ty++ {
		// Source row span covered by this cell (integer box filter).
		y0 := bounds.Min.Y + ty*srcH/targetHeight
		y1 := bounds.Min.Y + (ty+1)*srcH/targetHeight
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for tx := 0; tx < targetWidth; tx++ {
			x0 := bounds.Min.X + tx*srcW/targetWidth
			x1 := bounds.Min.X + (tx+1)*srcW/targetWidth
			if x1 <= x0 {
				x1 = x0 + 1
			}

			var sr, sg, sb, n uint32
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, b, _ := img.At(x, y).RGBA()
					sr += r >> 8
					sg += g >> 8
					sb += b >> 8
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			avg := color.RGB{R: uint8(sr / n), G: uint8(sg / n), B: uint8(sb / n)}
			lum := avg.Luminance()
			// Nearest ramp level (round, not truncate) so both ends are reachable.
			idx := int(float64(lum)/255.0*float64(len(asciiRamp)-1) + 0.5)
			if idx < 0 {
				idx = 0
			} else if idx >= len(asciiRamp) {
				idx = len(asciiRamp) - 1
			}
			frame.SetBlock(tx, ty, asciiRamp[idx], avg, color.Black)
		}
	}
	return frame
}
