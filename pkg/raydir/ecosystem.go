package raydir

import (
	"math"
	"math/rand"

	"github.com/svend4/infon/pkg/raytrace"
)

// ecosystem.go is a small artificial-life world that runs WITHOUT a player: a
// population of creatures forages a grid of food, gains energy by eating, spends it
// moving, reproduces when fed and dies when starved — so the population rises,
// crashes, and migrates on its own. The food regenerates at a rate set by the
// HexCA climate (rain zones feed faster than snow), so the weather automaton drives
// the ecology: a deterministic, readable world that lives. It joins climate +
// foraging life into one self-sustaining system.

type creature struct {
	x, z, energy float64
	alive        bool
}

// Ecosystem is a foraging population over a food grid, fed by a climate.
type Ecosystem struct {
	gw, gh    int
	cell      float64 // world units per grid cell
	food      []float64
	creatures []creature
	climate   *Climate
	rng       *rand.Rand
	tick      int
	maxPop    int
}

// NewEcosystem seeds a gw×gh arena with n creatures, fed at a rate set by climate
// (pass nil for a uniform climate).
func NewEcosystem(gw, gh, n int, climate *Climate, seed int64) *Ecosystem {
	rng := rand.New(rand.NewSource(seed))
	e := &Ecosystem{gw: gw, gh: gh, cell: 4, food: make([]float64, gw*gh), climate: climate, rng: rng, maxPop: 250}
	for i := range e.food {
		e.food[i] = rng.Float64()
	}
	for i := 0; i < n; i++ {
		e.creatures = append(e.creatures, creature{
			x: rng.Float64() * float64(gw) * e.cell, z: rng.Float64() * float64(gh) * e.cell, energy: 1.0, alive: true,
		})
	}
	return e
}

func (e *Ecosystem) span() (float64, float64) { return float64(e.gw) * e.cell, float64(e.gh) * e.cell }

// Span is the arena's extent in world units (width, depth) — for framing a camera.
func (e *Ecosystem) Span() (float64, float64) { return e.span() }

func (e *Ecosystem) cellIdx(x, z float64) int {
	cx := clampi(int(x/e.cell), 0, e.gw-1)
	cz := clampi(int(z/e.cell), 0, e.gh-1)
	return cz*e.gw + cx
}

// regen is the per-tick food regrowth at (x,z): the climate's rain zones grow food
// fast, snow slow (a uniform middling rate without a climate).
func (e *Ecosystem) regen(x, z float64) float64 {
	if e.climate == nil {
		return 0.02
	}
	switch e.climate.KindAt(x, z, e.cell*3) {
	case "rain":
		return 0.045
	case "snow":
		return 0.006
	default:
		return 0.02
	}
}

// Step advances the world one tick: food regrows, creatures forage, eat, reproduce
// and die.
func (e *Ecosystem) Step() {
	e.tick++
	sw, sh := e.span()
	for cz := 0; cz < e.gh; cz++ {
		for cx := 0; cx < e.gw; cx++ {
			i := cz*e.gw + cx
			e.food[i] = math.Min(1, e.food[i]+e.regen((float64(cx)+0.5)*e.cell, (float64(cz)+0.5)*e.cell))
		}
	}
	var births []creature
	for i := range e.creatures {
		c := &e.creatures[i]
		if !c.alive {
			continue
		}
		tx, tz, best := c.x, c.z, -1.0
		for cz := 0; cz < e.gh; cz++ { // seek the richest nearby food cell
			for cx := 0; cx < e.gw; cx++ {
				fx, fz := (float64(cx)+0.5)*e.cell, (float64(cz)+0.5)*e.cell
				d := math.Hypot(fx-c.x, fz-c.z)
				if d > 16 {
					continue
				}
				if score := e.food[cz*e.gw+cx] - d*0.02; score > best {
					best, tx, tz = score, fx, fz
				}
			}
		}
		dx, dz := tx-c.x, tz-c.z
		if d := math.Hypot(dx, dz); d > 0.4 {
			step := 1.4
			c.x += dx / d * step
			c.z += dz / d * step
		}
		c.x = math.Mod(c.x+sw, sw)
		c.z = math.Mod(c.z+sh, sh)
		fi := e.cellIdx(c.x, c.z)
		eaten := math.Min(e.food[fi], 0.5)
		e.food[fi] -= eaten
		c.energy += eaten - 0.06 // eating gains, living costs
		if c.energy <= 0 {
			c.alive = false
			continue
		}
		if c.energy > 1.6 && len(e.creatures)+len(births) < e.maxPop {
			c.energy -= 0.8
			births = append(births, creature{x: c.x + e.rng.Float64()*2 - 1, z: c.z + e.rng.Float64()*2 - 1, energy: 0.6, alive: true})
		}
	}
	e.creatures = append(e.creatures, births...)
	if e.tick%20 == 0 { // compact out the dead
		live := e.creatures[:0]
		for _, c := range e.creatures {
			if c.alive {
				live = append(live, c)
			}
		}
		e.creatures = live
	}
}

// Population is the number of living creatures.
func (e *Ecosystem) Population() int {
	n := 0
	for i := range e.creatures {
		if e.creatures[i].alive {
			n++
		}
	}
	return n
}

// TotalFood is the standing crop across the whole grid (0..gw*gh): the resource the
// population grazes against, so population-vs-food traces the boom/bust cycle.
func (e *Ecosystem) TotalFood() float64 {
	sum := 0.0
	for _, f := range e.food {
		sum += f
	}
	return sum
}

// Tick is the number of steps taken.
func (e *Ecosystem) Tick() int { return e.tick }

// Objects renders the world: food as low green tiles (brighter = richer) and
// creatures as spheres coloured by energy (red starving -> green fed).
func (e *Ecosystem) Objects(at raytrace.Vec3) []raytrace.Object {
	var out []raytrace.Object
	for cz := 0; cz < e.gh; cz++ {
		for cx := 0; cx < e.gw; cx++ {
			a := e.food[cz*e.gw+cx]
			if a < 0.1 {
				continue
			}
			out = append(out, raytrace.Sphere{
				Center: raytrace.Vec3{X: at.X + (float64(cx)+0.5)*e.cell, Y: at.Y + 0.05, Z: at.Z + (float64(cz)+0.5)*e.cell},
				Radius: 0.25 + 0.5*a, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.2, Y: 0.3 + 0.5*a, Z: 0.2}, Rough: 0.8},
			})
		}
	}
	for i := range e.creatures {
		c := e.creatures[i]
		if !c.alive {
			continue
		}
		t := clampf(c.energy/1.6, 0, 1)
		col := raytrace.Vec3{X: 1 - t*0.7, Y: 0.3 + t*0.6, Z: 0.2}
		out = append(out, raytrace.Sphere{Center: raytrace.Vec3{X: at.X + c.x, Y: at.Y + 0.5, Z: at.Z + c.z}, Radius: 0.5, Mat: raytrace.Material{Color: col, Emit: col.Scale(0.3), Rough: 0.5}})
	}
	return out
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
