package fold

import (
	"math"
	"testing"

	t7 "github.com/svend4/infon/pkg/tangram7"
)

func TestRotZIdentityAndMove(t *testing.T) {
	f := []Face{{Pts: []Pt3{{X: 2, Y: 0, Z: 1}, {X: 0, Y: 2, Z: 0}, {X: 0, Y: 0, Z: 0}}, Color: "red", Shade: 1}}
	if z := rotZ(f, 0); len(z) != len(f) || z[0].Pts[0] != f[0].Pts[0] {
		t.Error("yaw 0 must be identity")
	}
	r := rotZ(f, math.Pi/2)
	if len(r) != 1 || len(r[0].Pts) != 3 {
		t.Fatal("shape changed")
	}
	if r[0].Pts[0].Z != 1 {
		t.Error("rotation about z must preserve z")
	}
	moved := false
	for i := range r[0].Pts {
		if math.Abs(r[0].Pts[i].X-f[0].Pts[i].X) > 1e-9 {
			moved = true
		}
	}
	if !moved {
		t.Error("rotation should move points in xy")
	}
}

func TestRenderCam(t *testing.T) {
	faces := Frame(t7.Square(), 1)
	if RenderCam(faces, 120, 120, 0) == nil || RenderCam(faces, 120, 120, math.Pi/3) == nil {
		t.Fatal("nil render")
	}
}
