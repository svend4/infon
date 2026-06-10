package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A non-animated spec passes through unchanged; an unknown motion is ignored.
func TestAnimatePassthrough(t *testing.T) {
	o := brain.ObjSpec{Kind: "sphere", Y: 1, R: 1}
	if animateSpec(o, 5, 0) != o {
		t.Error("a static object should be unchanged by animateSpec")
	}
	o.Anim = "not-a-motion"
	if animateSpec(o, 5, 0) != o {
		t.Error("an unknown motion should be ignored")
	}
}

// Bob oscillates Y within the amplitude and is periodic (its per-object seed adds
// a phase offset, so we test over a whole period rather than at t=0).
func TestAnimateBob(t *testing.T) {
	o := brain.ObjSpec{Kind: "sphere", Y: 2, R: 1, Anim: "bob", AAmp: 0.5, ASpeed: 1}
	var lo, hi = math.Inf(1), math.Inf(-1)
	for i := 0; i <= 20; i++ {
		y := animateSpec(o, float64(i)/20, 0).Y // one full period at speed 1
		lo, hi = math.Min(lo, y), math.Max(hi, y)
	}
	if hi-2 < 0.4 || 2-lo < 0.4 || hi-2 > 0.5+1e-9 || 2-lo > 0.5+1e-9 {
		t.Errorf("bob should swing ~±0.5 around base 2, got [%.3f, %.3f]", lo, hi)
	}
	if math.Abs(animateSpec(o, 1, 0).Y-animateSpec(o, 0, 0).Y) > 1e-9 {
		t.Error("bob should be periodic over one period")
	}
}

// Orbit traces a circle in the ground plane (radius ~= amplitude).
func TestAnimateOrbit(t *testing.T) {
	o := brain.ObjSpec{Kind: "sphere", X: 0, Z: 0, R: 0.3, Anim: "orbit", AAmp: 2, ASpeed: 1}
	a := animateSpec(o, 0, 0)
	b := animateSpec(o, 0.25, 0)
	if math.Abs(math.Hypot(a.X, a.Z)-2) > 1e-6 || math.Abs(math.Hypot(b.X, b.Z)-2) > 1e-6 {
		t.Errorf("orbit should keep radius ~2, got %.3f and %.3f", math.Hypot(a.X, a.Z), math.Hypot(b.X, b.Z))
	}
	if a.X == b.X && a.Z == b.Z {
		t.Error("orbit should move over time")
	}
}

// Pulse scales emission above and below the base over a period.
func TestAnimatePulse(t *testing.T) {
	o := brain.ObjSpec{R: 0.5, Emit: [3]float64{4, 4, 4}, Anim: "pulse", AAmp: 0.5, ASpeed: 1}
	var lo, hi = math.Inf(1), math.Inf(-1)
	for i := 0; i <= 20; i++ {
		e := animateSpec(o, float64(i)/20, 0).Emit[0]
		lo, hi = math.Min(lo, e), math.Max(hi, e)
	}
	if !(hi > 4 && lo < 4) {
		t.Errorf("pulse should brighten and dim around base 4: [%.2f, %.2f]", lo, hi)
	}
}

// An animated object is kept out of the static props and re-placed as the world's
// animation clock advances; its rendered position changes over time.
func TestWorldAnimatesOverTime(t *testing.T) {
	w := NewWorld()
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 5}, Spec: brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "sphere", X: 0, Y: 2, Z: 0, R: 1, Color: [3]float64{1, 1, 1}, Anim: "orbit", AAmp: 3, ASpeed: 1},
	}}})
	if w.Props() != 0 {
		t.Errorf("an animated object should not be a static prop, got %d", w.Props())
	}
	// A fixed probe ray hits the orbiting sphere at a different distance (or not at
	// all) as it moves, so its position genuinely changes with the clock.
	probe := raytrace.Ray{Origin: raytrace.Vec3{X: 3, Y: 2, Z: 5}, Dir: raytrace.Vec3{X: -1, Y: 0, Z: 0}}
	w.SetAnimTime(0)
	s0 := w.SceneWith(nil)
	w.SetAnimTime(0.5)
	s1 := w.SceneWith(nil)
	d0, ok0 := nearestHit(s0, probe)
	d1, ok1 := nearestHit(s1, probe)
	if ok0 == ok1 && ok0 && math.Abs(d0.T-d1.T) < 1e-6 {
		t.Error("the orbiting object should be in a different place at t=0 vs t=0.5")
	}
}

func TestRefSceneAnimKeyword(t *testing.T) {
	_, spec, _ := AuthorScene(brain.Local{}, "a flock of birds over a calm world")
	found := false
	for _, o := range spec.Objects {
		if o.Anim == "orbit" {
			found = true
		}
	}
	if !found {
		t.Error("'flock of birds' should author an orbiting object")
	}
}
