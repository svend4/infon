package fleet

import (
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

// camp.go turns Part 1's "logistics in a camp" into the pieces already in the
// repo: the yard floor is a raydir.SquareForest (the dream hackers' navigable grid
// of blocks and roads, here a warehouse), routes are planned with the BubbleGraph
// router (logistics = the shortest chain of open cells), and each robot is placed
// on the Q6 hypercube by a role hexagram so the swarm's relationships draw with
// HexBridges / Q6Map — the same 6-bit coordination info150's portal uses. It gives
// a job to three pieces that shipped dormant: BubbleGraph.Route for logistics,
// HexBridges, and the Q6 navigator.

// Camp is a logistics yard over a square-forest grid.
type Camp struct {
	forest *raydir.SquareForest
	at     raytrace.Vec3
}

// NewCamp builds a gx×gz yard (a seeded square forest).
func NewCamp(gx, gz int, seed int64) *Camp {
	return &Camp{forest: raydir.NewSquareForest(gx, gz, seed)}
}

// NewCampFromForest wraps an existing forest (for deterministic tests/layouts).
func NewCampFromForest(f *raydir.SquareForest) *Camp { return &Camp{forest: f} }

// Dims is the yard's grid size in cells.
func (c *Camp) Dims() (int, int) { return c.forest.Dims() }

// open reports whether cell (x,z) is an open, walkable square.
func (c *Camp) open(x, z int) bool {
	gx, gz := c.forest.Dims()
	if x < 0 || z < 0 || x >= gx || z >= gz {
		return false
	}
	return c.forest.Cell(x, z) == raydir.CellOpen
}

// OpenCells lists every open square, sorted (x then z) — the candidate stations.
func (c *Camp) OpenCells() [][2]int {
	gx, gz := c.forest.Dims()
	var out [][2]int
	for x := 0; x < gx; x++ {
		for z := 0; z < gz; z++ {
			if c.open(x, z) {
				out = append(out, [2]int{x, z})
			}
		}
	}
	return out
}

// gridGraph makes one bubble per open cell, linked to its open 4-neighbours, with
// the bubble laid out at the cell's (x,z) — the router's view of the yard.
func (c *Camp) gridGraph() (*raydir.BubbleGraph, map[[2]int]int) {
	g := raydir.NewBubbleGraph()
	id := map[[2]int]int{}
	for _, cell := range c.OpenCells() {
		id[cell] = g.Add("", raytrace.Vec3{X: float64(cell[0]), Z: float64(cell[1])}, brain.SceneSpec{})
	}
	for cell, a := range id {
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if b, ok := id[[2]int{cell[0] + d[0], cell[1] + d[1]}]; ok {
				g.Link(a, b)
			}
		}
	}
	return g, id
}

// PlanRoute returns the shortest chain of open cells from `from` to `to`
// (inclusive), or nil if either is not open or they are not connected. Computed
// with the BubbleGraph router — logistics on the camp grid.
func (c *Camp) PlanRoute(from, to [2]int) [][2]int {
	g, id := c.gridGraph()
	a, ok1 := id[from]
	b, ok2 := id[to]
	if !ok1 || !ok2 {
		return nil
	}
	path := g.Route(a, b)
	if path == nil {
		return nil
	}
	cellOf := make(map[int][2]int, len(id))
	for cell, i := range id {
		cellOf[i] = cell
	}
	out := make([][2]int, len(path))
	for i, n := range path {
		out[i] = cellOf[n]
	}
	return out
}

// cellWorld is the world-space centre of cell (x,z).
func (c *Camp) cellWorld(cell [2]int) (x, z float64) {
	p := c.forest.Pitch
	return c.at.X + float64(cell[0])*p + p/2, c.at.Z + float64(cell[1])*p + p/2
}

// CampUnit is a robot in the yard: a name, a job (from -> to), and a role placed
// on the Q6 hypercube as a hexagram.
type CampUnit struct {
	Name     string
	Role     raydir.Hexagram
	From, To [2]int
}

// SwarmGraph links units whose role hexagrams lie within maxHamming on the Q6
// hypercube (via HexBridges) — units with similar roles cluster, the relationship
// info150's portal calls a domain-neighbour bridge. The graph is laid out so it
// draws directly with BubbleMap / SVG.
func SwarmGraph(units []CampUnit, maxHamming int) *raydir.BubbleGraph {
	places := make([]raydir.HexPlace, len(units))
	for i, u := range units {
		places[i] = raydir.HexPlace{Name: u.Name, Hex: u.Role}
	}
	g := raydir.HexBridges(places, maxHamming)
	g.Layout(220)
	return g
}

// CampScene authors the 3-D yard: the square-forest geometry, each unit as a
// coloured marker at its start cell, and a planned route as a line of glowing
// markers. It builds a base rayscene then appends the forest's renderer objects.
func (c *Camp) CampScene(units []CampUnit, route [][2]int) *raytrace.Scene {
	spec := brain.SceneSpec{
		Light:  [3]float64{8, 14, -4},
		SkyTop: [3]float64{0.45, 0.6, 0.85}, SkyBot: [3]float64{0.82, 0.86, 0.92},
		Objects: []brain.ObjSpec{{Kind: "plane", Color: [3]float64{0.55, 0.55, 0.5}}},
	}
	for _, cell := range route { // the planned route, a line of glowing markers (drawn first, under the robots)
		x, z := c.cellWorld(cell)
		spec.Objects = append(spec.Objects, brain.ObjSpec{X: x, Y: 0.25, Z: z, R: 0.45, Emit: [3]float64{4, 3.4, 1.6}})
	}
	for _, u := range units {
		x, z := c.cellWorld(u.From)
		col := roleColor(u.Role)
		spec.Objects = append(spec.Objects, brain.ObjSpec{ // robot body
			Kind: "sphere", X: x, Y: 1.2, Z: z, R: 1.2,
			Color: col, Metal: 0.5, Rough: 0.25,
		})
		spec.Objects = append(spec.Objects, brain.ObjSpec{ // a status light so the robot reads even in shadow
			X: x, Y: 2.6, Z: z, R: 0.35, Color: col, Emit: [3]float64{col[0] * 2.5, col[1] * 2.5, col[2] * 2.5},
		})
	}
	scene := raydir.BuildScene(spec)
	scene.Objects = append(scene.Objects, c.forest.Objects(c.at)...)
	scene.BuildBVH() // the forest is hundreds of objects — opt into the top-level BVH
	return scene
}

// roleColor maps a role hexagram to a stable colour (by its number).
func roleColor(h raydir.Hexagram) [3]float64 {
	pal := [][3]float64{
		{0.9, 0.4, 0.3}, {0.4, 0.7, 0.95}, {0.5, 0.85, 0.4},
		{0.95, 0.8, 0.3}, {0.75, 0.5, 0.9}, {0.4, 0.85, 0.8},
	}
	return pal[h.Number()%len(pal)]
}
