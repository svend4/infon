package raytrace

import (
	"math"
	"testing"
)

// Ratio tracking must give Beer-Lambert transmittance exp(-sigma*d) in a
// homogeneous medium.
func TestMediumTransmittanceBeerLambert(t *testing.T) {
	const sigma, dist = 0.8, 1.5
	m := &Medium{SigmaMax: 2.0, Density: func(Vec3) float64 { return sigma }}
	r := Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{1, 0, 0}}
	rg := newRNG(1)
	const n = 200000
	var sum float64
	for i := 0; i < n; i++ {
		sum += m.transmittance(r, 0, dist, rg)
	}
	got, want := sum/float64(n), math.Exp(-sigma*dist)
	if math.Abs(got-want) > 0.01 {
		t.Errorf("transmittance = %.4f, want exp(-sigma*d) = %.4f", got, want)
	}
}

// The estimate must not depend on the majorant (the defining property of
// delta/ratio tracking's fictitious collisions).
func TestMediumMajorantInvariance(t *testing.T) {
	const sigma, dist = 0.8, 1.5
	r := Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{1, 0, 0}}
	mean := func(sigmaMax float64, seed uint64) float64 {
		m := &Medium{SigmaMax: sigmaMax, Density: func(Vec3) float64 { return sigma }}
		rg := newRNG(seed)
		const n = 200000
		var sum float64
		for i := 0; i < n; i++ {
			sum += m.transmittance(r, 0, dist, rg)
		}
		return sum / float64(n)
	}
	want := math.Exp(-sigma * dist)
	a, b := mean(1.0, 2), mean(5.0, 3)
	if math.Abs(a-want) > 0.012 || math.Abs(b-want) > 0.012 || math.Abs(a-b) > 0.015 {
		t.Errorf("majorant should not change the mean: sigmaMax1=%.4f sigmaMax5=%.4f want=%.4f", a, b, want)
	}
}

// Delta tracking's escape probability is the transmittance.
func TestMediumEscapeProbability(t *testing.T) {
	const sigma, dist = 0.9, 1.2
	m := &Medium{SigmaMax: 3.0, Density: func(Vec3) float64 { return sigma }}
	r := Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{1, 0, 0}}
	rg := newRNG(7)
	const n = 200000
	escapes := 0
	for i := 0; i < n; i++ {
		if _, ok := m.sampleCollision(r, 0, dist, rg); !ok {
			escapes++
		}
	}
	got, want := float64(escapes)/float64(n), math.Exp(-sigma*dist)
	if math.Abs(got-want) > 0.01 {
		t.Errorf("escape fraction = %.4f, want exp(-sigma*d) = %.4f", got, want)
	}
}

// A genuinely heterogeneous (step) density: transmittance must match the analytic
// integral over the ray's chords through the dense core and the thin shell.
func TestMediumHeterogeneousStep(t *testing.T) {
	const s0, s1, ri, ro = 1.0, 0.3, 1.0, 2.0
	m := &Medium{
		Center: Vec3{0, 0, 0}, Radius: ro, SigmaMax: s0,
		Density: func(p Vec3) float64 {
			if p.Len() < ri {
				return s0
			}
			return s1
		},
	}
	r := Ray{Origin: Vec3{-3, 0, 0}, Dir: Vec3{1, 0, 0}}
	t0, t1, ok := m.span(r, 1e9)
	if !ok {
		t.Fatal("ray should cross the medium")
	}
	rg := newRNG(11)
	const n = 300000
	var sum float64
	for i := 0; i < n; i++ {
		sum += m.transmittance(r, t0, t1, rg)
	}
	// central ray: inner chord 2*ri, shell chord 2*(ro-ri).
	want := math.Exp(-(s0*2*ri + s1*2*(ro-ri)))
	if got := sum / float64(n); math.Abs(got-want) > 0.01 {
		t.Errorf("heterogeneous transmittance = %.4f, want %.4f", got, want)
	}
}

func TestHGPhaseNormalized(t *testing.T) {
	for _, g := range []float64{-0.6, 0, 0.4, 0.8} {
		const steps = 4000
		var integ float64
		for i := 0; i < steps; i++ {
			u := -1 + 2*(float64(i)+0.5)/steps // cos theta
			integ += hgPhase(u, g) * (2.0 / steps)
		}
		integ *= 2 * math.Pi // integral over phi
		if math.Abs(integ-1) > 0.02 {
			t.Errorf("g=%.1f: phase integrates to %.4f, want 1", g, integ)
		}
	}
}

func TestHGSampleMeanCosine(t *testing.T) {
	w := Vec3{0, 0, 1}
	rg := newRNG(5)
	for _, g := range []float64{0, 0.6, -0.4} {
		const n = 200000
		var sum float64
		for i := 0; i < n; i++ {
			sum += hgSample(w, g, rg).Dot(w)
		}
		if mean := sum / float64(n); math.Abs(mean-g) > 0.02 {
			t.Errorf("g=%.1f: sampled mean cosine = %.4f, want g", g, mean)
		}
	}
}

// End to end: a medium lit by an off-screen light glows from in-scattering, so a
// view that is otherwise empty (black sky) lights up when the medium is present.
func TestMediumScattersLight(t *testing.T) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	base := &Scene{
		Objects: []Object{Sphere{Center: v(4, 0, 5), Radius: 0.6, Mat: Material{Emit: v(60, 60, 60)}}},
	}
	cam := Camera{Pos: v(0, 0, 0), FOV: 1}
	opt := PathOptions{Samples: 40, MaxDepth: 6, Seed: 1, NEE: true}
	without := ppmMeanLum(PathRender(base, cam, 48, 48, opt))

	withMed := &Scene{
		Objects: base.Objects,
		Medium: &Medium{
			Center: v(0, 0, 5), Radius: 2, SigmaMax: 1.2, Albedo: v(0.9, 0.9, 0.9),
			Density: func(Vec3) float64 { return 1.2 },
		},
	}
	with := ppmMeanLum(PathRender(withMed, cam, 48, 48, opt))
	if with <= without {
		t.Errorf("medium in-scattering should brighten the view: with=%.3f without=%.3f", with, without)
	}
}
