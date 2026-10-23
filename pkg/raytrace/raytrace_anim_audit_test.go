package raytrace

import (
	"runtime"
	"testing"

	"github.com/svend4/infon/pkg/deltastream"
)

// A delta-stream frame that repeats its predecessor costs two bytes on the wire
// but decodes to a full SceneWire of N spheres, so a small packet can claim a huge
// animation. DecodeAnim must refuse a stream whose total sphere count exceeds its
// budget instead of allocating it. Before the fix any such stream decoded in full:
// at the maximum frame width a 68 KB packet expanded to about 1 GB of spheres.
func TestAuditDecodeAnimBoundsTotalSpheres(t *testing.T) {
	const n = 255
	frames := 20000 // 5.1M spheres; 61.5 MB of frame bytes, under deltastream's own cap
	da := deltastream.Anim{Width: frameWidth(n)}
	key := make([]byte, da.Width)
	for i := 0; i < frames; i++ {
		da.Frames = append(da.Frames, key) // identical frames: zero deltas each
	}
	wire := deltastream.Encode(da, 4)
	t.Logf("wire %d bytes claims %d frames x %d spheres", len(wire), frames, n)
	var a SceneAnim
	var err error
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	a, err = DecodeAnim(wire, 4)
	runtime.ReadMemStats(&after)
	// The ~45 KB wire itself is RS-decoded; the 61.5 MB of frames it declares must
	// not be built before the sphere budget is checked.
	if got := after.TotalAlloc - before.TotalAlloc; got > 8<<20 {
		t.Errorf("rejecting the oversized stream allocated %d bytes", got)
	}
	if err == nil {
		t.Fatalf("DecodeAnim accepted %d frames x %d spheres (%d spheres decoded)", len(a.Frames), a.N, len(a.Frames)*a.N)
	}

	// A stream within the budget still round-trips.
	ok := SceneAnim{N: n}
	for i := 0; i < maxAnimSpheres/n; i++ {
		ok.Frames = append(ok.Frames, SceneWire{Spheres: make([]Sphere, n)})
	}
	got, err := DecodeAnim(EncodeAnim(ok, 4), 4)
	if err != nil {
		t.Fatalf("in-budget animation rejected: %v", err)
	}
	if got.N != n || len(got.Frames) != len(ok.Frames) {
		t.Fatalf("in-budget animation shape = N%d/%d frames, want N%d/%d", got.N, len(got.Frames), n, len(ok.Frames))
	}
}
