package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func TestRegionRoundTrip(t *testing.T) {
	r := Region{
		Index: 7,
		At:    raytrace.Vec3{X: 1, Y: 2, Z: 30},
		Spec: brain.SceneSpec{
			Light:   [3]float64{6, 9, -4},
			Objects: []brain.ObjSpec{{X: -1.4, Y: 1, R: 1, Color: [3]float64{0.8, 0.3, 0.3}}, {Y: 6, R: 0.7, Emit: [3]float64{18, 18, 17}}},
		},
	}
	got, err := DecodeRegion(r.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if got.Index != r.Index || got.At != r.At || len(got.Spec.Objects) != len(r.Spec.Objects) {
		t.Errorf("round trip = %+v, want %+v", got, r)
	}
	if _, err := DecodeRegion([]byte{1, 2}); err == nil {
		t.Error("short payload should error")
	}
}

// Re-applying the same region index must not duplicate its props (UDP re-broadcast
// safety).
func TestAddRegionIdempotent(t *testing.T) {
	w := NewWorld()
	host := NewWorld()
	reg, n, err := host.AuthorRegion(brain.Local{}, "studio", 3, raytrace.Vec3{Z: 20})
	if err != nil || n == 0 {
		t.Fatalf("author failed: n=%d err=%v", n, err)
	}
	if got := w.AddRegion(reg); got != n {
		t.Errorf("first AddRegion added %d, want %d", got, n)
	}
	if got := w.AddRegion(reg); got != 0 {
		t.Errorf("re-applying the same region should add 0, added %d", got)
	}
	if w.Props() != n || !w.Has(3) {
		t.Errorf("idempotency broken: props=%d has(3)=%v", w.Props(), w.Has(3))
	}
}

// A guest that applies the regions a host broadcast ends up with the host's exact
// world — the basis for keeping a shared world in sync with any (even live) brain.
func TestHostGuestConverge(t *testing.T) {
	b := brain.Local{}
	prompts := []string{"a calm world", "a gold sphere", "glass and metal"}
	host := NewWorld()
	guest := NewWorld()
	var broadcast []Region
	for i := 0; i < 3; i++ {
		reg, _, err := host.AuthorRegion(b, prompts[i], i, raytrace.Vec3{Z: float64(8 + i*12)})
		if err != nil {
			t.Fatal(err)
		}
		broadcast = append(broadcast, reg)
	}
	// guest applies them (in a shuffled order, some twice — like lossy UDP re-sends).
	order := []int{2, 0, 0, 1, 2}
	for _, i := range order {
		guest.AddRegion(broadcast[i])
	}
	if host.Chunks() != guest.Chunks() || host.Props() != guest.Props() {
		t.Fatalf("worlds diverged: host(chunks=%d props=%d) guest(chunks=%d props=%d)",
			host.Chunks(), host.Props(), guest.Chunks(), guest.Props())
	}
	// the props must be the same spheres in the same places.
	hs, gs := host.Scene().Objects, guest.Scene().Objects
	if len(hs) != len(gs) {
		t.Fatalf("object counts differ: %d vs %d", len(hs), len(gs))
	}
	matched := 0
	for _, ho := range hs {
		h, ok := ho.(raytrace.Sphere)
		if !ok {
			continue
		}
		for _, go_ := range gs {
			if g, ok := go_.(raytrace.Sphere); ok && g.Center == h.Center && g.Radius == h.Radius {
				matched++
				break
			}
		}
	}
	if spheres := countSpheres(hs); matched < spheres {
		t.Errorf("guest world should match the host's: matched %d of %d spheres", matched, spheres)
	}
}

func countSpheres(objs []raytrace.Object) int {
	n := 0
	for _, o := range objs {
		if _, ok := o.(raytrace.Sphere); ok {
			n++
		}
	}
	return n
}
