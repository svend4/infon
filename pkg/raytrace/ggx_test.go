package raytrace

import (
	"image"
	"math"
	"testing"
)

func TestGGXMetalReflectsEnvironment(t *testing.T) {
	build := func(rough float64) *Scene {
		return &Scene{
			Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 1, 1}, Metal: 1, Rough: rough}}},
			SkyTop:    Vec3{0, 1, 0}, // uniform green environment
			SkyBottom: Vec3{0, 1, 0},
		}
	}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: 1.0}
	imgs := map[string]image.Image{
		"smooth": PathRender(build(0.03), cam, 24, 24, PathOptions{Samples: 16, MaxDepth: 3, Seed: 1}),
		"rough":  PathRender(build(0.5), cam, 24, 24, PathOptions{Samples: 16, MaxDepth: 3, Seed: 2}),
	}
	for name, img := range imgs {
		r, g, b, _ := img.At(12, 12).RGBA()
		if g>>8 < 150 || g <= r || g <= b {
			t.Errorf("%s metal should reflect the green env: R=%d G=%d B=%d", name, r>>8, g>>8, b>>8)
		}
	}
}
