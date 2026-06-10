package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// season.go turns a long walk into a journey through the year. The world's
// distance forward maps to a season; foliage, ground and sky shift from spring
// green to summer deep-green to autumn gold to winter snow and back. A region's
// trees are tinted by the season where the region sits (baked once, at build
// time, so it costs nothing per frame); the shared floor and sky follow the
// frontier, advancing each time the world grows. Everything is a pure function of
// position, so it's deterministic and testable.

// Season is the look of one time of year.
type Season struct {
	Name    string
	Foliage raytrace.Vec3 // tree canopy colour
	Ground  raytrace.Vec3 // floor tint
	Sky     raytrace.Vec3 // sky tint
	Snow    float64       // 0..1 snow lying on the ground
}

// seasonLen is how far you walk (world units) through one season before the next.
const seasonLen = 60.0

// the year, in order. Walking forward cycles through them.
var seasons = [4]Season{
	{Name: "spring", Foliage: raytrace.Vec3{X: 0.4, Y: 0.72, Z: 0.32}, Ground: raytrace.Vec3{X: 0.32, Y: 0.5, Z: 0.28}, Sky: raytrace.Vec3{X: 0.6, Y: 0.82, Z: 0.95}, Snow: 0},
	{Name: "summer", Foliage: raytrace.Vec3{X: 0.13, Y: 0.5, Z: 0.16}, Ground: raytrace.Vec3{X: 0.26, Y: 0.44, Z: 0.2}, Sky: raytrace.Vec3{X: 0.45, Y: 0.7, Z: 0.96}, Snow: 0},
	{Name: "autumn", Foliage: raytrace.Vec3{X: 0.86, Y: 0.45, Z: 0.12}, Ground: raytrace.Vec3{X: 0.46, Y: 0.33, Z: 0.16}, Sky: raytrace.Vec3{X: 0.82, Y: 0.7, Z: 0.52}, Snow: 0},
	{Name: "winter", Foliage: raytrace.Vec3{X: 0.52, Y: 0.45, Z: 0.4}, Ground: raytrace.Vec3{X: 0.86, Y: 0.88, Z: 0.92}, Sky: raytrace.Vec3{X: 0.72, Y: 0.76, Z: 0.86}, Snow: 1},
}

// mixVec linearly blends two colours/vectors.
func mixVec(a, b raytrace.Vec3, t float64) raytrace.Vec3 { return a.Scale(1 - t).Add(b.Scale(t)) }

// SeasonAt returns the (blended) season at world depth z. Adjacent seasons cross-
// fade smoothly, so there is no hard seam between, say, autumn and winter.
func SeasonAt(z float64) Season {
	phase := z / seasonLen
	base := math.Floor(phase)
	frac := phase - base
	i := ((int(base) % 4) + 4) % 4
	j := (i + 1) % 4
	t := frac * frac * (3 - 2*frac) // smoothstep
	a, b := seasons[i], seasons[j]
	name := a.Name
	if t >= 0.5 {
		name = b.Name
	}
	return Season{
		Name:    name,
		Foliage: mixVec(a.Foliage, b.Foliage, t),
		Ground:  mixVec(a.Ground, b.Ground, t),
		Sky:     mixVec(a.Sky, b.Sky, t),
		Snow:    a.Snow*(1-t) + b.Snow*t,
	}
}

// seasonTintSpec recolours a tree spec to the season's foliage (other kinds pass
// through unchanged), so a region's canopy matches the time of year where it sits.
func seasonTintSpec(o brain.ObjSpec, z float64) brain.ObjSpec {
	if o.Kind != "tree" {
		return o
	}
	f := SeasonAt(z).Foliage
	o.Color = [3]float64{f.X, f.Y, f.Z}
	return o
}

// seasonFloor tints a floor plane for the season: toward the ground palette, then
// whitened by lying snow.
func seasonFloor(p raytrace.Plane, se Season) raytrace.Plane {
	white := raytrace.Vec3{X: 1, Y: 1, Z: 1}
	p.C1 = mixVec(mixVec(p.C1, se.Ground, 0.6), white, se.Snow*0.85)
	p.C2 = mixVec(mixVec(p.C2, se.Ground, 0.6), white, se.Snow*0.85)
	return p
}
