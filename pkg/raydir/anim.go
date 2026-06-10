package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/brain"
)

// anim.go makes the world move. An object can carry a named motion (bob, orbit,
// drift, pulse, wander); every peer evaluates it locally from a shared clock, so a
// bird flies, a light pulses and a moon orbits without a single frame crossing the
// wire — the motion is a formula (meaning), reconstructed identically everywhere.
// Animated objects are kept apart from the static props and re-placed each frame at
// the current time.

// animKinds is the set of recognised motions.
var animKinds = map[string]bool{
	"bob": true, "orbit": true, "drift": true, "pulse": true, "wander": true,
}

// isAnimated reports whether name is a recognised motion.
func isAnimated(name string) bool { return animKinds[name] }

// hash01 maps a small integer to a stable value in [0,1) (for per-object phase /
// wander seeds), so different animated objects don't move in lockstep.
func hash01(i int) float64 {
	x := uint32(i)*2654435761 + 0x9e3779b9
	x ^= x >> 15
	x *= 0x85ebca6b
	x ^= x >> 13
	return float64(x) / float64(1<<32)
}

// animateSpec returns a copy of o with its position/size/emission displaced by its
// motion at time t (seconds), keyed by seed so sibling objects desynchronise. A
// non-animated spec is returned unchanged.
func animateSpec(o brain.ObjSpec, t float64, seed int) brain.ObjSpec {
	if !isAnimated(o.Anim) {
		return o
	}
	amp := o.AAmp
	spd := o.ASpeed
	if spd <= 0 {
		spd = 0.3
	}
	phase := 2 * math.Pi * (spd*t + hash01(seed))
	switch o.Anim {
	case "bob": // vertical float
		if amp == 0 {
			amp = 0.5
		}
		o.Y += amp * math.Sin(phase)
	case "orbit": // circle in the ground plane
		if amp == 0 {
			amp = 2
		}
		o.X += amp * math.Cos(phase)
		o.Z += amp * math.Sin(phase)
	case "drift": // slow horizontal sway
		if amp == 0 {
			amp = 1.5
		}
		o.X += amp * math.Sin(phase)
	case "pulse": // breathing size + emission
		if amp == 0 {
			amp = 0.35
		}
		f := 1 + amp*math.Sin(phase)
		if o.R > 0 {
			o.R *= f
		}
		for i := range o.Emit {
			o.Emit[i] *= f
		}
	case "wander": // smooth pseudo-random roaming (a simple creature)
		if amp == 0 {
			amp = 3
		}
		s2 := hash01(seed + 1)
		o.X += amp * (math.Sin(phase) + 0.5*math.Sin(2.3*phase+6*s2))
		o.Z += amp * (math.Cos(0.8*phase) + 0.5*math.Sin(1.7*phase+6*s2))
	}
	return o
}
