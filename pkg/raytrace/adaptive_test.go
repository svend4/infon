package raytrace

import (
	"math"
	"testing"
)

func TestAdaptiveStopsEarlyOnFlatRegions(t *testing.T) {
	// A constant-colour environment: every ray returns the same value, so every
	// pixel converges immediately and stops near the minimum sample count.
	flat := &Scene{SkyTex: SolidTex{C: Vec3{0.3, 0.6, 0.4}}}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: 1.0}
	_, counts := AdaptiveRender(flat, cam, 16, 16, PathOptions{Samples: 64, MaxDepth: 2, Seed: 1}, 0.04)
	for i, c := range counts {
		if c > 9 {
			t.Fatalf("flat pixel %d used %d samples, expected ~minimum", i, c)
		}
	}
}

func TestAdaptiveSpendsMoreOnNoise(t *testing.T) {
	// A floor lit only by a small emitter, sampled naively: noisy pixels must draw
	// more than the minimum somewhere.
	noisy := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.8, 0.8, 0.8}, C2: Vec3{0.8, 0.8, 0.8}},
			Sphere{Center: Vec3{0, 4, 0}, Radius: 0.6, Mat: Material{Emit: Vec3{25, 25, 25}}},
		},
	}
	cam := Camera{Pos: Vec3{0, 2, 6}, Yaw: math.Pi, Pitch: -0.35, FOV: 1.0}
	_, counts := AdaptiveRender(noisy, cam, 24, 24, PathOptions{Samples: 64, MaxDepth: 4, Seed: 1}, 0.02)
	maxC := 0
	for _, c := range counts {
		if c > maxC {
			maxC = c
		}
	}
	if maxC <= 8 {
		t.Errorf("noisy scene should use more than the minimum somewhere, got max %d", maxC)
	}
}
