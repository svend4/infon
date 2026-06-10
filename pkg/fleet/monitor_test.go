package fleet

import (
	"testing"

	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func TestRobotMonitorRedensBeacon(t *testing.T) {
	r := raydir.NewRobot(raytrace.Vec3{}, nil, 0)
	m := NewRobotMonitor()
	// telemetry whose thermal climbs with time
	m.Add(r, "amr-1", func(tt float64) []Signal {
		return []Signal{{Name: "thermal", Value: 0.1 + 0.05*tt, Weight: 1}, {Name: "battery", Value: 0.3, Weight: 0.6}}
	})

	var first, last float64
	for i := 0; i < 30; i++ {
		m.Step(0.2)
		if i == 0 {
			first = r.Status
		}
		last = r.Status
	}
	if last <= first {
		t.Errorf("a heating robot's beacon should climb: first=%.3f last=%.3f", first, last)
	}
	as := m.Assessments()
	if len(as) != 1 {
		t.Fatalf("expected one assessment, got %d", len(as))
	}
	if as[0].Level == LevelOK {
		t.Errorf("a hot robot should not stay OK (sev %.2f)", as[0].Severity)
	}
}

func TestRobotMonitorAlsoMoves(t *testing.T) {
	start := raytrace.Vec3{X: 0, Z: 0}
	goal := raytrace.Vec3{X: 8, Z: 0}
	r := raydir.NewRobot(start, []raytrace.Vec3{goal}, 0.2)
	m := NewRobotMonitor()
	m.Add(r, "u", func(float64) []Signal { return []Signal{{Name: "x", Value: 0.2, Weight: 1}} })
	for i := 0; i < 20; i++ {
		m.Step(0.2)
	}
	if r.Pos.X == start.X && r.Pos.Z == start.Z {
		t.Error("the monitor should also step (move) the robot")
	}
}
