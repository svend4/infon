package raytrace

import (
	"math"
	"testing"
)

func TestProjectInvertsRayGeneration(t *testing.T) {
	cam := Camera{Pos: Vec3{1, 2, 5}, Yaw: 0.3, Pitch: -0.15, FOV: 1.0}
	b := cam.basis(64, 48)
	for _, s := range [][2]float64{{10, 12}, {32, 24}, {50, 40}} {
		r := b.ray(s[0], s[1])
		p := r.At(7) // a world point along that ray
		px, py, ok := Project(cam, p, 64, 48)
		if !ok {
			t.Fatalf("point should project on-screen, sample (%.0f,%.0f)", s[0], s[1])
		}
		if math.Abs(px-s[0]) > 1e-6 || math.Abs(py-s[1]) > 1e-6 {
			t.Errorf("Project not the inverse of ray: got (%.4f,%.4f) want (%.1f,%.1f)", px, py, s[0], s[1])
		}
	}
	// a point behind the camera does not project.
	if _, _, ok := Project(cam, cam.Pos.Sub(b.forward.Scale(2)), 64, 48); ok {
		t.Error("a point behind the camera must not project")
	}
}

func TestTemporalReprojectionReducesNoiseStaticView(t *testing.T) {
	scene := func() *Scene {
		return &Scene{
			Objects: []Object{
				Plane{Y: 0, Size: 1, C1: Vec3{0.8, 0.8, 0.8}, C2: Vec3{0.8, 0.8, 0.8}},
				Sphere{Center: Vec3{0, 4, 0}, Radius: 0.8, Mat: Material{Emit: Vec3{20, 20, 20}}},
			},
		}
	}
	cam := Camera{Pos: Vec3{0, 2, 6}, Yaw: math.Pi, Pitch: -0.35, FOV: 1.0}
	ref := PathRender(scene(), cam, 28, 28, PathOptions{Samples: 400, MaxDepth: 4, Seed: 9})

	tr := NewTemporalReprojector(28, 28, 0.5)
	opt := PathOptions{Samples: 2, MaxDepth: 4, Seed: 1}
	sc := scene()
	e1 := meanErr(tr.Frame(sc, cam, opt), ref) // first frame: fresh only
	var eN float64
	for i := 0; i < 8; i++ {
		eN = meanErr(tr.Frame(sc, cam, opt), ref) // history accumulates (static view)
	}
	if eN >= e1 {
		t.Errorf("temporal reuse should reduce error on a static view: %.0f -> %.0f", e1, eN)
	}
}
