package raydir

import (
	"math"
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func sampleGraph() (*BubbleGraph, [5]int) {
	g := NewBubbleGraph()
	home := g.Add("Home", raytrace.Vec3{X: 0, Z: 0}, brain.SceneSpec{})
	forest := g.Add("Forest", raytrace.Vec3{X: 4, Z: 2}, brain.SceneSpec{})
	city := g.Add("City", raytrace.Vec3{X: 8, Z: 0}, brain.SceneSpec{})
	cave := g.Add("Cave", raytrace.Vec3{X: 12, Z: 3}, brain.SceneSpec{})
	deadend := g.Add("Dead End", raytrace.Vec3{X: 4, Z: -4}, brain.SceneSpec{})
	g.Link(home, forest)
	g.Link(forest, city)
	g.Link(city, cave)
	g.Link(home, deadend) // a dead-end bubble off home
	return g, [5]int{home, forest, city, cave, deadend}
}

func TestBubbleAddAndLink(t *testing.T) {
	g, ids := sampleGraph()
	if g.Len() != 5 {
		t.Fatalf("expected 5 bubbles, got %d", g.Len())
	}
	if g.Home() != ids[0] {
		t.Errorf("the first bubble should be home, got %d", g.Home())
	}
	n := g.Neighbors(ids[0])
	if !reflect.DeepEqual(n, []int{ids[1], ids[4]}) { // forest and dead-end
		t.Errorf("home neighbours should be forest and dead-end, got %v", n)
	}
	// links are undirected
	found := false
	for _, m := range g.Neighbors(ids[1]) {
		if m == ids[0] {
			found = true
		}
	}
	if !found {
		t.Error("transits should be undirected")
	}
}

func TestBubbleRoute(t *testing.T) {
	g, ids := sampleGraph()
	home, _, city, cave, deadend := ids[0], ids[1], ids[2], ids[3], ids[4]
	route := g.Route(home, cave)
	if !reflect.DeepEqual(route, []int{home, ids[1], city, cave}) {
		t.Errorf("route home->cave should pass forest and city, got %v", route)
	}
	if r := g.Route(home, home); !reflect.DeepEqual(r, []int{home}) {
		t.Errorf("route to self should be a single bubble, got %v", r)
	}
	// dead end is reachable from home directly
	if r := g.Route(home, deadend); !reflect.DeepEqual(r, []int{home, deadend}) {
		t.Errorf("route home->deadend should be direct, got %v", r)
	}
	// unreachable
	lone := g.Add("Lone", raytrace.Vec3{X: -8}, brain.SceneSpec{})
	if r := g.Route(home, lone); r != nil {
		t.Errorf("an unlinked bubble should be unreachable, got %v", r)
	}
}

func TestBubbleMapRenders(t *testing.T) {
	g, ids := sampleGraph()
	route := g.Route(ids[0], ids[3])
	img := g.BubbleMap(ids[2], route, 320, 240)
	if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 240 {
		t.Fatalf("unexpected map size %v", img.Bounds())
	}
	// the diagram should not be empty background only: some bright node pixels exist
	bright := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, gg, bl, _ := img.At(x, y).RGBA()
			if r>>8 > 120 || gg>>8 > 120 || bl>>8 > 120 {
				bright++
			}
		}
	}
	if bright < 50 {
		t.Errorf("the bubble diagram should draw nodes/edges, only %d bright px", bright)
	}
}

func bdist(g *BubbleGraph, a, b int) float64 {
	x, _ := g.Get(a)
	y, _ := g.Get(b)
	return math.Hypot(x.At.X-y.At.X, x.At.Z-y.At.Z)
}

// Layout is deterministic, pins home at the origin, separates nodes (no overlap),
// and places graph-near bubbles spatially nearer than graph-far ones.
func TestBubbleLayout(t *testing.T) {
	g, ids := sampleGraph()
	g.Layout(400)
	home, forest, _, cave, _ := ids[0], ids[1], ids[2], ids[3], ids[4]

	// home pinned at the origin
	if h, _ := g.Get(home); math.Hypot(h.At.X, h.At.Z) > 1e-6 {
		t.Errorf("home should be pinned at the origin, got %+v", h.At)
	}
	// no two bubbles overlap, and all finite
	for i := 0; i < g.Len(); i++ {
		for j := i + 1; j < g.Len(); j++ {
			d := bdist(g, g.order[i], g.order[j])
			if math.IsNaN(d) || math.IsInf(d, 0) {
				t.Fatal("layout produced a non-finite position")
			}
			if d < 1.5 {
				t.Errorf("bubbles %d,%d overlap (d=%.2f)", g.order[i], g.order[j], d)
			}
		}
	}
	// a one-transit neighbour sits nearer than a three-transit world
	if bdist(g, home, forest) >= bdist(g, home, cave) {
		t.Errorf("graph-near should be spatially near: home-forest %.1f, home-cave %.1f",
			bdist(g, home, forest), bdist(g, home, cave))
	}
	// determinism: a second run from the same graph gives the same positions
	g2, _ := sampleGraph()
	g2.Layout(400)
	for _, id := range g.order {
		a, _ := g.Get(id)
		b, _ := g2.Get(id)
		if math.Hypot(a.At.X-b.At.X, a.At.Z-b.At.Z) > 1e-9 {
			t.Fatalf("layout should be deterministic; bubble %d differs", id)
		}
	}
}
