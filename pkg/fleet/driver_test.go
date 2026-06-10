package fleet

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

// stubBrain always dispatches to a fixed station index.
type stubBrain struct{ idx int }

func (s stubBrain) Decide(req brain.Request) (brain.Response, error) {
	i := s.idx
	return brain.Response{Protocol: brain.Protocol, Kind: "move", Move: &brain.Move{CardIndex: &i}}, nil
}

func TestBrainDriverFollowsBrain(t *testing.T) {
	stations := []raytrace.Vec3{{X: 0, Z: 0}, {X: 5, Z: 0}, {X: 0, Z: 6}}
	r := raydir.NewRobot(raytrace.Vec3{X: 0, Z: 0}, nil, 0.9)
	d := NewBrainDriver(stubBrain{idx: 2}) // always station 2
	d.Add(r, "amr-1", stations, []string{"base", "dock", "field"}, func() float64 { return 0.9 })
	for i := 0; i < 400; i++ {
		d.Step(0.05)
	}
	if got := d.Decisions()["amr-1"]; got != "field" {
		t.Errorf("the robot should be dispatched to the brain's choice (field), got %q", got)
	}
	// it should be near station 2
	s2 := stations[2]
	if dx, dz := r.Pos.X-s2.X, r.Pos.Z-s2.Z; dx*dx+dz*dz > 1.5 {
		t.Errorf("robot should travel to the brain's station, ended at %v", r.Pos)
	}
}

func TestBrainDriverFallsBackRoundRobin(t *testing.T) {
	stations := []raytrace.Vec3{{X: 0, Z: 0}, {X: 4, Z: 0}, {X: 8, Z: 0}}
	r := raydir.NewRobot(raytrace.Vec3{X: 0, Z: 0}, nil, 0.2)
	d := NewBrainDriver(nil) // no brain -> round-robin
	d.Add(r, "u", stations, nil, nil)
	visited := map[int]bool{}
	for i := 0; i < 1200; i++ {
		d.Step(0.05)
		for j, s := range stations {
			if dx, dz := r.Pos.X-s.X, r.Pos.Z-s.Z; dx*dx+dz*dz < 0.3*0.3 {
				visited[j] = true
			}
		}
	}
	if len(visited) < 3 {
		t.Errorf("round-robin should visit all stations, visited %v", visited)
	}
}
