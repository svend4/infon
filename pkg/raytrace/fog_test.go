package raytrace

import "testing"

func TestFogFadesWithDistance(t *testing.T) {
	build := func() *Scene {
		return &Scene{
			Objects:    []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{0.9, 0.9, 0.9}}}},
			Light:      Vec3{5, 5, 5},
			LightInt:   1,
			Ambient:    0.2,
			FogDensity: 0.1,
			FogColor:   Vec3{0.1, 0.3, 1.0}, // bluish fog
		}
	}
	near := build().Trace(Ray{Origin: Vec3{0, 0, 3}, Dir: Vec3{0, 0, -1}}) // hit ~2 away
	far := build().Trace(Ray{Origin: Vec3{0, 0, 15}, Dir: Vec3{0, 0, -1}}) // hit ~14 away
	blueNear := near.Z - near.X
	blueFar := far.Z - far.X
	if blueFar <= blueNear {
		t.Errorf("distant surface should be foggier/bluer: near=%.3f far=%.3f", blueNear, blueFar)
	}
	// With fog off the two should match (no distance tint).
	noFog := &Scene{Objects: []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{0.9, 0.9, 0.9}}}}, Light: Vec3{5, 5, 5}, LightInt: 1, Ambient: 0.2}
	a := noFog.Trace(Ray{Origin: Vec3{0, 0, 3}, Dir: Vec3{0, 0, -1}})
	b := noFog.Trace(Ray{Origin: Vec3{0, 0, 15}, Dir: Vec3{0, 0, -1}})
	if a.Sub(b).Len() > 1e-9 {
		t.Errorf("without fog, distance must not change the colour")
	}
}
