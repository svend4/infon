package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// The flyer closes on the walker over time.
func TestFlyerPursues(t *testing.T) {
	f := NewFlyer(raytrace.Vec3{X: 0, Z: 0})
	player := raytrace.Vec3{X: 30, Z: 0}
	before := f.Pos.Sub(player).Len()
	for i := 0; i < 60; i++ {
		f.Step(0.1, player)
	}
	after := f.Pos.Sub(player).Len()
	if after >= before {
		t.Errorf("the flyer should pursue (get closer): before %.1f after %.1f", before, after)
	}
}

// It faces the walker as it hunts.
func TestFlyerFaces(t *testing.T) {
	f := NewFlyer(raytrace.Vec3{X: 0, Z: 0})
	f.Step(0.1, raytrace.Vec3{X: 10, Z: 0}) // player to the +X
	if math.Abs(f.Yaw-math.Pi/2) > 0.3 {
		t.Errorf("a flyer chasing toward +X should face ~pi/2, got %.2f", f.Yaw)
	}
}

// Step reports draining only within reach.
func TestFlyerDrainRange(t *testing.T) {
	f := NewFlyer(raytrace.Vec3{X: 0, Z: 0})
	if f.Step(0, raytrace.Vec3{X: 50, Z: 0}) {
		t.Error("a distant walker should not be drained")
	}
	near := NewFlyer(raytrace.Vec3{X: 0, Z: 0})
	if !near.Step(0, raytrace.Vec3{X: 1, Z: 0}) {
		t.Error("a walker within reach should be drained")
	}
}

// The flyer is a dark shape made of several lobes.
func TestFlyerObjects(t *testing.T) {
	f := NewFlyer(raytrace.Vec3{Z: 5})
	objs := f.Objects()
	if len(objs) < 5 {
		t.Fatalf("the flyer should be several lobes, got %d", len(objs))
	}
	for _, o := range objs {
		sp := o.(raytrace.Sphere)
		if sp.Mat.Color.X > 0.2 || sp.Mat.Color.Y > 0.2 || sp.Mat.Color.Z > 0.2 {
			t.Errorf("the flyer should be dark, got colour %+v", sp.Mat.Color)
		}
	}
}

// The world spawns and steps a flyer, which is in the scene and animated.
func TestWorldFlyer(t *testing.T) {
	w := NewWorld()
	base := len(w.SceneWith(nil).Objects)
	w.SpawnFlyer(raytrace.Vec3{X: 0, Z: 20})
	if !w.HasFlyer() || !w.HasAnimated() {
		t.Fatal("a world with a flyer should report it and be animated")
	}
	if got := len(w.SceneWith(nil).Objects); got <= base {
		t.Errorf("the flyer should add objects: base %d after %d", base, got)
	}
	drained := false
	for i := 0; i < 200; i++ {
		if w.StepFlyer(0.1, raytrace.Vec3{}) { // player at origin; flyer starts 20 away
			drained = true
		}
	}
	if !drained {
		t.Error("after pursuing, the flyer should reach and drain the walker")
	}
}
