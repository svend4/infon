package raydir

import "testing"

func TestHexBridges(t *testing.T) {
	places := []HexPlace{
		{"A", HexagramFromNumber(0b000000)},
		{"B", HexagramFromNumber(0b000001)}, // 1 from A
		{"C", HexagramFromNumber(0b000011)}, // 1 from B, 2 from A
		{"D", HexagramFromNumber(0b111111)}, // 6 from A (the antipode), far from all
	}
	// maxHamming = 1: only adjacent pairs bridge (A-B, B-C)
	g := HexBridges(places, 1)
	if g.Len() != 4 {
		t.Fatalf("expected 4 places, got %d", g.Len())
	}
	hasEdge := func(g *BubbleGraph, a, b int) bool {
		for _, n := range g.Neighbors(a) {
			if n == b {
				return true
			}
		}
		return false
	}
	if !hasEdge(g, 0, 1) || !hasEdge(g, 1, 2) {
		t.Error("A-B and B-C should bridge at Hamming 1")
	}
	if hasEdge(g, 0, 2) {
		t.Error("A-C are Hamming 2 apart and should not bridge at maxHamming 1")
	}
	if len(g.Neighbors(3)) != 0 {
		t.Errorf("the antipode D should bridge to nothing at Hamming 1, got %v", g.Neighbors(3))
	}
	// maxHamming = 6: everything bridges to everything (complete graph)
	full := HexBridges(places, 6)
	for i := 0; i < 4; i++ {
		if len(full.Neighbors(i)) != 3 {
			t.Errorf("at maxHamming 6 every place should bridge to the other 3, %d has %d", i, len(full.Neighbors(i)))
		}
	}
}
