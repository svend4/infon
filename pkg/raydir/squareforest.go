package raydir

import (
	"math/rand"

	"github.com/svend4/infon/pkg/raytrace"
)

// squareforest.go builds the "square forest" the dream hackers describe: a forest
// or park always divided into square sections by clearings and roads, dead flat,
// where some squares are impenetrable jungle and others are buildings filling a
// whole block — a comprehensible maze you navigate along the roads between the
// blocks. Seeded and deterministic.

// ForestCell is what fills one square of the grid.
type ForestCell int

const (
	CellOpen     ForestCell = iota // an open square — walkable
	CellJungle                     // dense trees — a wall
	CellBuilding                   // a block-filling building — a wall
)

// SquareForest is a flat grid of square cells separated by roads.
type SquareForest struct {
	Grid  [][]ForestCell // [z][x]
	Pitch float64        // cell-to-cell spacing (block + road)
	Block float64        // filled block size (< Pitch; the remainder is road)
	seed  int64
}

// NewSquareForest builds a gx×gz grid (seeded). Roughly half the squares are
// jungle or buildings; the roads between them always stay clear.
func NewSquareForest(gx, gz int, seed int64) *SquareForest {
	if gx < 1 {
		gx = 1
	}
	if gz < 1 {
		gz = 1
	}
	f := &SquareForest{Pitch: 8, Block: 5, seed: seed}
	rng := rand.New(rand.NewSource(seed))
	f.Grid = make([][]ForestCell, gz)
	for z := 0; z < gz; z++ {
		f.Grid[z] = make([]ForestCell, gx)
		for x := 0; x < gx; x++ {
			switch r := rng.Float64(); {
			case r < 0.4:
				f.Grid[z][x] = CellJungle
			case r < 0.6:
				f.Grid[z][x] = CellBuilding
			default:
				f.Grid[z][x] = CellOpen
			}
		}
	}
	return f
}

// Dims returns the grid width and height in cells.
func (f *SquareForest) Dims() (int, int) {
	if len(f.Grid) == 0 {
		return 0, 0
	}
	return len(f.Grid[0]), len(f.Grid)
}

// Cell returns the cell at grid (x,z), or CellOpen out of bounds (the open field
// around the forest).
func (f *SquareForest) Cell(x, z int) ForestCell {
	gx, gz := f.Dims()
	if x < 0 || z < 0 || x >= gx || z >= gz {
		return CellOpen
	}
	return f.Grid[z][x]
}

// margin is the road half-width on each side of a block within its pitch.
func (f *SquareForest) margin() float64 { return (f.Pitch - f.Block) / 2 }

// Walkable reports whether the world point (wx,wz) is on a road or in an open
// square (so a walker can stand there). Points inside a jungle/building block are
// blocked — that is what makes the grid a maze.
func (f *SquareForest) Walkable(wx, wz float64, at raytrace.Vec3) bool {
	lx, lz := wx-at.X, wz-at.Z
	if lx < 0 || lz < 0 {
		return true // outside the grid: open field
	}
	ix, iz := int(lx/f.Pitch), int(lz/f.Pitch)
	gx, gz := f.Dims()
	if ix >= gx || iz >= gz {
		return true
	}
	if f.Cell(ix, iz) == CellOpen {
		return true
	}
	ox, oz := lx-float64(ix)*f.Pitch, lz-float64(iz)*f.Pitch
	m := f.margin()
	inBlock := ox >= m && ox <= m+f.Block && oz >= m && oz <= m+f.Block
	return !inBlock // on the road around the block is walkable; in the block is not
}

// blockCenter is the world-space centre of cell (x,z).
func (f *SquareForest) blockCenter(x, z int, at raytrace.Vec3) raytrace.Vec3 {
	return raytrace.Vec3{
		X: at.X + float64(x)*f.Pitch + f.Pitch/2,
		Y: at.Y,
		Z: at.Z + float64(z)*f.Pitch + f.Pitch/2,
	}
}

// Objects builds the forest geometry at `at`: clusters of trees for jungle squares,
// a building for building squares; open squares and roads are left clear. Flat —
// everything sits on the ground.
func (f *SquareForest) Objects(at raytrace.Vec3) []raytrace.Object {
	var out []raytrace.Object
	rng := rand.New(rand.NewSource(f.seed + 1)) // independent of grid generation, so Objects is repeatable
	gx, gz := f.Dims()
	for z := 0; z < gz; z++ {
		for x := 0; x < gx; x++ {
			c := f.blockCenter(x, z, at)
			switch f.Grid[z][x] {
			case CellJungle:
				for i := 0; i < 4; i++ { // a dense clump within the block
					ox := (rng.Float64()*2 - 1) * f.Block * 0.35
					oz := (rng.Float64()*2 - 1) * f.Block * 0.35
					out = append(out, treeObjects(raytrace.Vec3{X: c.X + ox, Y: at.Y, Z: c.Z + oz}, 1.1, raytrace.Material{})...)
				}
			case CellBuilding:
				out = append(out, houseObjects(c, f.Block*0.6, raytrace.Material{})...)
			case CellOpen:
				// left clear
			}
		}
	}
	return out
}
