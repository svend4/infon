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

	// Precompute OKLab for each valid sub-pixel ONCE (the conversion is the cost).
	var lab [4]color.OKLab
	for i := 0; i < 4; i++ {
		if valid[i] {
			lab[i] = pixels[i].ToOKLab()
		}
	}

	bestErr := -1.0
	var bestPattern int
	var bestFgLab, bestBgLab color.OKLab

	// Each pattern 0..15: bit (3-i) set means sub-pixel i is foreground. We work
	// entirely in OKLab from the cached lab[]: the group MEAN in OKLab is exactly
	// the color we'd render, so no slices, no RGB round-trips, no re-conversion.
	for pattern := 0; pattern < 16; pattern++ {
		var fl, fa, fb, bl, ba, bb float64
		var fn, bn int
		for i := 0; i < 4; i++ {
			if !valid[i] {
				continue
			}
			if pattern&(1<<uint(3-i)) != 0 {
				fl += lab[i].L
				fa += lab[i].A
				fb += lab[i].B
				fn++
			} else {
				bl += lab[i].L
				ba += lab[i].A
				bb += lab[i].B
				bn++
			}
		}
		var fgLab, bgLab color.OKLab
		if fn > 0 {
			fgLab = color.OKLab{L: fl / float64(fn), A: fa / float64(fn), B: fb / float64(fn)}
		}
		if bn > 0 {
			bgLab = color.OKLab{L: bl / float64(bn), A: ba / float64(bn), B: bb / float64(bn)}
		}

		// Reconstruction error: each sub-pixel scored against its group mean.
		errSum := 0.0
		for i := 0; i < 4; i++ {
			if !valid[i] {
				continue
			}
			if pattern&(1<<uint(3-i)) != 0 {
				errSum += labDist2(lab[i], fgLab)
			} else {
				errSum += labDist2(lab[i], bgLab)
			}
		}

		if bestErr < 0 || errSum < bestErr {
			bestErr = errSum
			bestPattern = pattern
			bestFgLab = fgLab
			bestBgLab = bgLab
		}
	}

	// Convert the two winning OKLab means back to RGB just ONCE (not per pattern).
	return glyphFromPattern(bestPattern), bestFgLab.ToRGB(), bestBgLab.ToRGB()
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
