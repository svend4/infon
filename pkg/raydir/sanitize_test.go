package raydir

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// rawBrain returns a fixed raw payload as the rayscene response (to simulate a
// sloppy or adversarial live model).
type rawBrain struct{ ray string }

func (r rawBrain) Decide(brain.Request) (brain.Response, error) {
	return brain.Response{Protocol: brain.Protocol, Kind: "move", Ray: json.RawMessage(r.ray)}, nil
}

// Unknown kinds and non-finite coordinates are dropped; valid objects survive.
func TestSanitizeDropsBadObjects(t *testing.T) {
	spec := brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "garbage", R: 1},                                              // unknown kind
		{Kind: "sphere", X: math.Inf(1), R: 1},                               // non-finite
		{Kind: "sphere", Y: math.NaN(), R: 1},                                // non-finite
		{Kind: "sphere", X: 0, Y: 1, Z: 5, R: 1, Color: [3]float64{1, 0, 0}}, // valid
	}}
	if n := countSpheres(BuildScene(spec).Objects); n != 1 {
		t.Errorf("want exactly 1 valid sphere, got %d", n)
	}
}

// Large-but-finite sizes/emission are clamped (wildly non-finite ones are dropped
// by validObj instead).
func TestSanitizeClamp(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "sphere", Y: 1, R: 5000, Emit: [3]float64{5000, 5000, 5000}, Metal: 9, Rough: -3},
	}})
	if len(s.Objects) != 1 {
		t.Fatalf("clampable object should survive, got %d objects", len(s.Objects))
	}
	sp := s.Objects[0].(raytrace.Sphere)
	if sp.Radius > 100 {
		t.Errorf("radius not clamped: %v", sp.Radius)
	}
	if sp.Mat.Emit.X > 300 {
		t.Errorf("emission not clamped: %v", sp.Mat.Emit)
	}
	if sp.Mat.Metal > 1 || sp.Mat.Rough < 0 {
		t.Errorf("material weights not clamped: metal=%v rough=%v", sp.Mat.Metal, sp.Mat.Rough)
	}
}

// A wildly non-finite size is dropped entirely.
func TestSanitizeDropsHugeSize(t *testing.T) {
	if n := len(BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "sphere", Y: 1, R: 1e9}}}).Objects); n != 0 {
		t.Errorf("a 1e9 radius should be dropped, got %d objects", n)
	}
}

// A runaway object count is capped.
func TestMaxRegionObjects(t *testing.T) {
	objs := make([]brain.ObjSpec, 1000)
	for i := range objs {
		objs[i] = brain.ObjSpec{Kind: "sphere", Y: 1, R: 0.1}
	}
	if n := countSpheres(BuildScene(brain.SceneSpec{Objects: objs}).Objects); n > maxRegionObjects {
		t.Errorf("object count not capped: %d > %d", n, maxRegionObjects)
	}
}

// A live model returning broken JSON, or nothing renderable, falls back to a safe
// region instead of erroring or producing an empty world.
func TestAuthorSceneFallback(t *testing.T) {
	for _, ray := range []string{
		"{not valid json",                    // broken
		`{"objects":[]}`,                     // empty
		`{"objects":[{"kind":"plane"}]}`,     // nothing renderable
		`{"objects":[{"kind":"???","r":1}]}`, // only unknown kinds
	} {
		s, spec, err := AuthorScene(rawBrain{ray}, "x")
		if err != nil {
			t.Fatalf("ray %q: unexpected error %v", ray, err)
		}
		if !hasRenderable(spec) {
			t.Errorf("ray %q: should fall back to a renderable spec", ray)
		}
		if len(s.Objects) == 0 {
			t.Errorf("ray %q: fallback scene should have objects", ray)
		}
	}
	// a good response is used as-is (no fallback).
	_, spec, _ := AuthorScene(rawBrain{`{"objects":[{"kind":"sphere","y":1,"r":1}]}`}, "x")
	if len(spec.Objects) != 1 || spec.Objects[0].Kind != "sphere" {
		t.Errorf("a valid response should be used as-is, got %+v", spec.Objects)
	}
}
