package glyphs

// Glyph represents a Unicode quadrant block character
type Glyph struct {
	Char    rune   // Unicode character
	Pattern uint8  // 4-bit pattern (0-15)
	Bits    [4]bool // TL, TR, BL, BR
}

// QuadrantGlyphs defines all 16 quadrant block patterns
// Pattern encoding: bit0=TL, bit1=TR, bit2=BL, bit3=BR
var QuadrantGlyphs = [16]Glyph{
	{Char: ' ', Pattern: 0b0000, Bits: [4]bool{false, false, false, false}}, // 0: empty
	{Char: '▗', Pattern: 0b0001, Bits: [4]bool{false, false, false, true}},  // 1: BR only
	{Char: '▖', Pattern: 0b0010, Bits: [4]bool{false, false, true, false}},  // 2: BL only
	{Char: '▄', Pattern: 0b0011, Bits: [4]bool{false, false, true, true}},   // 3: bottom
	{Char: '▝', Pattern: 0b0100, Bits: [4]bool{false, true, false, false}},  // 4: TR only
	{Char: '▐', Pattern: 0b0101, Bits: [4]bool{false, true, false, true}},   // 5: right
	{Char: '▞', Pattern: 0b0110, Bits: [4]bool{false, true, true, false}},   // 6: diagonal \
	{Char: '▟', Pattern: 0b0111, Bits: [4]bool{false, true, true, true}},    // 7: 3/4 no TL
	{Char: '▘', Pattern: 0b1000, Bits: [4]bool{true, false, false, false}},  // 8: TL only
	{Char: '▚', Pattern: 0b1001, Bits: [4]bool{true, false, false, true}},   // 9: diagonal /
	{Char: '▌', Pattern: 0b1010, Bits: [4]bool{true, false, true, false}},   // 10: left
	{Char: '▙', Pattern: 0b1011, Bits: [4]bool{true, false, true, true}},    // 11: 3/4 no TR
	{Char: '▀', Pattern: 0b1100, Bits: [4]bool{true, true, false, false}},   // 12: top
	{Char: '▜', Pattern: 0b1101, Bits: [4]bool{true, true, false, true}},    // 13: 3/4 no BL
	{Char: '▛', Pattern: 0b1110, Bits: [4]bool{true, true, true, false}},    // 14: 3/4 no BR
	{Char: '█', Pattern: 0b1111, Bits: [4]bool{true, true, true, true}},     // 15: full
}

// GetGlyph returns the glyph for a given pattern (0-15)
func GetGlyph(pattern uint8) Glyph {
	if pattern > 15 {
		return QuadrantGlyphs[0] // return empty on invalid
	}
	return QuadrantGlyphs[pattern]
}

// GetGlyphFromBits returns the glyph matching the given bit pattern
func GetGlyphFromBits(tl, tr, bl, br bool) Glyph {
	pattern := uint8(0)
	if tl {
		pattern |= 0b1000
	}
	if tr {
		pattern |= 0b0100
	}
	if bl {
		pattern |= 0b0010
	}
	if br {
		pattern |= 0b0001
	}
	return QuadrantGlyphs[pattern]
}

// HalfBlockGlyphs for simpler 2-level rendering (fallback)
var HalfBlockGlyphs = struct {
	Upper rune // ▀ upper half block
	Lower rune // ▄ lower half block
	Full  rune // █ full block
	Empty rune // space
}{
	Upper: '▀',
	Lower: '▄',
	Full:  '█',
	Empty: ' ',
}

// BraillePattern generates Braille Unicode codepoint from 8 dots
// dots[0-7] correspond to Braille dots 1-8
func BraillePattern(dots [8]bool) rune {
	base := rune(0x2800)
	val := 0
	// Braille dot positions: 1,2,3,7 (left), 4,5,6,8 (right)
	if dots[0] {
		val += 1
	} // dot 1
	if dots[1] {
		val += 2
	} // dot 2
	if dots[2] {
		val += 4
	} // dot 3
	if dots[3] {
		val += 64
	} // dot 4
	if dots[4] {
		val += 8
	} // dot 5
	if dots[5] {
		val += 16
	} // dot 6
	if dots[6] {
		val += 32
	} // dot 7
	if dots[7] {
		val += 128
	} // dot 8
	return base + rune(val)
}
