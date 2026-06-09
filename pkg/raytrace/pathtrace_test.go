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
