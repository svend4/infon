package raytrace

import "testing"

func sum3(v Vec3) float64 { return v.X + v.Y + v.Z }

func TestEmissiveIgnoresLighting(t *testing.T) {
	// An emissive sphere with no light and zero ambient still renders bright.
	sc := &Scene{
		Objects: []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Emit: Vec3{1, 1, 1}}}},
		Light:   Vec3{0, 0, 0},
		Ambient: 0,
	}
	c := sc.Trace(Ray{Origin: Vec3{0, 0, 5}, Dir: Vec3{0, 0, -1}})
	if c.X < 0.9 || c.Y < 0.9 || c.Z < 0.9 {
		t.Errorf("emissive sphere should be ~white, got %+v", c)
	}
}

func TestSpecularBrightensHighlight(t *testing.T) {
	// Light and camera coincide, so the central hit is exactly on the highlight:
	// a glossy sphere must be brighter there than a purely matte one.
	build := func(spec float64) Vec3 {
		sc := &Scene{
			Objects:  []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{0.3, 0.3, 0.3}, Spec: spec, Shine: 32}}},
			Light:    Vec3{0, 0, 5},
			LightInt: 1,
			Ambient:  0.1,
		}
		return sc.Trace(Ray{Origin: Vec3{0, 0, 5}, Dir: Vec3{0, 0, -1}})
	}
	if sum3(build(1)) <= sum3(build(0))+1e-6 {
		t.Errorf("specular highlight (%.3f) should beat matte (%.3f)", sum3(build(1)), sum3(build(0)))
	}
}

func TestGlassTransmitsLight(t *testing.T) {
	bright := &Scene{
		Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Glass: 1.5}}},
		SkyTop:    Vec3{1, 1, 1},
		SkyBottom: Vec3{1, 1, 1},
		MaxBounce: 5,
	}
	black := &Scene{
		Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{0, 0, 0}}}},
		SkyTop:    Vec3{1, 1, 1},
		SkyBottom: Vec3{1, 1, 1},
		MaxBounce: 5,
	}
	r := Ray{Origin: Vec3{0, 0, 5}, Dir: Vec3{0, 0, -1}}
	g := sum3(bright.Trace(r))
	b := sum3(black.Trace(r))
	if g < 0.5 {
		t.Errorf("glass should transmit the bright background, got %.3f", g)
	}
	if g <= b {
		t.Errorf("glass (%.3f) must be brighter than a black sphere (%.3f)", g, b)
	}
}

func TestSmoothNormalsDifferFromFlat(t *testing.T) {
	a, b, c := Vec3{-1, -1, 0}, Vec3{1, -1, 0}, Vec3{0, 1, 0}
	r := Ray{Origin: Vec3{0, -0.33, -1}, Dir: Vec3{0, 0, 1}}
	flat := Triangle{A: a, B: b, C: c}
	smooth := Triangle{A: a, B: b, C: c,
		Na: Vec3{-1, 0, 1}.Norm(), Nb: Vec3{1, 0, 1}.Norm(), Nc: Vec3{0, 1, 1}.Norm()}
	hf, ok1 := flat.Intersect(r, 1e-4, 1e9)
	hs, ok2 := smooth.Intersect(r, 1e-4, 1e9)
	if !ok1 || !ok2 {
		t.Fatal("both triangles should be hit")
	}
	if hf.N.Sub(hs.N).Len() < 0.2 {
		t.Errorf("smooth normal %+v should differ from flat %+v", hs.N, hf.N)
	}
}

func TestSoftShadowExtremes(t *testing.T) {
	// Soft-shadow path must still return the right extremes: fully lit vs fully
	// occluded.
	sc := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{1, 1, 1}, C2: Vec3{1, 1, 1}},
			Sphere{Center: Vec3{0, 2, 0}, Radius: 1.2, Mat: Material{Color: Vec3{1, 1, 1}}},
		},
		Light:         Vec3{0, 10, 0},
		LightInt:      1,
		Ambient:       0.1,
		LightRadius:   0.5,
		ShadowSamples: 16,
	}
	under := sum3(sc.Trace(Ray{Origin: Vec3{4, 4, 0}, Dir: Vec3{-1, -1, 0}.Norm()}))
	far := sum3(sc.Trace(Ray{Origin: Vec3{8, 5, 0}, Dir: Vec3{0, -1, 0}}))
	if under >= far {
		t.Errorf("soft-shadowed floor (%.3f) should be darker than lit floor (%.3f)", under, far)
	}
}

func TestAmbientOcclusionOnlyDarkens(t *testing.T) {
	// In a fully open scene AO is ~1 (no change); in a crease it can only darken.
	open := &Scene{Objects: []Object{Plane{Y: 0, Size: 1, C1: Vec3{1, 1, 1}, C2: Vec3{1, 1, 1}}}, Ambient: 0.6}
	if got := open.ao(Vec3{0, 0, 0}, Vec3{0, 1, 0}); got < 0.98 {
		t.Errorf("AO in the open should be ~1, got %.3f", got)
	}
	occl := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{1, 1, 1}, C2: Vec3{1, 1, 1}},
			Sphere{Center: Vec3{0, 1, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 1, 1}}},
		},
		Ambient: 0.6, AOSamples: 32, AORadius: 3,
	}
	// a floor point right beside the sphere's contact: some hemisphere rays hit it.
	if got := occl.ao(Vec3{0.9, 0, 0}, Vec3{0, 1, 0}); got >= 1 {
		t.Errorf("AO next to the sphere should be < 1, got %.3f", got)
	}
}
