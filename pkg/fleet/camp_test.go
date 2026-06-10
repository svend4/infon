package fleet

import (
	"testing"

	"github.com/svend4/infon/pkg/raydir"
)

// a deterministic forest from an explicit grid (grid[z][x]).
func forestGrid(g [][]raydir.ForestCell) *raydir.SquareForest {
	return &raydir.SquareForest{Grid: g, Pitch: 8, Block: 5}
}

func TestCampRouteGoesAroundWalls(t *testing.T) {
	o, j := raydir.CellOpen, raydir.CellJungle
	camp := NewCampFromForest(forestGrid([][]raydir.ForestCell{
		{o, o, o},
		{j, j, o},
		{o, o, o},
	}))
	route := camp.PlanRoute([2]int{0, 0}, [2]int{0, 2})
	if route == nil {
		t.Fatal("expected a route around the jungle wall")
	}
	if route[0] != [2]int{0, 0} || route[len(route)-1] != [2]int{0, 2} {
		t.Fatalf("route must run from start to goal, got %v", route)
	}
	for i, cell := range route {
		if !camp.open(cell[0], cell[1]) {
			t.Errorf("route step %d is not an open cell: %v", i, cell)
		}
		if i > 0 {
			d := abs(cell[0]-route[i-1][0]) + abs(cell[1]-route[i-1][1])
			if d != 1 {
				t.Errorf("route step %d is not 4-adjacent (%v -> %v)", i, route[i-1], cell)
			}
		}
	}
}

func TestCampRouteBlockedAndDisconnected(t *testing.T) {
	o, j := raydir.CellOpen, raydir.CellJungle
	camp := NewCampFromForest(forestGrid([][]raydir.ForestCell{
		{o, j, o},
		{j, j, j},
		{o, j, o},
	}))
	if r := camp.PlanRoute([2]int{0, 0}, [2]int{0, 1}); r != nil {
		t.Errorf("routing into a wall cell should fail, got %v", r)
	}
	if r := camp.PlanRoute([2]int{0, 0}, [2]int{2, 0}); r != nil {
		t.Errorf("disconnected open cells should not route, got %v", r)
	}
	if r := camp.PlanRoute([2]int{0, 0}, [2]int{0, 0}); len(r) != 1 {
		t.Errorf("a cell should route to itself as a single step, got %v", r)
	}
}

func TestCampOpenCells(t *testing.T) {
	o, j, b := raydir.CellOpen, raydir.CellJungle, raydir.CellBuilding
	camp := NewCampFromForest(forestGrid([][]raydir.ForestCell{
		{o, j},
		{b, o},
	}))
	open := camp.OpenCells()
	if len(open) != 2 {
		t.Fatalf("expected 2 open cells, got %d (%v)", len(open), open)
	}
	for _, c := range open {
		if !camp.open(c[0], c[1]) {
			t.Errorf("OpenCells returned a non-open cell: %v", c)
		}
	}
}

func TestSwarmGraphClustersByRole(t *testing.T) {
	hex := func(s string) raydir.Hexagram { h, _ := raydir.ParseHexagram(s); return h }
	units := []CampUnit{
		{Name: "haul-1", Role: hex("110000")},
		{Name: "haul-2", Role: hex("110000")},    // same role (Hamming 0)
		{Name: "charge-1", Role: hex("110001")},  // 1 line from haulers
		{Name: "inspect-1", Role: hex("001100")}, // far
	}
	g := SwarmGraph(units, 1)
	if g.Len() != 4 {
		t.Fatalf("expected 4 units, got %d", g.Len())
	}
	id := map[string]int{}
	for _, b := range g.Bubbles() {
		id[b.Name] = b.ID
	}
	linked := func(a, b int) bool {
		for _, n := range g.Neighbors(a) {
			if n == b {
				return true
			}
		}
		return false
	}
	if !linked(id["haul-1"], id["haul-2"]) || !linked(id["haul-1"], id["charge-1"]) {
		t.Error("haulers and the adjacent charger should bridge at Hamming 1")
	}
	if len(g.Neighbors(id["inspect-1"])) != 0 {
		t.Errorf("the distant inspector should not bridge, got %v", g.Neighbors(id["inspect-1"]))
	}
}

func TestCampSceneRenders(t *testing.T) {
	camp := NewCamp(6, 6, 3)
	open := camp.OpenCells()
	if len(open) < 2 {
		t.Skip("degenerate camp")
	}
	units := []CampUnit{{Name: "u", Role: raydir.HexagramFromNumber(5), From: open[0], To: open[len(open)-1]}}
	route := camp.PlanRoute(units[0].From, units[0].To)
	scene := camp.CampScene(units, route)
	if scene == nil || len(scene.Objects) == 0 {
		t.Fatal("camp scene did not build")
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
