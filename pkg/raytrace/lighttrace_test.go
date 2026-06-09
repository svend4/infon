package raytrace

import "testing"

func TestIsDiffuseMat(t *testing.T) {
	if !isDiffuseMat(Material{Color: Vec3{1, 1, 1}}) {
		t.Error("plain colour should be diffuse")
	}
	for _, m := range []Material{{Glass: 1.5}, {Metal: 1}, {Reflect: 1}, {Film: 300}, {SSS: 1.4}, {Principled: true}} {
		if isDiffuseMat(m) {
			t.Errorf("non-diffuse material treated as diffuse: %+v", m)
		}
	}
}

// The light tracer and the path tracer solve the same transport, so on a diffuse
// scene (light out of frame, black sky) their images agree in overall brightness.
// This validates the camera-importance normalization.
func TestLightTraceMatchesPathTracer(t *testing.T) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	scene := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: v(0.6, 0.6, 0.6), C2: v(0.6, 0.6, 0.6)},
			Sphere{Center: v(0, 1, 3), Radius: 1, Mat: Material{Color: v(0.7, 0.5, 0.5)}},
			Sphere{Center: v(0, 9, 2), Radius: 0.5, Mat: Material{Emit: v(60, 60, 60)}}, // high, out of view
		},
		SkyTop: v(0, 0, 0), SkyBottom: v(0, 0, 0),
	}
	cam := Camera{Pos: v(0, 2, -3), Pitch: -0.25, FOV: 1.0}
	const w, h = 64, 48
	pt := ppmMeanLum(PathRender(scene, cam, w, h, PathOptions{Samples: 300, MaxDepth: 3, Seed: 9, NEE: true}))
	lt := ppmMeanLum(LightTraceRender(scene, cam, w, h, LightTraceOptions{Paths: 1200000, MaxBounce: 3, Seed: 1}))
	if pt < 5 || lt < 5 {
		t.Fatalf("both should be lit: pt=%.2f lt=%.2f", pt, lt)
	}
	if ratio := lt / pt; ratio < 0.88 || ratio > 1.12 {
		t.Errorf("light tracer should match the path tracer, ratio LT/PT=%.3f (pt=%.2f lt=%.2f)", ratio, pt, lt)
	}
}

// Rigorous unbiasedness check in linear space (no tone-map confound): the light
// tracer's mean radiance matches the path tracer's on the same diffuse scene.
func TestLightTraceUnbiasedLinear(t *testing.T) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	scene := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: v(0.6, 0.6, 0.6), C2: v(0.6, 0.6, 0.6)},
			Sphere{Center: v(0, 1, 3), Radius: 1, Mat: Material{Color: v(0.7, 0.5, 0.5)}},
			Sphere{Center: v(0, 9, 2), Radius: 0.5, Mat: Material{Emit: v(60, 60, 60)}},
		},
		SkyTop: v(0, 0, 0), SkyBottom: v(0, 0, 0),
	}
	cam := Camera{Pos: v(0, 2, -3), Pitch: -0.25, FOV: 1.0}
	const w, h, spp = 64, 48, 300
	refMean := meanLinLum(pathSum(scene, cam, w, h, PathOptions{Samples: spp, MaxDepth: 3, Seed: 9, NEE: true})) / float64(spp)
	ltMean := meanLinLum(lightTraceLinear(scene, cam, w, h, LightTraceOptions{Paths: 3000000, MaxBounce: 3, Seed: 1}))
	if ratio := ltMean / refMean; ratio < 0.93 || ratio > 1.07 {
		t.Errorf("light tracer linear mean should match the path tracer: ratio=%.3f (lt=%.4f ref=%.4f)", ratio, ltMean, refMean)
	}
}
