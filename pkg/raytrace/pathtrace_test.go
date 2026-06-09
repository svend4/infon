package raytrace

import (
	"math"
	"testing"
)

func TestPathRenderIsDeterministic(t *testing.T) {
	sc := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.7, 0.7, 0.7}, C2: Vec3{0.7, 0.7, 0.7}},
			Sphere{Center: Vec3{0, 1, 0}, Radius: 1, Mat: Material{Color: Vec3{0.8, 0.3, 0.3}}},
		},
		SkyTop: Vec3{1, 1, 1}, SkyBottom: Vec3{1, 1, 1},
	}
	cam := Camera{Pos: Vec3{0, 1.5, 5}, Yaw: math.Pi, Pitch: -0.1, FOV: 1.0}
	opt := PathOptions{Samples: 8, MaxDepth: 4, Seed: 42}
	a := PathRender(sc, cam, 24, 24, opt)
	b := PathRender(sc, cam, 24, 24, opt)
	r1, g1, b1, _ := a.At(12, 14).RGBA()
	r2, g2, b2, _ := b.At(12, 14).RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 {
		t.Error("path render must be deterministic for a fixed seed")
	}
}

func TestPathSkyLightsTheScene(t *testing.T) {
	// No point light at all: a white sky alone must illuminate a diffuse floor via
	// path tracing (global illumination), so a floor pixel is clearly non-black.
	sc := &Scene{
		Objects: []Object{Plane{Y: 0, Size: 1, C1: Vec3{0.8, 0.8, 0.8}, C2: Vec3{0.8, 0.8, 0.8}}},
		SkyTop:  Vec3{1, 1, 1}, SkyBottom: Vec3{1, 1, 1},
	}
	cam := Camera{Pos: Vec3{0, 2, 4}, Pitch: -0.5, FOV: 1.0}
	img := PathRender(sc, cam, 24, 24, PathOptions{Samples: 16, MaxDepth: 3, Seed: 1})
	r, g, b, _ := img.At(12, 20).RGBA() // lower part of the frame = floor
	if r>>8 < 60 || g>>8 < 60 || b>>8 < 60 {
		t.Errorf("sky should light the floor, got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}
}

func TestNEELightsSmallEmitterAtLowSPP(t *testing.T) {
	build := func() *Scene {
		return &Scene{
			Objects: []Object{
				Plane{Y: 0, Size: 1, C1: Vec3{0.8, 0.8, 0.8}, C2: Vec3{0.8, 0.8, 0.8}},
				Sphere{Center: Vec3{0, 3, 0}, Radius: 0.4, Mat: Material{Emit: Vec3{20, 20, 20}}},
			},
			// Black sky: the small emissive sphere is the only light, so naive path
			// tracing rarely finds it at low spp while NEE samples it every bounce.
		}
	}
	cam := Camera{Pos: Vec3{0, 2, 5}, Yaw: math.Pi, Pitch: -0.35, FOV: 1.0}
	naive := PathRender(build(), cam, 32, 32, PathOptions{Samples: 8, MaxDepth: 4, Seed: 7, NEE: false})
	nee := PathRender(build(), cam, 32, 32, PathOptions{Samples: 8, MaxDepth: 4, Seed: 7, NEE: true})
	nr, ng, nb, _ := naive.At(16, 24).RGBA()
	er, eg, eb, _ := nee.At(16, 24).RGBA()
	if lum(er, eg, eb) <= lum(nr, ng, nb) {
		t.Errorf("NEE floor (%.0f) should beat naive (%.0f) at 8 spp", lum(er, eg, eb), lum(nr, ng, nb))
	}
}

func TestMISAndNEEAreUnbiased(t *testing.T) {
	build := func() *Scene {
		return &Scene{
			Objects: []Object{
				Plane{Y: 0, Size: 1, C1: Vec3{0.7, 0.7, 0.7}, C2: Vec3{0.7, 0.7, 0.7}},
				Sphere{Center: Vec3{0, 4, 0}, Radius: 1.0, Mat: Material{Emit: Vec3{8, 8, 8}}},
			},
		}
	}
	cam := Camera{Pos: Vec3{0, 2, 6}, Yaw: math.Pi, Pitch: -0.3, FOV: 1.0}
	avg := func(opt PathOptions) float64 {
		img := PathRender(build(), cam, 28, 28, opt)
		var s float64
		n := 0
		for y := 14; y < 26; y++ {
			for x := 8; x < 20; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				s += lum(r, g, b)
				n++
			}
		}
		return s / float64(n)
	}
	ref := avg(PathOptions{Samples: 300, MaxDepth: 4, Seed: 1}) // naive ground truth
	mis := avg(PathOptions{Samples: 64, MaxDepth: 4, Seed: 2, MIS: true})
	nee := avg(PathOptions{Samples: 64, MaxDepth: 4, Seed: 3, NEE: true})
	tol := ref*0.2 + 300 // generous Monte-Carlo tolerance
	if math.Abs(mis-ref) > tol {
		t.Errorf("MIS mean %.0f vs naive %.0f (tol %.0f) - biased?", mis, ref, tol)
	}
	if math.Abs(nee-ref) > tol {
		t.Errorf("NEE mean %.0f vs naive %.0f (tol %.0f) - biased?", nee, ref, tol)
	}
}

func TestPathEmissiveIsBright(t *testing.T) {
	sc := &Scene{
		Objects: []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Emit: Vec3{1, 1, 1}}}},
		SkyTop:  Vec3{0, 0, 0}, SkyBottom: Vec3{0, 0, 0},
	}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: 1.0}
	img := PathRender(sc, cam, 16, 16, PathOptions{Samples: 4, MaxDepth: 2, Seed: 5})
	r, _, _, _ := img.At(8, 8).RGBA()
	if r>>8 < 200 {
		t.Errorf("emissive sphere centre should be bright, got R=%d", r>>8)
	}
}
