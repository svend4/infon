package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// funnel.go builds the "voronka" — the funnel/whirlpool the dream hackers fall
// through to transit between worlds. It is a glowing vortex of orbs that spirals
// inward and downward and spins over time, narrowing from a wide rim to a point;
// optionally it carries a portal at its mouth, so stepping in transits you
// elsewhere. Re-placed each frame from the shared clock, like water and the flock.

// FunnelColor is the vortex's glow.
var FunnelColor = raytrace.Vec3{X: 0.5, Y: 0.72, Z: 1.0}

const (
	funnelRings   = 16
	funnelPerRing = 14
	funnelTop     = 3.0 // rim height above the base
	funnelDepth   = 4.5 // total drop from rim to point
	funnelRadius  = 3.0 // rim radius
)

// FunnelObjects builds the vortex at `at` for animation time t: a downward-
// narrowing spiral of glowing orbs, twisted along its depth and spun by t.
func FunnelObjects(at raytrace.Vec3, t float64) []raytrace.Object {
	mat := raytrace.Material{Color: FunnelColor, Emit: FunnelColor.Scale(1.6)}
	out := make([]raytrace.Object, 0, funnelRings*funnelPerRing)
	for k := 0; k < funnelRings; k++ {
		f := float64(k) / float64(funnelRings-1) // 0 rim .. 1 point
		y := at.Y + funnelTop - f*funnelDepth
		r := funnelRadius*(1-f) + 0.18
		phase := f*6 + t*2 // twist down the funnel, spin with time
		for m := 0; m < funnelPerRing; m++ {
			a := phase + float64(m)/float64(funnelPerRing)*2*math.Pi
			out = append(out, raytrace.Sphere{
				Center: raytrace.Vec3{X: at.X + math.Cos(a)*r, Y: y, Z: at.Z + math.Sin(a)*r},
				Radius: 0.11 + 0.05*(1-f),
				Mat:    mat,
			})
		}
	}
	return out
}

// funnelObj is one placed funnel: where it sits and an optional transit link.
type funnelObj struct {
	at   raytrace.Vec3
	link *raytrace.Transform
}

// AddFunnel places a swirling funnel at `at`. If a link transform is given, the
// funnel also transits: a portal at its mouth teleports a ray (and, in spirit, a
// walker) by that transform.
func (w *World) AddFunnel(at raytrace.Vec3, link ...raytrace.Transform) {
	f := funnelObj{at: at}
	if len(link) > 0 {
		lp := link[0]
		f.link = &lp
	}
	w.funnels = append(w.funnels, f)
}

// HasFunnels reports whether the world has any funnels (so the frame is animated).
func (w *World) HasFunnels() bool { return len(w.funnels) > 0 }
