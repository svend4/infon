package glyphs

import "testing"

func TestOctantSpecialPatterns(t *testing.T) {
	cases := map[uint8]rune{
		0x00: ' ', // empty
		0xFF: '█', // full
		0x55: '▌', // left column (bits 0,2,4,6)
		0xAA: '▐', // right column (bits 1,3,5,7)
		0x0F: '▀', // top two rows (bits 0,1,2,3)
		0xF0: '▄', // bottom two rows (bits 4,5,6,7)
	}
	for pat, want := range cases {
		if got := OctantRune(pat); got != want {
			t.Errorf("OctantRune(0x%02X) = %q, want %q", pat, got, want)
		}
	}
}

func TestOctantBlockRange(t *testing.T) {
	// The vast majority of patterns must be real octant glyphs in U+1CD00..U+1CDFB.
	inBlock := 0
	for p := 0; p < 256; p++ {
		r := OctantRune(uint8(p))
		if r >= 0x1CD00 && r <= 0x1CDFB {
			inBlock++
		}
	}
	if inBlock < 220 {
		t.Errorf("only %d patterns map into the octant block, expected >=220", inBlock)
	}
}

func TestOctantFromBits(t *testing.T) {
	if OctantFromBits(false, false, false, false, false, false, false, false) != 0 {
		t.Error("all-false should be 0")
	}
	if OctantFromBits(true, true, true, true, true, true, true, true) != 0xFF {
		t.Error("all-true should be 0xFF")
	}
	// Left column = TL,UML,LML,BL = bits 0,2,4,6 = 0x55.
	if got := OctantFromBits(true, false, true, false, true, false, true, false); got != 0x55 {
		t.Errorf("left column = 0x%02X, want 0x55", got)
	}
	// Round-trip: left column renders as the left half block.
	if OctantRune(OctantFromBits(true, false, true, false, true, false, true, false)) != '▌' {
		t.Error("left column should render as ▌")
	}
}

func TestOctantTableComplete(t *testing.T) {
	// Every pattern must map to a non-zero rune (space is 0x20, still non-zero),
	// so there are no accidental gaps in the 256-entry table.
	for p := 0; p < 256; p++ {
		if OctantRune(uint8(p)) == 0 {
			t.Fatalf("pattern 0x%02X maps to NUL — table gap", p)
		}
	}
}
