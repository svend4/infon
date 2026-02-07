package glyphs

import "testing"

func TestGetGlyph(t *testing.T) {
	tests := []struct {
		pattern uint8
		want    rune
	}{
		{0b0000, ' '},
		{0b1111, '█'},
		{0b0011, '▄'},
		{0b1100, '▀'},
		{0b0101, '▐'},
		{0b1010, '▌'},
	}

	for _, tt := range tests {
		got := GetGlyph(tt.pattern)
		if got.Char != tt.want {
			t.Errorf("GetGlyph(%04b) = %c, want %c", tt.pattern, got.Char, tt.want)
		}
	}
}

func TestGetGlyphFromBits(t *testing.T) {
	// Test all 16 combinations
	for pattern := uint8(0); pattern < 16; pattern++ {
		tl := (pattern & 0b1000) != 0
		tr := (pattern & 0b0100) != 0
		bl := (pattern & 0b0010) != 0
		br := (pattern & 0b0001) != 0

		got := GetGlyphFromBits(tl, tr, bl, br)
		want := QuadrantGlyphs[pattern]

		if got.Char != want.Char {
			t.Errorf("GetGlyphFromBits(%v,%v,%v,%v) = %c, want %c",
				tl, tr, bl, br, got.Char, want.Char)
		}
	}
}

func TestBraillePattern(t *testing.T) {
	// Test empty pattern
	dots := [8]bool{}
	got := BraillePattern(dots)
	if got != 0x2800 {
		t.Errorf("BraillePattern(empty) = U+%04X, want U+2800", got)
	}

	// Test all dots
	allDots := [8]bool{true, true, true, true, true, true, true, true}
	got = BraillePattern(allDots)
	if got != 0x28FF {
		t.Errorf("BraillePattern(all) = U+%04X, want U+28FF", got)
	}
}
