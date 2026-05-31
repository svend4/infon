package tangram_test

import (
	"math/big"
	"testing"

	"github.com/svend4/infon/pkg/tangram"
)

// Number must be a bijection: decoding it back to a figure yields the same id.
func TestNumberRoundTrip(t *testing.T) {
	for name, f := range tangram.Catalog() {
		n := f.Number()
		back := tangram.FromNumber(n, f.H, f.W)
		if back.Number().Cmp(n) != 0 {
			t.Errorf("%s: round-trip number mismatch", name)
		}
		if back.H != f.H || back.W != f.W {
			t.Errorf("%s: round-trip dims %dx%d want %dx%d", name, back.H, back.W, f.H, f.W)
		}
	}
}

// A known tiny grid pins the encoding: [full, tri-ul] = digits [1,2] base 16 = 18.
func TestNumberKnown(t *testing.T) {
	f := tangram.Figure{Name: "x", H: 1, W: 2, Pieces: []tangram.Piece{
		{Glyph: "full", Y: 0, X: 0}, {Glyph: "tri-ul", Y: 0, X: 1},
	}}
	if got, want := f.Number(), big.NewInt(18); got.Cmp(want) != 0 {
		t.Fatalf("Number = %s, want %s", got, want)
	}
	if tangram.FromNumber(big.NewInt(18), 1, 2).Number().Cmp(big.NewInt(18)) != 0 {
		t.Fatal("FromNumber(18) did not round-trip")
	}
}

// The canonical id is invariant to rotation and reflection.
func TestCanonicalInvariant(t *testing.T) {
	for name, f := range tangram.Catalog() {
		base := f.CanonicalNumber()
		if f.RotateCW().CanonicalNumber().Cmp(base) != 0 {
			t.Errorf("%s: rotation changed the canonical id", name)
		}
		if f.MirrorX().CanonicalNumber().Cmp(base) != 0 {
			t.Errorf("%s: reflection changed the canonical id", name)
		}
		if f.RotateCW().RotateCW().CanonicalNumber().Cmp(base) != 0 {
			t.Errorf("%s: 180-degree rotation changed the canonical id", name)
		}
	}
}

// Distinct figures get distinct numbers (and distinct canonical ids).
func TestDistinctNumbers(t *testing.T) {
	seen := map[string]string{}
	canon := map[string]string{}
	for name, f := range tangram.Catalog() {
		if other, ok := seen[f.Number().String()]; ok {
			t.Errorf("%s and %s share a Number", name, other)
		}
		seen[f.Number().String()] = name
		if other, ok := canon[f.CanonicalNumber().String()]; ok {
			t.Errorf("%s and %s share a CanonicalNumber", name, other)
		}
		canon[f.CanonicalNumber().String()] = name
	}
}

// Triangle-only figures are trigram-clean; a figure using a half (boat's mast)
// needs the full nibble alphabet.
func TestTrigramClean(t *testing.T) {
	if !tangram.Tree().TrigramClean() {
		t.Error("tree should be trigram-clean")
	}
	if tangram.Boat().TrigramClean() {
		t.Error("boat uses a half glyph, should not be trigram-clean")
	}
}
