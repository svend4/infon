package raytrace

import (
	"math"
	"testing"
)

func inUnit(v Vec3) bool {
	return v.X >= 0 && v.X <= 1 && v.Y >= 0 && v.Y <= 1 && v.Z >= 0 && v.Z <= 1
}

// The interference factor is a valid per-channel weight in [0,1].
func TestThinFilmInRange(t *testing.T) {
	for _, d := range []float64{50, 150, 300, 550, 800} {
		for _, c := range []float64{0.1, 0.5, 0.9, 1.0} {
			if got := thinFilmTint(c, d, 1.33); !inUnit(got) {
				t.Errorf("thinFilmTint(cos=%.1f,d=%.0f) = %v out of [0,1]", c, d, got)
			}
		}
	}
}

// Iridescence: the colour shifts with viewing angle (same film, different cosI).
func TestThinFilmVariesWithAngle(t *testing.T) {
	a := thinFilmTint(0.95, 320, 1.4)
	b := thinFilmTint(0.35, 320, 1.4)
	if a.Sub(b).Len() < 0.1 {
		t.Errorf("thin-film colour should shift with angle: %v vs %v", a, b)
	}
}

// And with thickness (same angle, different film).
func TestThinFilmVariesWithThickness(t *testing.T) {
	a := thinFilmTint(0.8, 200, 1.33)
	b := thinFilmTint(0.8, 420, 1.33)
	if a.Sub(b).Len() < 0.1 {
		t.Errorf("thin-film colour should shift with thickness: %v vs %v", a, b)
	}
}

// At the thickness where a wavelength's round trip is a half-integer number of
// waves, that channel is fully constructive (factor 1); the pi front-surface
// shift makes opd = (k-0.5)*lambda constructive. Check green peaks there.
func TestThinFilmConstructivePeak(t *testing.T) {
	const n = 1.33
	lambdaG := rgbWavelengths[1]
	// normal incidence (cosT=1): opd = 2*n*d ; want opd = 0.5*lambdaG -> first peak.
	d := 0.5 * lambdaG / (2 * n)
	g := thinFilmTint(1, d, n).Y
	if math.Abs(g-1) > 1e-6 {
		t.Errorf("green should be fully constructive at d=%.2f, got factor %.4f", d, g)
	}
}

// End to end: a thin-film sphere renders with strong hue variation across its
// surface (the angle-dependent interference), unlike a plain mirror.
func TestThinFilmRendersIridescentSphere(t *testing.T) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	scene := &Scene{
		Objects: []Object{
			Sphere{Center: v(0, 1, 0), Radius: 1, Mat: Material{Film: 320, FilmIOR: 1.4}},
			Sphere{Center: v(0, 6, -1), Radius: 1, Mat: Material{Emit: v(20, 20, 20)}},
		},
		SkyTop: v(0.3, 0.4, 0.6), SkyBottom: v(0.7, 0.75, 0.85),
	}
	cam := Camera{Pos: v(0, 1, -5), FOV: 1}
	img := PathRender(scene, cam, 64, 64, PathOptions{Samples: 24, MaxDepth: 4, Seed: 1, NEE: true})
	// the iridescent sphere should show more than one hue: measure channel spread.
	b := img.Bounds()
	var minR, minG, minB = 999, 999, 999
	var maxR, maxG, maxB int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			ri, gi, bi := int(r>>8), int(g>>8), int(bl>>8)
			if ri < minR {
				minR = ri
			}
			if gi < minG {
				minG = gi
			}
			if bi < minB {
				minB = bi
			}
			if ri > maxR {
				maxR = ri
			}
			if gi > maxG {
				maxG = gi
			}
			if bi > maxB {
				maxB = bi
			}
		}
	}
	// a plain mirror would reflect the smooth sky (low per-channel spread); the
	// thin film adds strong hue variation across the surface.
	if (maxR-minR)+(maxG-minG)+(maxB-minB) < 150 {
		t.Errorf("iridescent sphere should show wide hue variation, spread=%d", (maxR-minR)+(maxG-minG)+(maxB-minB))
	}
}
