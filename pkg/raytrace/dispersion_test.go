package raytrace

import (
	"math"
	"testing"
)

func TestRefractBendsByIOR(t *testing.T) {
	d := Vec3{0.3, 0, 1}.Norm()
	n := Vec3{0, 0, -1} // faces against d
	t1, ok1 := refract(d, n, 1/1.4)
	t2, ok2 := refract(d, n, 1/1.7)
	if !ok1 || !ok2 {
		t.Fatal("near-normal incidence should refract, not TIR")
	}
	if t1.Sub(t2).Len() < 1e-6 {
		t.Errorf("different IOR must bend to different directions: %v vs %v", t1, t2)
	}
}

func TestDispersiveGlassRendersChroma(t *testing.T) {
	scene := &Scene{
		Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Glass: 1.5, Disperse: 0.12}}},
		SkyTop:    Vec3{1, 1, 1},
		SkyBottom: Vec3{0.2, 0.2, 0.2}, // a gradient so the bent channels diverge
	}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: 1.0}
	img := PathRender(scene, cam, 24, 24, PathOptions{Samples: 32, MaxDepth: 8, Seed: 1})
	maxChroma := 0
	bnd := img.Bounds()
	for y := bnd.Min.Y; y < bnd.Max.Y; y++ {
		for x := bnd.Min.X; x < bnd.Max.X; x++ {
			r, _, b, _ := img.At(x, y).RGBA()
			ch := int(r>>8) - int(b>>8)
			if ch < 0 {
				ch = -ch
			}
			if ch > maxChroma {
				maxChroma = ch
			}
		}
	}
	if maxChroma < 8 {
		t.Errorf("dispersive glass should show colour separation, max |R-B|=%d", maxChroma)
	}
}
