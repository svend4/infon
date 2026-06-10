package raytrace

import "testing"

func arSimpleScene() *Scene {
	return &Scene{
		SkyTop: Vec3{0.4, 0.55, 0.85}, SkyBottom: Vec3{0.85, 0.88, 0.95},
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.7, 0.7, 0.7}, C2: Vec3{0.5, 0.5, 0.5}},
			Sphere{Center: Vec3{-1, 1, 3}, Radius: 1, Mat: Material{Color: Vec3{0.8, 0.3, 0.3}}},
			Sphere{Center: Vec3{1, 1, 3}, Radius: 1, Mat: Material{Color: Vec3{0.3, 0.3, 0.8}}},
			Sphere{Center: Vec3{0, 6, 2}, Radius: 1.5, Mat: Material{Emit: Vec3{15, 15, 14}}},
		},
	}
}

func arCausticScene() *Scene {
	return &Scene{
		SkyTop: Vec3{0.05, 0.06, 0.09}, SkyBottom: Vec3{0.08, 0.09, 0.12},
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.7, 0.7, 0.7}, C2: Vec3{0.5, 0.5, 0.5}},
			Sphere{Center: Vec3{0, 1, 3}, Radius: 1, Mat: Material{Glass: 1.5}},
			Sphere{Center: Vec3{0, 7, 3}, Radius: 0.3, Mat: Material{Emit: Vec3{400, 400, 380}}},
		},
	}
}

func TestPickIntegratorByDifficulty(t *testing.T) {
	cam := Camera{Pos: Vec3{0, 2.5, -2}, Pitch: -0.3, FOV: 1.0472}
	const tol = 0.065
	s := PickIntegrator(arSimpleScene(), cam, 80, 60, tol)
	c := PickIntegrator(arCausticScene(), cam, 80, 60, tol)
	t.Logf("simple noise=%.4f pick=%s ; caustic noise=%.4f pick=%s", s.Noise, s.Name, c.Noise, c.Name)
	if c.Noise <= s.Noise {
		t.Errorf("the caustic scene should probe noisier (%.4f) than the simple one (%.4f)", c.Noise, s.Noise)
	}
	if s.Name != "path" {
		t.Errorf("a simple diffuse scene should pick the path tracer, got %q (noise %.4f)", s.Name, s.Noise)
	}
	if c.Name != "mlt" {
		t.Errorf("a caustic scene should pick mlt, got %q (noise %.4f)", c.Name, c.Noise)
	}
}

func TestRenderBestDispatches(t *testing.T) {
	cam := Camera{Pos: Vec3{0, 2.5, -2}, Pitch: -0.3, FOV: 1.0472}
	opt := PathOptions{Samples: 8, MaxDepth: 6, Seed: 1, NEE: true, MIS: true}
	if _, name := RenderBest(arSimpleScene(), cam, 64, 48, opt); name != "path" {
		t.Errorf("simple scene should render with path, got %q", name)
	}
	img, name := RenderBest(arCausticScene(), cam, 64, 48, opt)
	if name != "mlt" {
		t.Errorf("caustic scene should render with mlt, got %q", name)
	}
	if img == nil || img.Bounds().Dx() != 64 {
		t.Error("RenderBest should return the rendered image")
	}
}
