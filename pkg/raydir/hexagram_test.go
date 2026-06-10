package raydir

import (
	"strings"
	"testing"
)

// The six-bit number is a bijection with the lines.
func TestHexagramNumber(t *testing.T) {
	if (Hexagram{}).Number() != 0 {
		t.Error("all-yin should be number 0")
	}
	all := Hexagram{Lines: [6]bool{true, true, true, true, true, true}}
	if all.Number() != 63 {
		t.Errorf("all-yang should be 63, got %d", all.Number())
	}
	seen := map[int]bool{}
	for n := 0; n < 64; n++ {
		var h Hexagram
		for i := 0; i < 6; i++ {
			h.Lines[i] = n&(1<<i) != 0
		}
		if h.Number() != n {
			t.Fatalf("round-trip failed for %d -> %d", n, h.Number())
		}
		seen[h.Number()] = true
	}
	if len(seen) != 64 {
		t.Errorf("expected 64 distinct hexagrams, got %d", len(seen))
	}
}

// The name and prompt come from the two trigrams.
func TestHexagramNameAndPrompt(t *testing.T) {
	// lower = Mountain (001), upper = Water (010)
	h := Hexagram{Lines: [6]bool{true, false, false, false, true, false}}
	if h.Name() != "Water over Mountain" {
		t.Errorf("name should pair the trigrams, got %q", h.Name())
	}
	p := h.Prompt()
	if !strings.Contains(p, "water") || !strings.Contains(p, "mountains") {
		t.Errorf("prompt should carry both trigram themes, got %q", p)
	}
}

// Casting is deterministic in its seed; different seeds (usually) differ.
func TestCastHexagram(t *testing.T) {
	a := CastHexagram(7)
	b := CastHexagram(7)
	if a != b {
		t.Error("the same seed should cast the same hexagram")
	}
	differ := false
	for s := int64(0); s < 20; s++ {
		if CastHexagram(s) != a {
			differ = true
			break
		}
	}
	if !differ {
		t.Error("different seeds should be able to cast different hexagrams")
	}
	// the hexagram's own seed is stable
	s1, s2 := a.Seed(), a.Seed()
	if s1 != s2 {
		t.Error("Seed should be deterministic")
	}
}

// Parsing reads 1/0 and y/n and rejects bad input.
func TestParseHexagram(t *testing.T) {
	h, ok := ParseHexagram("101010")
	if !ok {
		t.Fatal("a valid 6-char string should parse")
	}
	if h.Lines[0] != true || h.Lines[1] != false || h.Lines[5] != false {
		t.Errorf("parsed lines wrong: %+v", h.Lines)
	}
	if h2, ok := ParseHexagram("yynnyn"); !ok || h2.Lines[0] != true {
		t.Error("y/n form should parse")
	}
	if _, ok := ParseHexagram("10101"); ok {
		t.Error("five chars should not parse")
	}
	if _, ok := ParseHexagram("1010x0"); ok {
		t.Error("an invalid char should not parse")
	}
}
