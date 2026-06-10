package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// The grid has the requested size, valid cells, and is deterministic in its seed.
func TestSquareForestGrid(t *testing.T) {
	f := NewSquareForest(7, 5, 42)
	gx, gz := f.Dims()
	if gx != 7 || gz != 5 {
		t.Fatalf("expected 7x5, got %dx%d", gx, gz)
	}
	for z := 0; z < gz; z++ {
		for x := 0; x < gx; x++ {
			if c := f.Cell(x, z); c < CellOpen || c > CellBuilding {
				t.Fatalf("invalid cell at (%d,%d): %d", x, z, c)
			}
		}
	}
	g2 := NewSquareForest(7, 5, 42)
	for z := 0; z < gz; z++ {
		for x := 0; x < gx; x++ {
			if f.Cell(x, z) != g2.Cell(x, z) {
				t.Fatal("same seed should give the same grid")
			}
		}
	}
	if NewSquareForest(7, 5, 43).Grid[0][0] == f.Grid[0][0] && allEqual(f, NewSquareForest(7, 5, 43)) {
		t.Error("a different seed should change the grid")
	}
}

func allEqual(a, b *SquareForest) bool {
	gx, gz := a.Dims()
	for z := 0; z < gz; z++ {
		for x := 0; x < gx; x++ {
			if a.Cell(x, z) != b.Cell(x, z) {
				return false
			}
		}
	}
	return true
}

// Roads and open squares are walkable; the interior of a jungle/building block is
// not — that is the maze.
func TestSquareForestWalkable(t *testing.T) {
	f := NewSquareForest(5, 5, 7)
	at := raytrace.Vec3{}
	// force known cells so the test is independent of the seed
	f.Grid[1][1] = CellJungle
	f.Grid[1][2] = CellOpen
	// a point in the road gutter between cells (small offset from a cell corner)
	roadX := f.Pitch + f.margin()*0.3 // inside cell (1,*) but within the road margin
	if !f.Walkable(roadX, f.Pitch+f.margin()*0.3, at) {
		t.Error("the road around a block should be walkable")
	}
	// the centre of the jungle block (1,1) is blocked
	cx := f.Pitch + f.Pitch/2
	if f.Walkable(cx, cx, at) {
		t.Error("the interior of a jungle block should be blocked")
	}
	// the centre of the open cell (2,1) is walkable
	ox := 2*f.Pitch + f.Pitch/2
	if !f.Walkable(ox, f.Pitch+f.Pitch/2, at) {
		t.Error("an open square should be walkable")
	}
	// outside the grid is open field
	if !f.Walkable(-50, -50, at) {
		t.Error("outside the grid should be open")
	}
}

// Objects are produced, flat (on the ground), within the grid extent, and
// repeatable.
func TestSquareForestObjects(t *testing.T) {
	f := NewSquareForest(6, 6, 99)
	at := raytrace.Vec3{Y: 0, Z: 10}
	objs := f.Objects(at)
	if len(objs) == 0 {
		t.Fatal("a forest with jungle/buildings should produce objects")
	}
	// repeatable: a second call yields the same count (deterministic placement)
	if len(f.Objects(at)) != len(objs) {
		t.Error("Objects should be repeatable")
	}
	// flat: every triangle vertex sits at or above the ground plane
	gx, gz := f.Dims()
	maxZ := at.Z + float64(gz)*f.Pitch + 2
	maxX := at.X + float64(gx)*f.Pitch + 2
	for _, o := range objs {
		if tr, ok := o.(raytrace.Triangle); ok {
			for _, v := range []raytrace.Vec3{tr.A, tr.B, tr.C} {
				if v.Y < at.Y-1e-6 {
					t.Fatalf("forest should be flat on the ground, got Y=%.2f", v.Y)
				}
				if v.X < at.X-2 || v.X > maxX || v.Z < at.Z-2 || v.Z > maxZ {
					t.Fatalf("object outside the grid extent: %+v", v)
				}
			}
		}
	}
}
