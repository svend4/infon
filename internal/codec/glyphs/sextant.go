package glyphs

// Sextants (roadmap A2): Unicode 13 "Symbols for Legacy Computing" (U+1FB00…)
// provide every 2×3 block-fill pattern. A character cell is divided into 6
// sub-cells:
//
//	bit 0: top-left      bit 1: top-right
//	bit 2: middle-left   bit 3: middle-right
//	bit 4: bottom-left   bit 5: bottom-right
//
// 64 combinations = 6 bits -> 6 sub-pixels/cell (vs 4 for quadrants), a 1.5×
// density gain. The U+1FB00 block holds 60 glyphs; the other 4 patterns reuse
// existing block characters: 0=space, 21=▌ (left column), 42=▐ (right column),
// 63=█ (full). The table below is generated from the official Unicode names and
// verified, so it is correct rather than assuming contiguity.
var sextantRunes = [64]rune{
	' ', 0x1FB00, 0x1FB01, 0x1FB02, 0x1FB03, 0x1FB04, 0x1FB05, 0x1FB06,
	0x1FB07, 0x1FB08, 0x1FB09, 0x1FB0A, 0x1FB0B, 0x1FB0C, 0x1FB0D, 0x1FB0E,
	0x1FB0F, 0x1FB10, 0x1FB11, 0x1FB12, 0x1FB13, '▌', 0x1FB14, 0x1FB15,
	0x1FB16, 0x1FB17, 0x1FB18, 0x1FB19, 0x1FB1A, 0x1FB1B, 0x1FB1C, 0x1FB1D,
	0x1FB1E, 0x1FB1F, 0x1FB20, 0x1FB21, 0x1FB22, 0x1FB23, 0x1FB24, 0x1FB25,
	0x1FB26, 0x1FB27, '▐', 0x1FB28, 0x1FB29, 0x1FB2A, 0x1FB2B, 0x1FB2C,
	0x1FB2D, 0x1FB2E, 0x1FB2F, 0x1FB30, 0x1FB31, 0x1FB32, 0x1FB33, 0x1FB34,
	0x1FB35, 0x1FB36, 0x1FB37, 0x1FB38, 0x1FB39, 0x1FB3A, 0x1FB3B, '█',
}

// SextantRune returns the Unicode glyph for a 6-bit sextant pattern (0..63).
func SextantRune(pattern uint8) rune {
	return sextantRunes[pattern&0x3F]
}

// SextantFromBits builds the 6-bit pattern from the six sub-cells.
func SextantFromBits(tl, tr, ml, mr, bl, br bool) uint8 {
	var p uint8
	if tl {
		p |= 1 << 0
	}
	if tr {
		p |= 1 << 1
	}
	if ml {
		p |= 1 << 2
	}
	if mr {
		p |= 1 << 3
	}
	if bl {
		p |= 1 << 4
	}
	if br {
		p |= 1 << 5
	}
	return p
}
