package babe

import (
	"image"

	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// ImageToFrameSextant renders using 2×3 sextant glyphs (roadmap A2): each cell
// samples six sub-pixels, splits them into two perceptual color groups (OKLab
// lightness), and picks the sextant glyph whose set bits match the lighter
// group. This yields 6 sub-pixels/cell (cols×2 wide, rows×3 tall) — denser than
// the 2×2 quadrant path — while still using just fg+bg per cell.
//
// Requires a terminal/font with Unicode 13 legacy-computing glyphs; callers
// should gate this on terminal.Capability.Unicode13 and fall back otherwise.
func ImageToFrameSextant(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return frame
	}

	// Each cell covers 2 sub-pixels wide × 3 tall.
	scaleX := float64(srcWidth) / float64(targetWidth*2)
	scaleY := float64(srcHeight) / float64(targetHeight*3)

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			// Sub-cell order: TL,TR,ML,MR,BL,BR.
			var px [6]color.RGB
			offsets := [6][2]int{
				{0, 0}, {1, 0}, // top row
				{0, 1}, {1, 1}, // middle row
				{0, 2}, {1, 2}, // bottom row
			}
			for i, off := range offsets {
				sx := int((float64(tx*2+off[0]) + 0.5) * scaleX)
				sy := int((float64(ty*3+off[1]) + 0.5) * scaleY)
				px[i] = sampleRGB(img, sx, sy, bounds)
			}
			glyph, fg, bg := encodeSextant(px)
			frame.SetBlock(tx, ty, glyph, fg, bg)
		}
	}
	return frame
}

// encodeSextant splits six sub-pixels into two OKLab-lightness groups, sets the
// pattern bit for each sub-pixel in the lighter (foreground) group, and returns
// the matching sextant glyph with perceptual average colors.
func encodeSextant(px [6]color.RGB) (rune, color.RGB, color.RGB) {
	// Use the darkest and lightest sub-pixels as the two cluster centers and
	// assign each sub-pixel to the nearer one. This is more robust than a mean
	// threshold (which mis-splits a uniform block due to float rounding) and
	// matches simple k=2 clustering: a uniform block has min==max, so every
	// sub-pixel joins the foreground -> the full glyph.
	ls := [6]float64{}
	minL, maxL := ls[0], ls[0]
	for i, c := range px {
		ls[i] = c.ToOKLab().L
		if i == 0 || ls[i] < minL {
			minL = ls[i]
		}
		if i == 0 || ls[i] > maxL {
			maxL = ls[i]
		}
	}

	var light, dark []color.RGB
	var bits [6]bool
	for i := range px {
		// Nearer to the light center (ties -> light, so uniform -> full).
		if maxL-ls[i] <= ls[i]-minL {
			bits[i] = true
			light = append(light, px[i])
		} else {
			dark = append(dark, px[i])
		}
	}

	pattern := glyphs.SextantFromBits(bits[0], bits[1], bits[2], bits[3], bits[4], bits[5])
	fg := averageColorOKLab(light)
	bg := averageColorOKLab(dark)
	return glyphs.SextantRune(pattern), fg, bg
}
