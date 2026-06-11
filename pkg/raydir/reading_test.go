package raydir

import (
	"strings"
	"testing"
)

func TestCastReadingDeterministic(t *testing.T) {
	a, b := CastReading(7), CastReading(7)
	if a != b {
		t.Fatal("same seed should cast the same reading")
	}
}

func TestRelatingFlipsOnlyChangingLines(t *testing.T) {
	r := CastReading(3)
	rel := r.Relating()
	for i := 0; i < 6; i++ {
		want := r.Primary.Lines[i] != r.Changing[i] // XOR: flipped iff changing
		if rel.Lines[i] != want {
			t.Errorf("line %d: relating=%v primary=%v changing=%v", i, rel.Lines[i], r.Primary.Lines[i], r.Changing[i])
		}
	}
}

func TestStableReadingRelatesToItself(t *testing.T) {
	r := Reading{Primary: HexagramFromNumber(0b101010)} // no changing lines
	if !r.Stable() || r.Mask() != 0 {
		t.Fatal("a reading with no changing lines is stable")
	}
	if r.Relating().Number() != r.Primary.Number() {
		t.Error("a stable reading relates to itself")
	}
}

func TestReadingsHaveVarietyOfChange(t *testing.T) {
	stable, changing := 0, 0
	for s := int64(0); s < 60; s++ {
		if CastReading(s).Stable() {
			stable++
		} else {
			changing++
		}
	}
	if stable == 0 || changing == 0 {
		t.Errorf("expected a mix of stable and changing readings, got stable=%d changing=%d", stable, changing)
	}
}

func TestReadingStringForms(t *testing.T) {
	if s := (Reading{Primary: HexagramFromNumber(0)}).String(); !strings.Contains(s, "stable") {
		t.Errorf("a stable reading should say so: %q", s)
	}
	r := Reading{Primary: HexagramFromNumber(0)}
	r.Changing[0], r.Changing[3] = true, true
	if s := r.String(); !strings.Contains(s, "→") || !strings.Contains(s, "changing lines 1,4") {
		t.Errorf("a changing reading should show the arc and lines: %q", s)
	}
}
