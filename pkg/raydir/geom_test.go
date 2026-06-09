package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func trisOf(s *raytrace.Scene) []raytrace.Triangle {
	var out []raytrace.Triangle
	for _, o := range s.Objects {
		if t, ok := o.(raytrace.Triangle); ok {
			out = append(out, t)
		}
	}
	return out
}

func oneObjScene(o brain.ObjSpec) *raytrace.Scene {
	return BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{o}})
}

func TestPrimitiveTriangleCounts(t *testing.T) {
	cases := map[string]int{"pyramid": 6, "cylinder": 64, "box": 12, "house": 18, "tree": 68}
	for kind, want := range cases {
		got := len(trisOf(oneObjScene(brain.ObjSpec{Kind: kind, R: 1})))
		if got != want {
			t.Errorf("%s: %d triangles, want %d", kind, got, want)
		}
	}
}

// A tree is a brown trunk plus green foliage by default.
func TestTreeMaterials(t *testing.T) {
	tr := trisOf(oneObjScene(brain.ObjSpec{Kind: "tree", R: 1}))
	brown, green := false, false
	for _, t := range tr {
		c := t.Mat.Color
		if c.X > c.Y && c.X > c.Z && c.Z < 0.2 { // brownish trunk
			brown = true
		}
		if c.Y > c.X && c.Y > c.Z { // green foliage
			green = true
		}
	}
	if !brown || !green {
		t.Errorf("tree should have a brown trunk and green foliage: brown=%v green=%v", brown, green)
	}
}

// A cylinder placed on the floor is hit on its side by a horizontal ray.
func TestCylinderHit(t *testing.T) {
	s := oneObjScene(brain.ObjSpec{Kind: "cylinder", X: 0, Y: 1, Z: 5, S: [3]float64{1, 1, 1}})
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	hit, near := false, 1e9
	for _, o := range s.Objects {
		if h, ok := o.Intersect(ray, 1e-9, near); ok {
			hit, near = true, h.T
		}
	}
	if !hit || near > 5 { // front of the cylinder is near z=4 (radius 1 at z=5)
		t.Errorf("ray should hit the cylinder side around z=4, hit=%v t=%.2f", hit, near)
	}
}

// The reference author places named meshes for the matching prompts, not otherwise.
func TestRefSceneNamedMeshKeyword(t *testing.T) {
	has := func(prompt, kind string) bool {
		_, spec, _ := AuthorScene(brain.Local{}, prompt)
		for _, o := range spec.Objects {
			if o.Kind == kind {
				return true
			}
		}
		return false
	}
	if !has("a quiet forest path", "tree") {
		t.Error("'forest' should author a tree")
	}
	if !has("a small village", "house") {
		t.Error("'village' should author a house")
	}
	if has("a calm scene of spheres", "tree") || has("a calm scene of spheres", "house") {
		t.Error("a plain prompt should not author trees/houses")
	}
}
