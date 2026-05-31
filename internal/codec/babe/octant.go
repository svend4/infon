package babe

import (
	"image"

	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// ImageToFrameOctant renders using 2×4 octant glyphs (roadmap A2 extended): each
// cell samples eight sub-pixels, clusters them into two perceptual color groups
// (nearest OKLab-lightness center, robust on uniform blocks), and picks the
// octant glyph whose set bits match the lighter group. This is the densest glyph
// mode — 8 sub-pixels/cell (cols×2 wide, rows×4 tall) — needing Unicode 16
// support; gate via terminal.Capability and fall back otherwise.
func ImageToFrameOctant(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return frame
	}

	scaleX := float64(srcWidth) / float64(targetWidth*2)
	scaleY := float64(srcHeight) / float64(targetHeight*4)

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			// Sub-cell order matches OctantFromBits: TL,TR,UML,UMR,LML,LMR,BL,BR.
			var px [8]color.RGB
			offsets := [8][2]int{
				{0, 0}, {1, 0}, // top
				{0, 1}, {1, 1}, // upper-mid
				{0, 2}, {1, 2}, // lower-mid
				{0, 3}, {1, 3}, // bottom
			}
			for i, off := range offsets {
				sx := int((float64(tx*2+off[0]) + 0.5) * scaleX)
				sy := int((float64(ty*4+off[1]) + 0.5) * scaleY)
				px[i] = sampleRGB(img, sx, sy, bounds)
			}
			glyph, fg, bg := encodeOctant(px)
			frame.SetBlock(tx, ty, glyph, fg, bg)
		}
	}
	return frame
}

// encodeOctant clusters eight sub-pixels by nearest OKLab-lightness center, sets
// the pattern bit for each lighter-group sub-pixel, and returns the octant glyph
// with perceptual average colors.
func encodeOctant(px [8]color.RGB) (rune, color.RGB, color.RGB) {
	var lab [8]color.OKLab
	var minL, maxL float64
	for i := range px {
		lab[i] = px[i].ToOKLab()
		if i == 0 || lab[i].L < minL {
			minL = lab[i].L
		}
		if i == 0 || lab[i].L > maxL {
			maxL = lab[i].L
		}
	}

	var l0, a0, b0, l1, a1, b1 float64
	var n0, n1 int
	var bits [8]bool
	for i := range px {
		// Nearer to the light center (ties -> light, so uniform -> full).
		if maxL-lab[i].L <= lab[i].L-minL {
			bits[i] = true
			l1 += lab[i].L
			a1 += lab[i].A
			b1 += lab[i].B
			n1++
		} else {
			l0 += lab[i].L
			a0 += lab[i].A
			b0 += lab[i].B
			n0++
		}
	}

	pattern := glyphs.OctantFromBits(bits[0], bits[1], bits[2], bits[3], bits[4], bits[5], bits[6], bits[7])
	fg := meanOKLab(l1, a1, b1, n1)
	bg := meanOKLab(l0, a0, b0, n0)
	return glyphs.OctantRune(pattern), fg, bg
}
