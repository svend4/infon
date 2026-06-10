package raytrace

import (
	"math"
	"testing"
)

func TestPrincipledMetalReflects(t *testing.T) {
	// metallic = 1: behaves like a metal mirror, reflecting the green environment.
	scene := &Scene{
		Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 1, 1}, Principled: true, Metal: 1, Rough: 0.05}}},
		SkyTop:    Vec3{0, 1, 0},
		SkyBottom: Vec3{0, 1, 0},
	}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: 1.0}
	img := PathRender(scene, cam, 24, 24, PathOptions{Samples: 24, MaxDepth: 3, Seed: 1})
	r, g, b, _ := img.At(12, 12).RGBA()
	if g>>8 < 130 || g <= r || g <= b {
		t.Errorf("principled metal should reflect green env: R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}
}

func TestPrincipledDielectricIsBrightDiffuse(t *testing.T) {
	// metallic = 0: a bright diffuse surface under a white sky (not a mirror).
	scene := &Scene{
		Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{0.8, 0.8, 0.8}, Principled: true, Metal: 0, Rough: 0.6}}},
		SkyTop:    Vec3{1, 1, 1},
		SkyBottom: Vec3{1, 1, 1},
	}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: 1.0}
	img := PathRender(scene, cam, 64, 24, PathOptions{Samples: 64, MaxDepth: 3, Seed: 2})
	r, g, b, _ := img.At(12, 12).RGBA()
	if r>>8 < 120 || g>>8 < 120 || b>>8 < 120 {
		t.Errorf("principled dielectric under white sky should be bright: R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}
}
