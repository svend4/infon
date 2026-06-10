package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func TestAnimateFigure8(t *testing.T) {
	o := brain.ObjSpec{Anim: "figure8", AAmp: 2.5, ASpeed: 0.3}
	if !isAnimated(o.Anim) {
		t.Fatal("figure8 should be a recognised motion")
	}
	period := 1 / o.ASpeed
	// periodicity: one full period returns to the start offset
	a := animateSpec(o, 0, 0)
	b := animateSpec(o, period, 0)
	if math.Abs(a.X-b.X) > 1e-9 || math.Abs(a.Z-b.Z) > 1e-9 {
		t.Errorf("figure8 should be periodic: %v vs %v", a, b)
	}
	maxX, maxZ, nearCenter := 0.0, 0.0, false
	for i := 0; i <= 400; i++ {
		p := animateSpec(o, float64(i)/400*period, 0)
		maxX = math.Max(maxX, math.Abs(p.X))
		maxZ = math.Max(maxZ, math.Abs(p.Z))
		if math.Hypot(p.X, p.Z) < 0.1 {
			nearCenter = true // the figure-eight pinches through its centre
		}
	}
	if math.Abs(maxX-2.5) > 0.1 {
		t.Errorf("figure8 should reach ±amp in X, got %.2f", maxX)
	}
	if maxZ > 2.5*0.5+0.05 {
		t.Errorf("figure8 Z should stay within ~half amp, got %.2f", maxZ)
	}
	if !nearCenter {
		t.Error("figure8 should cross its centre")
	}
}

func TestAnimateRosette(t *testing.T) {
	o := brain.ObjSpec{Anim: "rosette", AAmp: 2.5, ASpeed: 0.25}
	if !isAnimated(o.Anim) {
		t.Fatal("rosette should be a recognised motion")
	}
	period := 1 / o.ASpeed
	minR, maxR := math.Inf(1), 0.0
	for i := 0; i <= 500; i++ {
		p := animateSpec(o, float64(i)/500*period, 0)
		r := math.Hypot(p.X, p.Z)
		minR = math.Min(minR, r)
		maxR = math.Max(maxR, r)
	}
	// a nested orbit has a varying radius (unlike a plain circular orbit)
	if maxR < minR*1.3 {
		t.Errorf("rosette radius should vary (nested orbits): min %.2f max %.2f", minR, maxR)
	}
}

// The reference director understands the new motion keywords.
func TestAuthorMotionKeywords(t *testing.T) {
	hasAnim := func(prompt, kind string) bool {
		_, spec, _ := AuthorScene(brain.Local{}, prompt)
		for _, o := range spec.Objects {
			if o.Anim == kind {
				return true
			}
		}
		return false
	}
	if !hasAnim("a glowing infinity sign", "figure8") {
		t.Error("'infinity' should author a figure8 motion")
	}
	if !hasAnim("a spirograph of light", "rosette") {
		t.Error("'spirograph' should author a rosette motion")
	}
}
