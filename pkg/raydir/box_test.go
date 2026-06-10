package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A box spec becomes 12 triangles, carries its material, and is hit where its
// front face is.
func TestBuildSceneBox(t *testing.T) {
	spec := brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "box", X: 0, Y: 1, Z: 5, S: [3]float64{1, 2, 0.5}, Color: [3]float64{0.5, 0.4, 0.3}, Rough: 0.4},
	}}
	s := BuildScene(spec)
	tris := 0
	for _, o := range s.Objects {
		if _, ok := o.(raytrace.Triangle); ok {
			tris++
		}
	}
	if tris != 12 {
		t.Fatalf("a box should be 12 triangles, got %d", tris)
	}
	// ray along +Z through the box centre hits the front face at z = 5 - 0.5.
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	var h raytrace.Hit
	ok, near := false, 1e9
	for _, o := range s.Objects {
		if hit, hitOK := o.Intersect(ray, 1e-9, near); hitOK {
			h, ok, near = hit, true, hit.T
		}
	}
	if !ok {
		t.Fatal("ray should hit the box")
	}
	if math.Abs(h.P.Z-4.5) > 1e-6 {
		t.Errorf("hit at z=%.4f, want 4.5 (box front face)", h.P.Z)
	}
	if h.Mat.Color.X != 0.5 || h.Mat.Rough != 0.4 {
		t.Errorf("box material not carried: %+v", h.Mat)
	}
}

// A box defaults to a cube of half-size R when no half-extents are given.
func TestBuildSceneBoxCubeFallback(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "box", Y: 0, R: 2}}})
	box, _ := objectBoundsViaScene(s)
	if math.Abs(box) < 1 {
		t.Errorf("cube fallback should have extent, got %.2f", box)
	}
}

// The reference author adds a box (recognizable structure) for structural prompts,
// and not otherwise.
func TestRefSceneBoxKeyword(t *testing.T) {
	_, spec, err := AuthorScene(brain.Local{}, "a tall tower")
	if err != nil {
		t.Fatal(err)
	}
	has := false
	for _, o := range spec.Objects {
		if o.Kind == "box" {
			has = true
		}
	}
	if !has {
		t.Error("a 'tower' prompt should author a box")
	}
	_, plain, _ := AuthorScene(brain.Local{}, "a calm scene of spheres")
	for _, o := range plain.Objects {
		if o.Kind == "box" {
			t.Error("a plain prompt should not author a box")
		}
	}
}

// helper: rough extent of a scene's first box via its triangle spread on X.
func objectBoundsViaScene(s *raytrace.Scene) (float64, bool) {
	min, max := math.MaxFloat64, -math.MaxFloat64
	found := false
	for _, o := range s.Objects {
		if tr, ok := o.(raytrace.Triangle); ok {
			found = true
			for _, x := range []float64{tr.A.X, tr.B.X, tr.C.X} {
				if x < min {
					min = x
				}
				if x > max {
					max = x
				}
			}
		}
	}
	return max - min, found
}
