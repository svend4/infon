package raydir

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func baseScene() brain.SceneSpec {
	return brain.SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: [3]float64{0.4, 0.5, 0.8}, SkyBot: [3]float64{0.8, 0.8, 0.9},
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: [3]float64{0.5, 0.5, 0.5}},
			{Kind: "sphere", X: -1, Y: 1, Z: 3, R: 1, Color: [3]float64{0.8, 0.3, 0.3}},
			{Kind: "sphere", X: 1, Y: 1, Z: 3, R: 1, Color: [3]float64{0.3, 0.3, 0.8}},
		},
		Name: "base",
	}
}

func TestSceneDeltaRoundTrip(t *testing.T) {
	prev := baseScene()
	next := baseScene()
	next.Objects[1].X = 0.5 // one sphere moves
	d, ok := DiffScene(prev, next)
	if !ok {
		t.Fatal("a one-object move should be worth a delta")
	}
	got := ApplyScene(prev, d)
	if !reflect.DeepEqual(got, next) {
		t.Fatalf("delta round-trip mismatch:\n got  %+v\n want %+v", got, next)
	}
	if len(d.Changes) != 1 {
		t.Errorf("only one object changed, got %d changes", len(d.Changes))
	}
}

func TestSceneDeltaMuchSmallerThanFull(t *testing.T) {
	prev := baseScene()
	next := baseScene()
	next.Objects[2].Z = 4 // a small move
	d, _ := DiffScene(prev, next)
	if SceneBytes(d) >= SceneBytes(next) {
		t.Errorf("a delta (%d B) should be far smaller than the full scene (%d B)", SceneBytes(d), SceneBytes(next))
	}
}

func TestSceneDeltaAddRemoveAndSky(t *testing.T) {
	prev := baseScene()
	// add an object and change the sky
	added := baseScene()
	added.Objects = append(added.Objects, brain.ObjSpec{Kind: "sphere", Y: 2, Z: 5, R: 0.5, Emit: [3]float64{5, 5, 5}})
	added.SkyTop = [3]float64{0.1, 0.1, 0.2}
	d, _ := DiffScene(prev, added)
	if got := ApplyScene(prev, d); !reflect.DeepEqual(got, added) {
		t.Fatal("add + sky-change delta did not round-trip")
	}
	// remove two objects
	removed := baseScene()
	removed.Objects = removed.Objects[:1]
	d2, _ := DiffScene(added, removed)
	if got := ApplyScene(added, d2); !reflect.DeepEqual(got, removed) {
		t.Fatal("remove delta did not round-trip")
	}
}

func TestSceneStreamBandwidth(t *testing.T) {
	var s SceneStream
	scene := baseScene()
	full := 0
	for f := 0; f < 30; f++ {
		scene.Objects[1].X = float64(f) * 0.1 // the sphere orbits
		s.Push(scene)
		full += SceneBytes(scene)
	}
	if s.Frames != 30 || s.Keys != 1 {
		t.Errorf("expected 30 frames with 1 keyframe, got %d frames %d keys", s.Frames, s.Keys)
	}
	if s.Bytes >= full {
		t.Errorf("streaming deltas (%d B) should beat sending full scenes (%d B)", s.Bytes, full)
	}
}
