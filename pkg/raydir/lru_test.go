package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// Prune drops far-behind regions and rebuilds the rest, keeping the world bounded
// as you walk forward; survivors stay intact.
func TestPrune(t *testing.T) {
	w := NewWorld()
	sphere := func() brain.SceneSpec {
		return brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "sphere", Y: 1, R: 1, Color: [3]float64{0.8, 0.3, 0.3}}}}
	}
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: sphere()})
	w.AddRegion(Region{Index: 1, At: raytrace.Vec3{Z: 20}, Spec: sphere()})
	w.AddRegion(Region{Index: 2, At: raytrace.Vec3{Z: 40}, Spec: sphere()})
	if w.Chunks() != 3 {
		t.Fatalf("expected 3 chunks, got %d", w.Chunks())
	}
	propsBefore := w.Props()

	dropped := w.Prune(15) // drop everything behind z=15 (region 0 at z=8)
	if dropped != 1 {
		t.Fatalf("expected to drop 1 region, dropped %d", dropped)
	}
	if w.Chunks() != 2 {
		t.Errorf("after prune expected 2 chunks, got %d", w.Chunks())
	}
	if w.Props() >= propsBefore {
		t.Errorf("pruning should shrink props: before %d after %d", propsBefore, w.Props())
	}
	if len(w.Landmarks()) != 2 {
		t.Errorf("pruned region's landmark should be gone, got %d", len(w.Landmarks()))
	}
	// a survivor (region 1 at z=20) is still there
	probe := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 12}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	if _, ok := nearestHit(w.SceneWith(nil), probe); !ok {
		t.Error("a surviving region should still be hittable")
	}
	if w.Prune(15) != 0 {
		t.Error("pruning again should drop nothing")
	}
}
