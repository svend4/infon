package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// A placement becomes a one-object region that renders (the object sits in the
// builder's colour where the region is placed).
func TestPlacementSpec(t *testing.T) {
	col := raytrace.Vec3{X: 0.2, Y: 0.8, Z: 1}
	spec, ok := PlacementSpec("crystal", col)
	if !ok || len(spec.Objects) != 1 {
		t.Fatalf("crystal placement should be one object (ok=%v)", ok)
	}
	if spec.Objects[0].Kind != "mesh" || spec.Objects[0].Name != "crystal" {
		t.Errorf("crystal should map to mesh/crystal, got %+v", spec.Objects[0])
	}
	if spec.Objects[0].Color != [3]float64{0.2, 0.8, 1} {
		t.Errorf("placement should carry the builder's colour, got %v", spec.Objects[0].Color)
	}
	// it applies to a world as a region a ray can hit
	w := NewWorld()
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{X: 0, Y: 0, Z: 5}, Spec: spec})
	if w.Props() == 0 {
		t.Error("a placed crystal should yield props")
	}
}

func TestPlacementUnknown(t *testing.T) {
	if _, ok := PlacementSpec("not-a-thing", raytrace.Vec3{}); ok {
		t.Error("unknown placement kind should return ok=false")
	}
	if len(PlaceKinds()) == 0 {
		t.Error("PlaceKinds should list buildable kinds")
	}
}

// A fractal placement maps to the ray-marched kind.
func TestPlacementFractal(t *testing.T) {
	spec, ok := PlacementSpec("mandelbulb", raytrace.Vec3{X: 1})
	if !ok || spec.Objects[0].Kind != "fractal" || spec.Objects[0].Name != "mandelbulb" {
		t.Errorf("mandelbulb placement should map to fractal/mandelbulb, got ok=%v %+v", ok, spec.Objects)
	}
}

// Positional voice: a nearer speaker is louder than a far one, and a speaker ahead
// is louder than one behind at the same distance.
func TestVoiceGain(t *testing.T) {
	listener := raytrace.Vec3{X: 0, Y: 0, Z: 0}
	facing := raytrace.Vec3{X: 0, Y: 0, Z: 1} // looking +Z
	near := VoiceGain(listener, facing, raytrace.Vec3{X: 0, Y: 0, Z: 2})
	far := VoiceGain(listener, facing, raytrace.Vec3{X: 0, Y: 0, Z: 20})
	if near <= far {
		t.Errorf("nearer speaker should be louder: near %.3f far %.3f", near, far)
	}
	ahead := VoiceGain(listener, facing, raytrace.Vec3{X: 0, Y: 0, Z: 6})
	behind := VoiceGain(listener, facing, raytrace.Vec3{X: 0, Y: 0, Z: -6})
	if ahead <= behind {
		t.Errorf("speaker ahead should be louder than behind: ahead %.3f behind %.3f", ahead, behind)
	}
	if g := VoiceGain(listener, facing, listener); g < 0.99 {
		t.Errorf("a co-located speaker should be full volume, got %.3f", g)
	}
}

// The mixer applies per-source gain when summing.
func TestMixerGain(t *testing.T) {
	m := NewVoiceMixer(2)
	m.Add(1, []int16{1000, 1000})
	m.SetGain(1, 0.5)
	got := m.Mix()
	if got[0] != 500 || got[1] != 500 {
		t.Errorf("gain 0.5 should halve samples, got %v", got)
	}
}
