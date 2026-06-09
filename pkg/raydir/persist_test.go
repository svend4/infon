package raydir

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A saved world reloads identically — regions, positions and scene specs intact.
func TestWorldSaveLoadRoundTrip(t *testing.T) {
	regions := []Region{
		{Index: 0, At: raytrace.Vec3{X: 1, Y: 0, Z: 8}, Spec: fallbackSpec()},
		{Index: 1, At: raytrace.Vec3{X: -2, Y: 0, Z: 20}, Spec: brain.SceneSpec{
			Objects: []brain.ObjSpec{{Kind: "mesh", Name: "crystal", R: 2, Color: [3]float64{0.9, 0.2, 0.2}}},
			Light:   [3]float64{1, 2, 3},
		}},
	}
	path := filepath.Join(t.TempDir(), "world.rwld")
	if err := SaveWorld(path, regions); err != nil {
		t.Fatal(err)
	}
	got, err := LoadWorldFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(regions, got) {
		t.Fatalf("round-trip mismatch:\n want %+v\n got  %+v", regions, got)
	}
}

// A reloaded world rebuilds the same scene a guest would: the persisted regions
// apply to a fresh World and produce renderable props.
func TestLoadedWorldRebuilds(t *testing.T) {
	regions := []Region{{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: brain.SceneSpec{
		Objects: []brain.ObjSpec{{Kind: "mesh", Name: "crystal", R: 1}},
	}}}
	path := filepath.Join(t.TempDir(), "w.rwld")
	if err := SaveWorld(path, regions); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWorldFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorld()
	for _, r := range loaded {
		w.AddRegion(r)
	}
	if w.Chunks() != 1 {
		t.Errorf("expected 1 chunk applied, got %d", w.Chunks())
	}
	if w.Props() == 0 {
		t.Error("a reloaded crystal region should yield props")
	}
}

// Decoding rejects a corrupt or alien file instead of panicking.
func TestDecodeWorldBad(t *testing.T) {
	if _, err := DecodeWorld(nil); err == nil {
		t.Error("empty should error")
	}
	if _, err := DecodeWorld([]byte("XXXX....")); err == nil {
		t.Error("bad magic should error")
	}
	good := EncodeWorld([]Region{{Index: 1, Spec: fallbackSpec()}})
	if _, err := DecodeWorld(good[:len(good)-3]); err == nil {
		t.Error("truncated should error")
	}
	if _, err := LoadWorldFile(filepath.Join(t.TempDir(), "nope.rwld")); !os.IsNotExist(err) {
		t.Errorf("missing file should report not-exist, got %v", err)
	}
}

// A tinted mesh placement overrides the model's baked material; an untinted one
// keeps it (so a plain crystal stays its pretty cyan).
func TestMeshMaterialOverride(t *testing.T) {
	red := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "mesh", Name: "crystal", Y: 1, Z: 5, R: 2, Color: [3]float64{0.9, 0.1, 0.1}},
	}})
	plain := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "mesh", Name: "crystal", Y: 1, Z: 5, R: 2},
	}})
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	rh, ok1 := red.Objects[0].Intersect(ray, 1e-9, 1e9)
	ph, ok2 := plain.Objects[0].Intersect(ray, 1e-9, 1e9)
	if !ok1 || !ok2 {
		t.Fatal("both crystals should be hit")
	}
	if rh.Mat.Color != (raytrace.Vec3{X: 0.9, Y: 0.1, Z: 0.1}) {
		t.Errorf("tinted crystal should be red, got %v", rh.Mat.Color)
	}
	if ph.Mat.Color == (raytrace.Vec3{X: 0.9, Y: 0.1, Z: 0.1}) {
		t.Error("untinted crystal should keep its baked material, not the red tint")
	}
}
