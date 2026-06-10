package raydir

import (
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// hexbridge.go realises the coordination idea behind svend4/info150's `portal`:
// "not merger, compatibility" — discover relationships between independent things
// by reading a small coordinate, without coupling them. info150 places each domain
// on a 6-bit coordinate and bridges by Hamming distance; here each world carries a
// hexagram (its Q6 coordinate), and HexBridges links worlds whose hexagrams are
// close in the hypercube — a read-only view over the worlds, never modifying them.

// HexPlace is a world (or region) tagged with its hexagram coordinate.
type HexPlace struct {
	Name string
	Hex  Hexagram
}

// HexBridges builds a graph that links any two places whose hexagrams differ by at
// most `maxHamming` lines — Q6-proximity bridges discovered purely by reading each
// place's coordinate. The inputs are not modified (compatibility, not merger).
func HexBridges(places []HexPlace, maxHamming int) *BubbleGraph {
	g := NewBubbleGraph()
	ids := make([]int, len(places))
	for i, p := range places {
		ids[i] = g.Add(p.Name, raytrace.Vec3{}, brain.SceneSpec{Name: p.Name})
	}
	for i := 0; i < len(places); i++ {
		for j := i + 1; j < len(places); j++ {
			if places[i].Hex.Hamming(places[j].Hex) <= maxHamming {
				g.Link(ids[i], ids[j])
			}
		}
	}
	return g
}
