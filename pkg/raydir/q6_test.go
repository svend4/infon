package raydir

import "testing"

func TestHexagramFromNumberRoundTrip(t *testing.T) {
	for n := 0; n < 64; n++ {
		if got := HexagramFromNumber(n).Number(); got != n {
			t.Fatalf("round trip %d -> %d", n, got)
		}
	}
}

func TestHexagramFlipAndNeighbors(t *testing.T) {
	h := HexagramFromNumber(0)
	f := h.Flip(2)
	if f.Number() != 1<<2 || h.Number() != 0 {
		t.Errorf("Flip(2) should toggle only line 2 and not mutate the original")
	}
	n := h.Neighbors()
	if len(n) != 6 {
		t.Fatalf("expected 6 neighbours, got %d", len(n))
	}
	seen := map[int]bool{}
	for _, m := range n {
		if h.Hamming(m) != 1 {
			t.Errorf("a neighbour must differ by one line, got %d", h.Hamming(m))
		}
		seen[m.Number()] = true
	}
	if len(seen) != 6 {
		t.Errorf("neighbours should be distinct, got %d", len(seen))
	}
}

func TestHexagramAntipodeAndHamming(t *testing.T) {
	h := HexagramFromNumber(0b101010)
	a := h.Antipode()
	if h.Hamming(a) != 6 {
		t.Errorf("antipode should be Hamming 6 away, got %d", h.Hamming(a))
	}
	if a.Antipode().Number() != h.Number() {
		t.Error("antipode of antipode should be the original")
	}
	if HexagramFromNumber(0).Hamming(HexagramFromNumber(0b111)) != 3 {
		t.Error("Hamming(0, 000111) should be 3")
	}
}

// The Gray walk is a Hamiltonian path on Q6: 64 distinct hexagrams, starting where
// asked, each consecutive pair differing by exactly one line.
func TestGrayWalkHamiltonian(t *testing.T) {
	start := HexagramFromNumber(0b011010)
	walk := start.GrayWalk()
	if len(walk) != 64 {
		t.Fatalf("a grand tour should visit 64, got %d", len(walk))
	}
	if walk[0].Number() != start.Number() {
		t.Errorf("the walk should start at the start hexagram, got %d", walk[0].Number())
	}
	seen := map[int]bool{}
	for i, h := range walk {
		seen[h.Number()] = true
		if i > 0 && walk[i-1].Hamming(h) != 1 {
			t.Fatalf("step %d changes %d lines, must be exactly 1", i, walk[i-1].Hamming(h))
		}
	}
	if len(seen) != 64 {
		t.Errorf("the walk should visit every hexagram once, got %d distinct", len(seen))
	}
}

// The 6-bit interchange format round-trips for all 64 hexagrams (the lingua franca
// shared with meta/pro2/info150 — see docs/Q6_INTEROP.md).
func TestHexInteropRoundTrip(t *testing.T) {
	for n := 0; n < 64; n++ {
		h := HexagramFromNumber(n)
		s := h.String()
		if len(s) != 6 {
			t.Fatalf("interchange string for %d should be 6 chars, got %q", n, s)
		}
		back, ok := ParseHexagram(s)
		if !ok || back.Number() != n {
			t.Fatalf("round trip failed: %d -> %q -> %d (ok=%v)", n, s, back.Number(), ok)
		}
	}
}
