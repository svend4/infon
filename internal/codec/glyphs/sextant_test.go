package glyphs

import "testing"

func TestSextantSpecialPatterns(t *testing.T) {
	if SextantRune(0) != ' ' {
		t.Errorf("pattern 0 = %q, want space", SextantRune(0))
	}
	if SextantRune(63) != '█' {
		t.Errorf("pattern 63 = %q, want █ (full)", SextantRune(63))
	}
	// Pattern 21 = bits 0,2,4 set = entire left column = ▌ (left half block).
	if SextantRune(21) != '▌' {
		t.Errorf("pattern 21 = %q, want ▌", SextantRune(21))
	}
	// Pattern 42 = bits 1,3,5 set = entire right column = ▐.
	if SextantRune(42) != '▐' {
		t.Errorf("pattern 42 = %q, want ▐", SextantRune(42))
	}
}

func TestSextantBlockRange(t *testing.T) {
	// Every pattern except the 4 special ones must land in U+1FB00..U+1FB3B.
	special := map[uint8]bool{0: true, 21: true, 42: true, 63: true}
	for p := 0; p < 64; p++ {
		r := SextantRune(uint8(p))
		if special[uint8(p)] {
			continue
		}
		if r < 0x1FB00 || r > 0x1FB3B {
			t.Errorf("pattern %d -> U+%04X, outside the sextant block", p, r)
		}
	}
}

func TestSextantRunesAreUnique(t *testing.T) {
	// No two distinct patterns may map to the same glyph (a wrong table would
	// collide). All 64 must be distinct.
	seen := map[rune]int{}
	for p := 0; p < 64; p++ {
		r := SextantRune(uint8(p))
		if prev, ok := seen[r]; ok {
			t.Fatalf("patterns %d and %d both map to U+%04X", prev, p, r)
		}
		seen[r] = p
	}
	if len(seen) != 64 {
		t.Errorf("expected 64 distinct glyphs, got %d", len(seen))
	}
}

func TestSextantFromBits(t *testing.T) {
	if SextantFromBits(false, false, false, false, false, false) != 0 {
		t.Error("all-false should be pattern 0")
	}
	if SextantFromBits(true, true, true, true, true, true) != 63 {
		t.Error("all-true should be pattern 63")
	}
	// Left column only (TL, ML, BL) = bits 0,2,4 = 21.
	if got := SextantFromBits(true, false, true, false, true, false); got != 21 {
		t.Errorf("left column = %d, want 21", got)
	}
	// Round-trip: bits -> pattern -> rune is the left-half block.
	if SextantRune(SextantFromBits(true, false, true, false, true, false)) != '▌' {
		t.Error("left column should render as ▌")
	}
}
