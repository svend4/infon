// Package arena is the seed of "origami Warcraft": a multi-entity isometric
// world where every unit is a coloured tangram block on a terrain board and
// each side is driven by a Commander - the reference bot, or an LLM agent over
// tvcp-ai. The whole match serialises to a few bytes per tick (Snapshot) and is
// drawn by the shared isometric renderer (fold.Render). One world, many agents,
// streamed in bytes.
package arena

import "github.com/svend4/infon/pkg/fold"

// Kind is a unit archetype (a figure from the bestiary) with its stats.
type Kind struct {
	Name, Color string
	HP, Atk     uint8
	Range       uint8 // attack range in tiles (1 = melee)
}

// Bestiary is the small roster of unit kinds (drawn from the tangram menagerie).
var Bestiary = []Kind{
	{"fox", "orange", 6, 3, 1},
	{"cat", "gold", 9, 2, 1},
	{"turtle", "darkgreen", 13, 1, 1},
	{"swan", "skyblue", 5, 3, 2},
	{"owl", "amber", 5, 2, 3},
	{"crab", "coral", 8, 2, 2},
}

// Unit is one placed creature.
type Unit struct {
	Kind    uint8
	X, Y    uint8
	HP      uint8
	Faction uint8
	Alive   bool
}

// Arena is the board: a W x H terrain heightmap and the units on it.
type Arena struct {
	W, H    int
	Terrain []uint8 // len H*W, heights 0..3
	Units   []Unit
	Tick    int
}

func factionColor(f uint8) string {
	if f == 0 {
		return "blue"
	}
	return "red"
}

// Action is one unit's intent for a tick: a step (DX,DY in -1..1) or an attack.
type Action struct {
	Unit   int
	DX, DY int
	Attack int // target unit index, or -1
}

// Commander plans the actions for one faction each tick. An LLM agent over
// tvcp-ai implements the same interface.
type Commander interface {
	Plan(a *Arena, faction uint8) []Action
}

func (a *Arena) unitAt(x, y int) int {
	for i := range a.Units {
		if a.Units[i].Alive && int(a.Units[i].X) == x && int(a.Units[i].Y) == y {
			return i
		}
	}
	return -1
}

// AliveCount returns how many of a faction's units are still standing.
func (a *Arena) AliveCount(faction uint8) int {
	n := 0
	for _, u := range a.Units {
		if u.Alive && u.Faction == faction {
			n++
		}
	}
	return n
}

// TotalHP is the summed health of all living units (for tests/telemetry).
func (a *Arena) TotalHP() int {
	t := 0
	for _, u := range a.Units {
		if u.Alive {
			t += int(u.HP)
		}
	}
	return t
}

// RefCommander is the baseline bot: every unit charges the nearest enemy and
// attacks when adjacent. Deterministic, so matches are reproducible.
type RefCommander struct{}

// Plan implements Commander: every unit steps toward the nearest enemy; combat
// auto-resolves on contact in Step.
func (RefCommander) Plan(a *Arena, faction uint8) []Action {
	var acts []Action
	for i := range a.Units {
		u := a.Units[i]
		if !u.Alive || u.Faction != faction {
			continue
		}
		best, bd := -1, 1<<30
		for j := range a.Units {
			e := a.Units[j]
			if !e.Alive || e.Faction == faction {
				continue
			}
			if d := abs(int(u.X)-int(e.X)) + abs(int(u.Y)-int(e.Y)); d < bd {
				bd, best = d, j
			}
		}
		if best < 0 {
			continue
		}
		e := a.Units[best]
		dx, dy := sign(int(e.X)-int(u.X)), sign(int(e.Y)-int(u.Y))
		if abs(int(e.X)-int(u.X)) >= abs(int(e.Y)-int(u.Y)) {
			dy = 0
		} else {
			dx = 0
		}
		acts = append(acts, Action{Unit: i, DX: dx, DY: dy, Attack: -1})
	}
	return acts
}

// Step advances the world one tick: gather both commanders' moves, auto-resolve
// combat (every living unit strikes one adjacent enemy), then apply the moves
// into free, in-bounds tiles.
func (a *Arena) Step(c0, c1 Commander) {
	moves := append(c0.Plan(a, 0), c1.Plan(a, 1)...)
	dmg := make([]int, len(a.Units))
	for i := range a.Units {
		if !a.Units[i].Alive {
			continue
		}
		k := Bestiary[int(a.Units[i].Kind)%len(Bestiary)]
		j := a.targetInRange(i, int(k.Range))
		if j < 0 {
			continue
		}
		d := int(k.Atk)
		if a.height(int(a.Units[i].X), int(a.Units[i].Y)) > a.height(int(a.Units[j].X), int(a.Units[j].Y)) {
			d++ // high-ground bonus
		}
		dmg[j] += d
	}
	for i := range a.Units {
		if dmg[i] == 0 || !a.Units[i].Alive {
			continue
		}
		if int(a.Units[i].HP) <= dmg[i] {
			a.Units[i].HP, a.Units[i].Alive = 0, false
		} else {
			a.Units[i].HP -= uint8(dmg[i])
		}
	}
	for _, act := range moves {
		u := &a.Units[act.Unit]
		if !u.Alive || (act.DX == 0 && act.DY == 0) {
			continue
		}
		nx, ny := int(u.X)+act.DX, int(u.Y)+act.DY
		if nx < 0 || ny < 0 || nx >= a.W || ny >= a.H || a.unitAt(nx, ny) >= 0 {
			continue
		}
		u.X, u.Y = uint8(nx), uint8(ny)
	}
	a.Tick++
}

func (a *Arena) height(x, y int) int {
	if x < 0 || y < 0 || x >= a.W || y >= a.H {
		return 0
	}
	return int(a.Terrain[y*a.W+x])
}

// targetInRange returns the nearest enemy within Manhattan distance r of unit i.
func (a *Arena) targetInRange(i, r int) int {
	u := a.Units[i]
	best, bd := -1, 1<<30
	for j := range a.Units {
		e := a.Units[j]
		if !e.Alive || e.Faction == u.Faction {
			continue
		}
		if d := abs(int(u.X)-int(e.X)) + abs(int(u.Y)-int(e.Y)); d <= r && d < bd {
			bd, best = d, j
		}
	}
	return best
}

// UnitBytes is the wire size of one unit.
const UnitBytes = 5

// Snapshot packs the units to bytes (5 per unit: kind,x,y,hp,flags) - the wire
// frame a deltastream ships each tick.
func (a *Arena) Snapshot() []byte {
	b := make([]byte, 0, len(a.Units)*UnitBytes)
	for _, u := range a.Units {
		flags := u.Faction & 1
		if u.Alive {
			flags |= 2
		}
		b = append(b, u.Kind, u.X, u.Y, u.HP, flags)
	}
	return b
}

// FrameWidth is len(units)*UnitBytes, the deltastream frame width.
func (a *Arena) FrameWidth() int { return len(a.Units) * UnitBytes }

// Faces renders the board + units as isometric extruded blocks for fold.Render.
func Faces(a *Arena) []fold.Face {
	biome := []string{"teal", "green", "sand", "gold"}
	var fs []fold.Face
	for ty := 0; ty < a.H; ty++ {
		for tx := 0; tx < a.W; tx++ {
			th := float64(a.Terrain[ty*a.W+tx])
			col := biome[int(a.Terrain[ty*a.W+tx])%len(biome)]
			fs = append(fs, block(float64(tx), float64(ty), 1, 0, 0.3+th*0.4, col)...)
		}
	}
	for _, u := range a.Units {
		if !u.Alive {
			continue
		}
		base := 0.3 + float64(a.Terrain[int(u.Y)*a.W+int(u.X)])*0.4
		uh := 0.6 + float64(u.HP)*0.07
		fs = append(fs, block(float64(u.X)+0.2, float64(u.Y)+0.2, 0.6, base, base+uh, factionColor(u.Faction))...)
		kc := Bestiary[int(u.Kind)%len(Bestiary)].Color
		fs = append(fs, block(float64(u.X)+0.27, float64(u.Y)+0.27, 0.46, base+uh, base+uh+0.2, kc)...)
	}
	return fs
}

func block(x, y, s, z0, z1 float64, col string) []fold.Face {
	p := [4][2]float64{{x, y}, {x + s, y}, {x + s, y + s}, {x, y + s}}
	top := make([]fold.Pt3, 4)
	for i, q := range p {
		top[i] = fold.Pt3{X: q[0], Y: q[1], Z: z1}
	}
	fs := []fold.Face{{Pts: top, Color: col, Shade: 1.0}}
	shades := [4]float64{0.62, 0.82, 0.7, 0.88}
	for i := 0; i < 4; i++ {
		a0, b0 := p[i], p[(i+1)%4]
		fs = append(fs, fold.Face{Pts: []fold.Pt3{
			{X: a0[0], Y: a0[1], Z: z0}, {X: b0[0], Y: b0[1], Z: z0},
			{X: b0[0], Y: b0[1], Z: z1}, {X: a0[0], Y: a0[1], Z: z1},
		}, Color: col, Shade: shades[i]})
	}
	return fs
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
func sign(a int) int {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}
