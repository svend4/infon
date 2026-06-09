package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// Built-in bump maps resolve; unknown names don't.
func TestBumpLibrary(t *testing.T) {
	for _, name := range []string{"ripple", "waves", "bumps"} {
		if BumpFor(name) == nil {
			t.Errorf("built-in bump %q should be registered", name)
		}
	}
	if BumpFor("nope") != nil {
		t.Error("unknown bump should resolve to nil")
	}
}

// A named bump on an object sets Material.Bump; an unknown one leaves it flat.
func TestSurfaceBump(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "sphere", Y: 1, Z: 5, R: 1, Color: [3]float64{0.7, 0.7, 0.7}, Bump: "ripple"},
	}})
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	h, ok := nearestHit(s, ray)
	if !ok || h.Mat.Bump == nil {
		t.Fatalf("a bumped sphere should carry Material.Bump (ok=%v)", ok)
	}
	flat := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "sphere", Y: 1, Z: 5, R: 1, Color: [3]float64{0.7, 0.7, 0.7}, Bump: "bogus"},
	}})
	h2, ok2 := nearestHit(flat, ray)
	if !ok2 || h2.Mat.Bump != nil {
		t.Error("unknown bump should leave the surface flat")
	}
}
