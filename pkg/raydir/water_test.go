package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A kind:"water" spec becomes a reflective water surface a downward ray meets, kept
// in the animated set so it ripples with the world clock.
func TestWaterFromSpec(t *testing.T) {
	w := NewWorld()
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 6}, Spec: brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "water", Y: 0.1, Color: [3]float64{0.1, 0.3, 0.45}, Reflect: 0.55},
	}}})
	if w.Props() != 0 {
		t.Errorf("water should be animated, not a static prop, got %d", w.Props())
	}
	down := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 5, Z: 6}, Dir: raytrace.Vec3{X: 0, Y: -1, Z: 0}}

	w.SetAnimTime(0)
	s0 := w.SceneWith(nil)
	h0, ok0 := nearestHit(s0, down)
	if !ok0 {
		t.Fatal("a downward ray should hit the water")
	}
	if _, isW := s0.Objects[len(s0.Objects)-1].(raytrace.Water); !isW {
		t.Errorf("the water surface should be a raytrace.Water, got %T", s0.Objects[len(s0.Objects)-1])
	}
	// the surface ripples: its normal differs as the clock advances
	w.SetAnimTime(2)
	s1 := w.SceneWith(nil)
	h1, _ := nearestHit(s1, down)
	if h0.N == h1.N {
		t.Error("water normal should change as the waves move with the clock")
	}
}

func TestRefSceneWaterKeyword(t *testing.T) {
	_, spec, _ := AuthorScene(brain.Local{}, "a calm lake at the shore")
	found := false
	for _, o := range spec.Objects {
		if o.Kind == "water" {
			found = true
		}
	}
	if !found {
		t.Error("'lake' should author a water surface")
	}
}
