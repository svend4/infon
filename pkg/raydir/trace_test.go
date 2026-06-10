package raydir

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// Treading a cell repeatedly deepens it; a worn cell renders worn-earth objects.
func TestTraceWear(t *testing.T) {
	tr := NewTrace(1.5)
	if len(tr.Objects()) != 0 {
		t.Error("an untrodden world has no worn patches")
	}
	for i := 0; i < 5; i++ {
		tr.Tread(raytrace.Vec3{X: 0.2, Z: 0.3}) // same cell
	}
	tr.Tread(raytrace.Vec3{X: 10, Z: 10}) // a different, lightly trodden cell
	if tr.Cells() != 2 {
		t.Fatalf("expected 2 distinct cells, got %d", tr.Cells())
	}
	if len(tr.Objects()) == 0 {
		t.Error("a well-trodden cell should render worn earth")
	}
}

// The trace round-trips through encode/decode (so wear can be saved and shared).
func TestTraceRoundTrip(t *testing.T) {
	tr := NewTrace(2.0)
	tr.Tread(raytrace.Vec3{X: 1, Z: 1})
	tr.Tread(raytrace.Vec3{X: 1, Z: 1})
	tr.Tread(raytrace.Vec3{X: -5, Z: 7})
	got, err := DecodeTrace(tr.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if got.cell != tr.cell || !reflect.DeepEqual(got.counts, tr.counts) {
		t.Fatalf("trace round-trip mismatch:\n in %+v\nout %+v", tr.counts, got.counts)
	}
	if _, err := DecodeTrace([]byte("nope")); err == nil {
		t.Error("bad data should error")
	}
}

// Walking the world wears a visible path: a ray onto the floor where you walked
// meets worn earth that wasn't there before.
func TestWorldTread(t *testing.T) {
	w := NewWorld()
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 5}, Spec: brain.SceneSpec{}})
	spot := raytrace.Vec3{X: 3.75, Z: 5.25} // a ground-cell centre
	probe := raytrace.Ray{Origin: raytrace.Vec3{X: spot.X, Y: 3, Z: spot.Z}, Dir: raytrace.Vec3{X: 0, Y: -1, Z: 0}}
	before, _ := nearestHit(w.SceneWith(nil), probe)
	for i := 0; i < 4; i++ {
		w.Tread(spot)
	}
	after, ok := nearestHit(w.SceneWith(nil), probe)
	if !ok {
		t.Fatal("the ray should hit the floor or worn patch")
	}
	if after.T >= before.T-1e-9 {
		t.Error("a worn patch sits above the floor, so the ray should hit it sooner")
	}
}
