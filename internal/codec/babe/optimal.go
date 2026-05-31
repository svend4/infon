package babe

import (
	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
)

// EncodeBlockOptimal (roadmap C2) finds the glyph + two colors that minimize the
// PERCEPTUAL reconstruction error of a 2x2 block, instead of guessing a single
// luminance split. It evaluates all 16 quadrant partitions: for each, the two
// sub-pixel groups are averaged (in OKLab) into a foreground/background color,
// and the block is scored by summed OKLab distance between each original
// sub-pixel and the color it would render as. The lowest-error partition wins.
//
// This is the "learned encoder" idea reduced to its exact optimum: for a 2x2
// block the search space is tiny (16 partitions), so we can solve it directly
// rather than approximate. It is pure Go, deterministic, and has no runtime
// model — every frame from every source gets better colors for ~16x the encode
// cost of the luminance path (still microseconds per cell).
func EncodeBlockOptimal(pixels [4]color.RGB, valid [4]bool) (rune, color.RGB, color.RGB) {
	anyValid := false
	for _, v := range valid {
		if v {
			anyValid = true
			break
		}
	}
	if !anyValid {
		return ' ', color.Black, color.Black
	}

	// Precompute OKLab for each valid sub-pixel.
	var lab [4]color.OKLab
	for i := 0; i < 4; i++ {
		if valid[i] {
			lab[i] = pixels[i].ToOKLab()
		}
	}

	bestErr := -1.0
	var bestGlyph rune = ' '
	bestFg, bestBg := color.Black, color.Black

	// Each pattern 0..15: bit i (TL,TR,BL,BR) means sub-pixel i is foreground.
	for pattern := 0; pattern < 16; pattern++ {
		var fgGroup, bgGroup []color.RGB
		for i := 0; i < 4; i++ {
			if !valid[i] {
				continue
			}
			if pattern&(1<<uint(3-i)) != 0 { // bit3=TL .. bit0=BR (matches GetGlyphFromBits)
				fgGroup = append(fgGroup, pixels[i])
			} else {
				bgGroup = append(bgGroup, pixels[i])
			}
		}
		fg := averageColorOKLab(fgGroup)
		bg := averageColorOKLab(bgGroup)
		fgLab := fg.ToOKLab()
		bgLab := bg.ToOKLab()

		// Reconstruction error: each sub-pixel rendered as fg (if in fg group)
		// or bg, scored by squared OKLab distance.
		errSum := 0.0
		for i := 0; i < 4; i++ {
			if !valid[i] {
				continue
			}
			var rendered color.OKLab
			if pattern&(1<<uint(3-i)) != 0 {
				rendered = fgLab
			} else {
				rendered = bgLab
			}
			errSum += labDist2(lab[i], rendered)
		}

		if bestErr < 0 || errSum < bestErr {
			bestErr = errSum
			bestGlyph = glyphFromPattern(pattern)
			bestFg = fg
			bestBg = bg
		}
	}

	return bestGlyph, bestFg, bestBg
}

// glyphFromPattern maps a 4-bit pattern (bit3=TL,bit2=TR,bit1=BL,bit0=BR) to its
// quadrant glyph, matching glyphs.GetGlyphFromBits' bit order.
func glyphFromPattern(pattern int) rune {
	tl := pattern&0b1000 != 0
	tr := pattern&0b0100 != 0
	bl := pattern&0b0010 != 0
	br := pattern&0b0001 != 0
	return glyphs.GetGlyphFromBits(tl, tr, bl, br).Char
}

func labDist2(a, b color.OKLab) float64 {
	dl := a.L - b.L
	da := a.A - b.A
	db := a.B - b.B
	return dl*dl + da*da + db*db
}
