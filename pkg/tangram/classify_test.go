package tangram_test

import (
	"testing"
	"unicode/utf8"

	"github.com/svend4/infon/pkg/tangram"
)

// Every catalog figure classifies as itself with a perfect score.
func TestClassifyExact(t *testing.T) {
	for name, f := range tangram.Catalog() {
		m := tangram.Classify(f)
		if m.Name != name {
			t.Errorf("%s classified as %s", name, m.Name)
		}
		if m.Score < 0.999 {
			t.Errorf("%s self-score = %.3f, want 1", name, m.Score)
		}
	}
}

// Classification is pose-invariant: a mirrored or rotated cat is still a cat.
func TestClassifyPoseInvariant(t *testing.T) {
	cat := tangram.Cat()
	if m := tangram.Classify(cat.MirrorX()); m.Name != "cat" || m.Score < 0.999 {
		t.Fatalf("mirror(cat) -> %s %.3f", m.Name, m.Score)
	}
	if m := tangram.Classify(cat.RotateCW()); m.Name != "cat" || m.Score < 0.999 {
		t.Fatalf("rotate(cat) -> %s %.3f", m.Name, m.Score)
	}
}

// A near-cat (one stray cell) still names cat, with a high but sub-1 score.
func TestClassifyNearest(t *testing.T) {
	c := tangram.Cat()
	c.Pieces = append(c.Pieces, tangram.Piece{Glyph: "full", Y: 7, X: 0})
	m := tangram.Classify(c)
	if m.Name != "cat" {
		t.Fatalf("near-cat classified as %s", m.Name)
	}
	if !(m.Score > 0.9 && m.Score < 1.0) {
		t.Fatalf("near-cat score = %.4f, want in (0.9,1)", m.Score)
	}
}

// The address is the dialing sequence: one glyph per non-blank cell.
func TestAddressLength(t *testing.T) {
	cat := tangram.Cat()
	if n := utf8.RuneCountInString(cat.Address()); n != 32 {
		t.Fatalf("cat address has %d glyphs, want 32 (filled cells)", n)
	}
	if g := tangram.Cat().GateCode(7); utf8.RuneCountInString(g) != 7 {
		t.Fatalf("gate code has %d glyphs, want 7", utf8.RuneCountInString(g))
	}
}
