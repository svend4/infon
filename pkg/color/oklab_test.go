package color

import (
	"math"
	"testing"
)

func TestOKLabRoundTrip(t *testing.T) {
	// Converting RGB -> OKLab -> RGB should return (almost) the same color.
	cases := []RGB{
		{0, 0, 0}, {255, 255, 255}, {255, 0, 0}, {0, 255, 0}, {0, 0, 255},
		{128, 64, 200}, {17, 200, 99}, {240, 230, 140},
	}
	for _, c := range cases {
		got := c.ToOKLab().ToRGB()
		// Allow ±2 per channel for rounding through linear/cbrt math.
		if absDiff(got.R, c.R) > 2 || absDiff(got.G, c.G) > 2 || absDiff(got.B, c.B) > 2 {
			t.Errorf("round trip %v -> %v (too far)", c, got)
		}
	}
}

func TestOKLabLightnessOrdering(t *testing.T) {
	// Black < gray < white in perceptual lightness.
	black := RGB{0, 0, 0}.ToOKLab().L
	gray := RGB{128, 128, 128}.ToOKLab().L
	white := RGB{255, 255, 255}.ToOKLab().L
	if !(black < gray && gray < white) {
		t.Errorf("lightness not ordered: black=%.3f gray=%.3f white=%.3f", black, gray, white)
	}
	if math.Abs(white-1.0) > 0.02 {
		t.Errorf("white L = %.3f, want ~1.0", white)
	}
}

func TestPerceptualDistanceBasics(t *testing.T) {
	red := RGB{255, 0, 0}
	// Distance to itself is zero.
	if d := red.PerceptualDistance(red); d != 0 {
		t.Errorf("self distance = %v, want 0", d)
	}
	// A near-red is perceptually closer than a far blue.
	nearRed := RGB{245, 10, 10}
	blue := RGB{0, 0, 255}
	if red.PerceptualDistance(nearRed) >= red.PerceptualDistance(blue) {
		t.Error("near-red should be perceptually closer to red than blue is")
	}
}

func TestPerceptualVsNaiveDistanceDiffer(t *testing.T) {
	// A known case where perceptual and raw-RGB ranking disagree: the human eye
	// is far more sensitive to green differences than blue. Just assert the two
	// metrics are not identical so we know OKLab is doing something.
	base := RGB{100, 100, 100}
	greenShift := RGB{100, 140, 100}
	blueShift := RGB{100, 100, 140}
	rgbG := base.Distance(greenShift)
	rgbB := base.Distance(blueShift)
	okG := base.PerceptualDistance(greenShift)
	okB := base.PerceptualDistance(blueShift)
	// Raw RGB treats the +40 shifts identically; OKLab should not.
	if rgbG != rgbB {
		t.Logf("note: raw RGB distances already differ (%.0f vs %.0f)", rgbG, rgbB)
	}
	if math.Abs(okG-okB) < 1e-9 {
		t.Error("OKLab distances for green vs blue shift should differ")
	}
}

func TestBlendOKLabEndpoints(t *testing.T) {
	a := RGB{10, 20, 30}
	b := RGB{200, 100, 50}
	if got := a.BlendOKLab(b, 0); got != a {
		t.Errorf("alpha 0 = %v, want %v", got, a)
	}
	if got := a.BlendOKLab(b, 1); got != b {
		t.Errorf("alpha 1 = %v, want %v", got, b)
	}
	// Midpoint must lie between the endpoints on each channel (roughly).
	mid := a.BlendOKLab(b, 0.5)
	if mid.R < 10 || mid.R > 200 {
		t.Errorf("mid R = %d out of range", mid.R)
	}
}

func absDiff(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}
