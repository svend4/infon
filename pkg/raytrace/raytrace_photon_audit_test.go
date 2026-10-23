package raytrace

import (
	"math"
	"testing"
)

// causticPower is the total luminance power stored in a caustic photon map.
func causticPower(pm *PhotonMap) float64 {
	var s float64
	for _, ph := range pm.photons {
		s += lum3(ph.power)
	}
	return s
}

// Each emitter must deliver its full flux into the photon map however many other
// emitters share the photon budget: adding a second light that makes no caustics
// (it sits below the floor) must not dim the first light's caustic. Before the fix
// every photon carried flux/n instead of flux/(n/ne), so two lights halved it.
func TestAuditCausticPowerIndependentOfEmitterCount(t *testing.T) {
	light := Sphere{Center: Vec3{X: 0, Y: 5.5, Z: 0}, Radius: 0.3, Mat: Material{Emit: Vec3{X: 10, Y: 10, Z: 10}}}
	glass := Sphere{Center: Vec3{X: 0, Y: 2.5, Z: 0}, Radius: 1.5, Mat: Material{Glass: 1.5}}
	floor := Plane{Y: 0, Size: 1, C1: Vec3{X: 0.7, Y: 0.7, Z: 0.7}, C2: Vec3{X: 0.7, Y: 0.7, Z: 0.7}}
	under := Sphere{Center: Vec3{X: 0, Y: -50, Z: 0}, Radius: 0.3, Mat: Material{Emit: Vec3{X: 10, Y: 10, Z: 10}}}

	const n = 200000
	one := BuildCaustics(&Scene{Objects: []Object{floor, glass, light}}, n, 0.2, 3)
	two := BuildCaustics(&Scene{Objects: []Object{floor, glass, light, under}}, n, 0.2, 3)
	p1, p2 := causticPower(one), causticPower(two)
	t.Logf("one light: %d photons, power %.4f; two lights: %d photons, power %.4f", one.Count(), p1, two.Count(), p2)
	if p1 <= 0 {
		t.Fatal("no caustic photons from the light through the glass sphere")
	}
	if rel := math.Abs(p2-p1) / p1; rel > 0.08 {
		t.Errorf("caustic power changed by %.0f%% when a non-caustic light was added (%.4f -> %.4f)", rel*100, p1, p2)
	}
}

// When every photon becomes a caustic photon, the map must hold exactly the
// lights' total flux sum(Le*pi*A), including photon counts that are not a
// multiple of the emitter count (round-robin gives some lights one photon more).
// Two small lights sit inside a glass shell inside a diffuse shell, so each photon
// crosses glass (specular) before it lands on a diffuse surface.
func TestAuditCausticPhotonsCarryTotalFlux(t *testing.T) {
	a := Sphere{Center: Vec3{X: -0.6, Y: 0, Z: 0}, Radius: 0.1, Mat: Material{Emit: Vec3{X: 1, Y: 1, Z: 1}}}
	b := Sphere{Center: Vec3{X: 0.6, Y: 0, Z: 0}, Radius: 0.1, Mat: Material{Emit: Vec3{X: 3, Y: 3, Z: 3}}}
	shell := Sphere{Radius: 2, Mat: Material{Glass: 1.5}}
	room := Sphere{Radius: 3, Mat: Material{Color: Vec3{X: 0.5, Y: 0.5, Z: 0.5}}}
	s := &Scene{Objects: []Object{a, b, shell, room}}
	flux := func(e Sphere) float64 { return lum3(e.Mat.Emit) * math.Pi * 4 * math.Pi * e.Radius * e.Radius }
	want := flux(a) + flux(b)
	for _, n := range []int{3, 1001, 20000} {
		pm := BuildCaustics(s, n, 0.2, 5)
		got := causticPower(pm)
		t.Logf("n=%d: %d stored, power %.5f, want %.5f", n, pm.Count(), got, want)
		if n == 3 && pm.Count() != n {
			t.Fatalf("n=3: %d of 3 photons stored; the scene should store every photon", pm.Count())
		}
		// a few photons may hit the other light before any glass and be lost
		if rel := math.Abs(got-want) / want; rel > 0.01 {
			t.Errorf("n=%d: stored flux %.5f, want %.5f (%.1f%% off)", n, got, want, rel*100)
		}
	}
}
