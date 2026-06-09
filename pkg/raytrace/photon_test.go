package raytrace

import "testing"

func TestCausticsConcentrateUnderGlass(t *testing.T) {
	s := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.8, 0.8, 0.8}, C2: Vec3{0.8, 0.8, 0.8}},
			Sphere{Center: Vec3{0, 3, 0}, Radius: 1.5, Mat: Material{Glass: 1.5}},
			Sphere{Center: Vec3{0, 8, 0}, Radius: 0.5, Mat: Material{Emit: Vec3{30, 30, 30}}},
		},
	}
	pm := BuildCaustics(s, 60000, 0.3, 1)
	if pm.Count() == 0 {
		t.Fatal("expected caustic photons (light through glass onto the floor)")
	}
	near, far := 0, 0
	for _, ph := range pm.photons {
		if ph.pos.Sub(Vec3{0, 0, 0}).LenSq() < 1.0 {
			near++
		}
		if ph.pos.Sub(Vec3{5, 0, 0}).LenSq() < 1.0 {
			far++
		}
	}
	if near <= far {
		t.Errorf("caustic should concentrate under the glass: near=%d far=%d", near, far)
	}
	ef := pm.Estimate(Vec3{0, 0, 0}, Vec3{0, 1, 0}, Vec3{1, 1, 1})
	es := pm.Estimate(Vec3{5, 0, 0}, Vec3{0, 1, 0}, Vec3{1, 1, 1})
	if ef.X <= es.X {
		t.Errorf("caustic estimate should be brighter at the focus: %.3f vs %.3f", ef.X, es.X)
	}
}

func TestCausticsEmptyWithoutSpecular(t *testing.T) {
	// No specular surface -> no L S+ D paths -> no caustic photons stored.
	s := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.8, 0.8, 0.8}, C2: Vec3{0.8, 0.8, 0.8}},
			Sphere{Center: Vec3{0, 6, 0}, Radius: 0.5, Mat: Material{Emit: Vec3{20, 20, 20}}},
		},
	}
	if pm := BuildCaustics(s, 5000, 0.3, 2); pm.Count() != 0 {
		t.Errorf("no specular surface should store no caustic photons, got %d", pm.Count())
	}
}
