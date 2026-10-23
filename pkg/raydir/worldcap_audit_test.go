package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A stream of new region indices (a long walk, or a peer flooding regions) must
// not grow the world without bound: applied regions, landmarks and props stay
// capped, and the regions kept are the ones nearest the frontier.
func TestWorldCapsAppliedRegions(t *testing.T) {
	w := NewWorld()
	sphere := brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "sphere", Y: 1, R: 1}}}
	const N = 3000
	for i := 0; i < N; i++ {
		w.AddRegion(Region{Index: i, At: raytrace.Vec3{Z: float64(i)}, Spec: sphere})
	}
	if n := len(w.applied); n > 1024 {
		t.Errorf("applied regions = %d after %d arrivals, want <= 1024", n, N)
	}
	if n := len(w.Landmarks()); n > 1024 {
		t.Errorf("landmarks = %d after %d arrivals, want <= 1024", n, N)
	}
	if !w.Has(N - 1) {
		t.Error("the newest (furthest-ahead) region must be kept")
	}
	if w.Has(0) {
		t.Error("the furthest-behind region should have been shed")
	}
	if w.Chunks() != len(w.Known()) || w.Chunks() == 0 {
		t.Errorf("chunks %d inconsistent with known %d", w.Chunks(), len(w.Known()))
	}

	// one packet's worth of cylinders expands into ~1300 props: a few hundred such
	// regions must not grow the prop list without bound either.
	var objs []brain.ObjSpec
	for len(Region{Spec: brain.SceneSpec{Objects: objs}}.Encode()) < 1300 {
		objs = append(objs, brain.ObjSpec{Kind: "cylinder"})
	}
	dense := brain.SceneSpec{Objects: objs}
	w = NewWorld()
	const M = 200
	for i := 0; i < M; i++ {
		w.AddRegion(Region{Index: i, At: raytrace.Vec3{Z: float64(i)}, Spec: dense})
	}
	if n := w.Props(); n > 1<<16 {
		t.Errorf("props = %d after %d dense regions, want <= %d", n, M, 1<<16)
	}
	if !w.Has(M-1) || w.Props() == 0 {
		t.Errorf("the newest dense region must be kept (props=%d)", w.Props())
	}
}

// A region at a non-finite or out-of-range position is malformed: it is neither
// decoded nor applied (so Prune and the caps never have to order it).
func TestRegionRejectsBadPosition(t *testing.T) {
	spec := brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "sphere", Y: 1, R: 1}}}
	bad := []raytrace.Vec3{
		{Z: math.NaN()}, {Z: math.Inf(1)}, {X: math.Inf(-1)}, {Y: math.NaN()}, {Z: 1e12},
	}
	for i, at := range bad {
		r := Region{Index: 100 + i, At: at, Spec: spec}
		if _, err := DecodeRegion(r.Encode()); err == nil {
			t.Errorf("DecodeRegion accepted position %+v", at)
		}
		w := NewWorld()
		w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: spec})
		if n := w.AddRegion(r); n != 0 || w.Has(r.Index) || w.Chunks() != 1 {
			t.Errorf("AddRegion applied position %+v (added %d)", at, n)
		}
		if w.Prune(0) != 0 || w.Chunks() != 1 {
			t.Errorf("after rejecting %+v, Prune(0) should keep the one good region", at)
		}
	}
	// a NaN depth prunes nothing rather than everything.
	w := NewWorld()
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: spec})
	if d := w.Prune(math.NaN()); d != 0 || w.Chunks() != 1 {
		t.Errorf("Prune(NaN) dropped %d regions", d)
	}
}
