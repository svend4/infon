package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func TestFlyCamWalk(t *testing.T) {
	c := FlyCam{} // yaw 0 -> heading +Z, right +X
	c.Walk(2, 0)
	if c.Pos.Sub(raytrace.Vec3{X: 0, Y: 0, Z: 2}).Len() > 1e-9 {
		t.Errorf("forward walk -> %v, want (0,0,2)", c.Pos)
	}
	c = FlyCam{}
	c.Walk(0, 1)
	if c.Pos.Sub(raytrace.Vec3{X: 1, Y: 0, Z: 0}).Len() > 1e-9 {
		t.Errorf("strafe -> %v, want (1,0,0)", c.Pos)
	}
	// walking never leaves the ground plane.
	c = FlyCam{Yaw: 1.0, Pitch: 0.5}
	c.Walk(3, -1)
	if math.Abs(c.Pos.Y) > 1e-9 {
		t.Errorf("walking should stay on the ground, Y=%v", c.Pos.Y)
	}
}

func TestFlyCamTurnClampsPitch(t *testing.T) {
	c := FlyCam{}
	c.Turn(0, 5) // way past the limit
	if c.Pitch > 1.2+1e-9 {
		t.Errorf("pitch should clamp to 1.2, got %v", c.Pitch)
	}
	c.Turn(0, -10)
	if c.Pitch < -1.2-1e-9 {
		t.Errorf("pitch should clamp to -1.2, got %v", c.Pitch)
	}
}

// Grow places the brain-authored props at the requested spot and the world
// renders (a light is included so it is lit).
func TestWorldGrowPlacesPropsAhead(t *testing.T) {
	w := NewWorld()
	at := raytrace.Vec3{X: 0, Y: 0, Z: 12}
	n, err := w.Grow(brain.Local{}, "a studio scene", at)
	if err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Fatalf("expected several props, got %d", n)
	}
	if w.Chunks() != 1 || w.Props() != n {
		t.Errorf("chunks=%d props=%d", w.Chunks(), w.Props())
	}
	scene := w.Scene()
	// floor + props present, and at least one emissive prop (a light).
	if len(scene.Objects) != n+1 {
		t.Errorf("scene has %d objects, want %d (floor + %d props)", len(scene.Objects), n+1, n)
	}
	hasLight, nearAt := false, 0
	for _, o := range scene.Objects {
		sp, ok := o.(raytrace.Sphere)
		if !ok {
			continue
		}
		if sp.Mat.Emit.LenSq() > 0 {
			hasLight = true
		}
		if math.Abs(sp.Center.Z-at.Z) < 8 {
			nearAt++
		}
	}
	if !hasLight {
		t.Error("a grown chunk should include a light")
	}
	if nearAt < n {
		t.Errorf("props should be placed near `at` (z=%.0f), only %d/%d were", at.Z, nearAt, n)
	}
}

// Growing again extends the world (more props, more chunks).
func TestWorldGrowExtends(t *testing.T) {
	w := NewWorld()
	a, _ := w.Grow(brain.Local{}, "studio", raytrace.Vec3{Z: 10})
	b, _ := w.Grow(brain.Local{}, "a gold sphere", raytrace.Vec3{Z: 25})
	if w.Chunks() != 2 || w.Props() != a+b {
		t.Errorf("after two grows: chunks=%d props=%d (want 2, %d)", w.Chunks(), w.Props(), a+b)
	}
}
