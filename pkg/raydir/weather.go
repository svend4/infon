package raydir

import (
	"math"
	"math/rand"

	"github.com/svend4/infon/pkg/raytrace"
)

// weather.go gives the world a sky that does something: rain that streaks down on
// the wind, snow that drifts, or fog that swallows the distance. Precipitation is
// a band of particles kept around the walker (recycled as they fall or drift out
// of view, so the count — and the cost — stays flat however far you walk); fog is
// aerial perspective (Beer-Lambert distance fade) plus a hazed sky. Seeded and
// deterministic, re-placed every frame from the shared clock, like the flock and
// the water.

// wparticle is one drop or flake: where it is, a phase for its sway, and a
// per-particle speed factor so the fall isn't a rigid sheet.
type wparticle struct {
	Pos   raytrace.Vec3
	Phase float64
	Speed float64
}

// Weather is a precipitation/fog system that follows the walker.
type Weather struct {
	Kind     string        // "rain", "snow", or "fog"
	Wind     raytrace.Vec3 // horizontal drift (world units/second)
	parts    []wparticle
	rng      *rand.Rand
	box      float64 // half-extent of the active band around the walker (X and Z)
	top, bot float64 // spawn ceiling and recycle floor
	fall     float64 // base fall speed
	color    raytrace.Vec3
	emit     raytrace.Vec3
	fogDens  float64
	fogColor raytrace.Vec3
}

// NewWeather builds a weather system of the given kind (seeded for determinism).
func NewWeather(kind string, seed int64) *Weather {
	w := &Weather{Kind: kind, rng: rand.New(rand.NewSource(seed)), box: 20, top: 16, bot: 0}
	n := 0
	switch kind {
	case "rain":
		w.fall, w.Wind = 16, raytrace.Vec3{X: 3}
		w.color = raytrace.Vec3{X: 0.6, Y: 0.7, Z: 0.95}
		w.emit = w.color.Scale(0.5)
		n = 260
	case "snow":
		w.fall, w.Wind = 1.8, raytrace.Vec3{X: 1.0}
		w.color = raytrace.Vec3{X: 0.95, Y: 0.96, Z: 1.0}
		w.emit = w.color.Scale(0.35)
		n = 440
	case "fog":
		w.fogDens, w.fogColor = 0.05, raytrace.Vec3{X: 0.72, Y: 0.75, Z: 0.8}
	}
	for i := 0; i < n; i++ {
		w.parts = append(w.parts, w.spawn(raytrace.Vec3{}, true))
	}
	return w
}

// spawn places a particle within the band around centre. anyY scatters it through
// the whole column (for the initial fill); otherwise it starts at the ceiling.
func (w *Weather) spawn(centre raytrace.Vec3, anyY bool) wparticle {
	y := w.top
	if anyY {
		y = w.bot + w.rng.Float64()*(w.top-w.bot)
	}
	return wparticle{
		Pos: raytrace.Vec3{
			X: centre.X + (w.rng.Float64()*2-1)*w.box,
			Y: y,
			Z: centre.Z + (w.rng.Float64()*2-1)*w.box,
		},
		Phase: w.rng.Float64() * 2 * math.Pi,
		Speed: 0.7 + w.rng.Float64()*0.6,
	}
}

// Step advances every particle by dt seconds (gravity + wind, with a sway for
// snow) and recycles any that hit the floor or leave the band around centre, so
// the weather always surrounds the walker at constant cost.
func (w *Weather) Step(dt float64, centre raytrace.Vec3) {
	if dt <= 0 {
		return
	}
	for i := range w.parts {
		p := &w.parts[i]
		p.Pos.Y -= w.fall * p.Speed * dt
		p.Pos = p.Pos.Add(w.Wind.Scale(dt))
		if w.Kind == "snow" { // a gentle lateral drift so flakes don't fall on rails
			p.Phase += dt * 2
			p.Pos.X += math.Sin(p.Phase) * 0.6 * dt
		}
		if p.Pos.Y <= w.bot || math.Abs(p.Pos.X-centre.X) > w.box || math.Abs(p.Pos.Z-centre.Z) > w.box {
			*p = w.spawn(centre, false)
		}
	}
}

// FallDir is the unit direction precipitation travels: straight down, tilted by
// the wind.
func (w *Weather) FallDir() raytrace.Vec3 {
	return w.Wind.Add(raytrace.Vec3{Y: -w.fall}).Norm()
}

// Fog reports the aerial-perspective fog this weather adds: density, colour, and
// whether it is on.
func (w *Weather) Fog() (float64, raytrace.Vec3, bool) {
	return w.fogDens, w.fogColor, w.fogDens > 0
}

// streak builds a thin ribbon (two triangles) from a to b — a single rain drop's
// motion-blurred streak.
func streak(a, b raytrace.Vec3, width float64, mat raytrace.Material) []raytrace.Object {
	dir := b.Sub(a)
	perp := dir.Cross(raytrace.Vec3{Z: 1})
	if perp.LenSq() < 1e-9 {
		perp = dir.Cross(raytrace.Vec3{X: 1})
	}
	perp = perp.Norm().Scale(width * 0.5)
	a0, a1 := a.Sub(perp), a.Add(perp)
	b0, b1 := b.Sub(perp), b.Add(perp)
	return []raytrace.Object{
		raytrace.Triangle{A: a0, B: a1, C: b1, Mat: mat},
		raytrace.Triangle{A: a0, B: b1, C: b0, Mat: mat},
	}
}

// Objects renders the precipitation: snow as small bright flakes, rain as thin
// streaks along the fall direction. Fog has no objects (it's a scene property).
func (w *Weather) Objects() []raytrace.Object {
	if len(w.parts) == 0 {
		return nil
	}
	mat := raytrace.Material{Color: w.color, Emit: w.emit}
	if w.Kind == "snow" {
		out := make([]raytrace.Object, 0, len(w.parts))
		for _, p := range w.parts {
			out = append(out, raytrace.Sphere{Center: p.Pos, Radius: 0.08, Mat: mat})
		}
		return out
	}
	dir := w.FallDir().Scale(0.9) // streak length along travel
	out := make([]raytrace.Object, 0, len(w.parts)*2)
	for _, p := range w.parts {
		out = append(out, streak(p.Pos, p.Pos.Add(dir), 0.03, mat)...)
	}
	return out
}
