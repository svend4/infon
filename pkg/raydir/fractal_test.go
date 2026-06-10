package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A kind:"fractal" object becomes a ray-marched Marched primitive a ray can hit.
func TestFractalFromSpec(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "fractal", Name: "menger", X: 0, Y: 1, Z: 5, R: 1.5},
	}})
	if len(s.Objects) != 1 {
		t.Fatalf("a fractal should be one object, got %d", len(s.Objects))
	}
	if _, ok := s.Objects[0].(raytrace.Marched); !ok {
		t.Fatalf("fractal should be a Marched, got %T", s.Objects[0])
	}
	hit := false
	for _, dx := range []float64{-0.5, -0.2, 0, 0.2, 0.5} {
		r := raytrace.Ray{Origin: raytrace.Vec3{X: dx, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
		if _, ok := s.Objects[0].Intersect(r, 1e-9, 1e9); ok {
			hit = true
		}
	}
	if !hit {
		t.Error("a ray should hit the menger fractal")
	}
}

// An unknown ray-marched form is dropped by the sanitiser (not rendered, not
// counted as renderable) — a bad reference can't break the scene.
func TestFractalUnknownDropped(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "fractal", Name: "nope", Y: 1}}})
	if len(s.Objects) != 0 {
		t.Errorf("unknown fractal should be dropped, got %d objects", len(s.Objects))
	}
	if hasRenderable(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "fractal", Name: "nope", Y: 1}}}) {
		t.Error("an unknown fractal should not count as renderable")
	}
}

func TestRefSceneFractalKeyword(t *testing.T) {
	named := func(prompt, name string) bool {
		_, spec, _ := AuthorScene(brain.Local{}, prompt)
		for _, o := range spec.Objects {
			if (o.Kind == "fractal" || o.Kind == "sdf") && o.Name == name {
				return true
			}
		}
		return false
	}
	if !named("a glowing mandelbulb fractal", "mandelbulb") {
		t.Error("'mandelbulb' should author a mandelbulb fractal")
	}
	if !named("an escher tiling", "escher") {
		t.Error("'escher' should author the lattice form")
	}
	// the surreal combo brings several forms together
	if !named("a surreal dream", "melt") || !named("a surreal dream", "mandala") {
		t.Error("a surreal prompt should compose melt + mandala")
	}
}
