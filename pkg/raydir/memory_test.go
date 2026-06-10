package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func TestTokenizeAndJaccard(t *testing.T) {
	a := tokenize("a dark forest with tall trees")
	if a["the"] || a["a"] || a["with"] {
		t.Error("stopwords should be dropped")
	}
	if !a["forest"] || !a["trees"] {
		t.Error("content words should be kept")
	}
	b := tokenize("a quiet forest of trees")
	if jaccard(a, b) <= 0 {
		t.Error("overlapping texts should have positive similarity")
	}
	if jaccard(a, tokenize("a city of glass towers")) >= jaccard(a, b) {
		t.Error("a dissimilar text should score lower than a similar one")
	}
}

func TestMemoryRecall(t *testing.T) {
	m := &Memory{}
	m.Remember(0, "Forest", "a dark forest with fireflies")
	m.Remember(1, "City", "a city of glass towers by water")
	m.Remember(2, "Cave", "a cave of glowing crystals")
	e, score, ok := m.Recall("walking into a forest at night with fireflies")
	if !ok || e.Name != "Forest" {
		t.Errorf("should recall the Forest, got %q (score %.2f)", e.Name, score)
	}
	if _, _, ok := m.Recall("zxqw nothing matches here"); ok {
		t.Error("an unrelated query should recall nothing")
	}
}

func TestMemoryGraph(t *testing.T) {
	m := &Memory{}
	m.Remember(0, "Forest", "a dark forest with tall trees")
	m.Remember(1, "Grove", "a quiet grove of trees and ferns") // similar to Forest (trees)
	m.Remember(2, "City", "a city of glass towers")            // dissimilar
	g := m.Graph(0.1)
	if g.Len() != 3 {
		t.Fatalf("graph should have 3 places, got %d", g.Len())
	}
	// Forest and Grove share "trees" -> linked; City stands apart
	forestNeighbors := g.Neighbors(0)
	if len(forestNeighbors) != 1 || forestNeighbors[0] != 1 {
		t.Errorf("Forest should link only to Grove, got %v", forestNeighbors)
	}
	if len(g.Neighbors(2)) != 0 {
		t.Errorf("the dissimilar City should be unlinked, got %v", g.Neighbors(2))
	}
}

// The world remembers regions it grows and can recall a callback hint.
func TestWorldMemory(t *testing.T) {
	w := NewWorld()
	w.SetMemory(true)
	b := brain.Local{}
	_, _ = w.Grow(b, "a dark forest with fireflies and trees", raytrace.Vec3{Z: 10})
	_, _ = w.Grow(b, "a city of glass towers", raytrace.Vec3{Z: 24})
	if w.memory.Len() != 2 {
		t.Fatalf("the world should remember 2 regions, got %d", w.memory.Len())
	}
	hint, ok := w.RecallHint("another forest of trees ahead")
	if !ok || hint == "" {
		t.Errorf("a forest-like prompt should recall a callback, got %q ok=%v", hint, ok)
	}
}
