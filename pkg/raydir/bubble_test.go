package raydir

import (
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
