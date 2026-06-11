package raydir

// dungeon.go fuses the alife ecosystem (a living world) with the Q6 roguelike (a maze
// over the hypercube): every room on the solved route holds its own ecosystem, fed by
// a climate and a fertility set by the room's hexagram, and the rooms are COUPLED —
// each tick some creatures migrate along the route's edges. So a barren room (foggy,
// sunless) is kept alive by immigration from lush neighbours while it would crash on
// its own: source-sink dynamics. The living dungeon is a metapopulation strung along
// a path through six-dimensional space.

// LivingDungeon is a chain of ecosystems, one per room on a route, coupled by
// migration along the route's edges.
type LivingDungeon struct {
	route []Hexagram
	rooms []*Ecosystem
	migr  float64 // fraction of a room's creatures that emigrate each tick
	moved int     // total migrants moved over the dungeon's life
	tick  int
}

// roomRichness maps a room's hexagram to a fertility: lush when sunny and dense,
// barren when foggy and sunless — so the worlds along the route differ in how much
// life they can support.
func roomRichness(h Hexagram) float64 {
	v := VectorFromHexagram(h)
	return clampf(0.5+1.4*v[AxSun]+1.0*v[AxDensity]-1.2*v[AxFog], 0.15, 2.2)
}

// NewLivingDungeon strings an ecosystem along each room of route: each fed by a
// climate seeded from its hexagram, with a fertility from the hexagram, and seeded
// with startPop creatures. migr is the per-edge migration fraction each tick.
func NewLivingDungeon(route []Hexagram, gw, gh, startPop int, migr float64, seed int64) *LivingDungeon {
	d := &LivingDungeon{route: route, migr: clampf(migr, 0, 1)}
	for i, h := range route {
		e := NewEcosystem(gw, gh, startPop, NewClimate(h.Seed()), seed+int64(i))
		e.SetRichness(roomRichness(h))
		d.rooms = append(d.rooms, e)
	}
	return d
}

// Step advances every room one tick, then moves migrants along the route's edges
// (snapshot semantics: all rooms emigrate before anyone immigrates).
func (d *LivingDungeon) Step() {
	d.tick++
	for _, e := range d.rooms {
		e.Step()
	}
	if len(d.rooms) < 2 || d.migr <= 0 {
		return
	}
	leaving := make([][]Migrant, len(d.rooms))
	for i, e := range d.rooms {
		leaving[i] = e.Emigrate(d.migr)
		d.moved += len(leaving[i])
	}
	for i, ms := range leaving {
		if len(ms) == 0 {
			continue
		}
		switch {
		case i == 0: // only a forward neighbour
			d.rooms[i+1].Immigrate(ms)
		case i == len(d.rooms)-1: // only a backward neighbour
			d.rooms[i-1].Immigrate(ms)
		default: // split between the two neighbours
			mid := len(ms) / 2
			d.rooms[i-1].Immigrate(ms[:mid])
			d.rooms[i+1].Immigrate(ms[mid:])
		}
	}
}

// Populations is the living population of each room, in route order.
func (d *LivingDungeon) Populations() []int {
	out := make([]int, len(d.rooms))
	for i, e := range d.rooms {
		out[i] = e.Population()
	}
	return out
}

// Room returns the ecosystem in room i (for rendering its living arena).
func (d *LivingDungeon) Room(i int) *Ecosystem { return d.rooms[i] }

// Route is the chain of rooms (hexagrams) the dungeon lives along.
func (d *LivingDungeon) Route() []Hexagram { return d.route }

// Richness is room i's fertility (lush > 1 > barren).
func (d *LivingDungeon) Richness(i int) float64 { return roomRichness(d.route[i]) }

// Migrated is the total number of creatures that have crossed an edge.
func (d *LivingDungeon) Migrated() int { return d.moved }

// Tick is the number of steps taken.
func (d *LivingDungeon) Tick() int { return d.tick }
