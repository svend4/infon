package fleet

import "github.com/svend4/infon/pkg/raydir"

// monitor.go closes the loop between a world entity and the fleet monitor: a
// raydir.Robot patrols the shared world, and each tick the engine assesses its
// telemetry and writes the severity back to the robot's beacon — so a machine that
// overheats reddens in the world on its own, no fixed status. raydir stays a pure
// renderer (it cannot import fleet); the coupling lives here, over the public
// Robot.Status field.

// monitoredUnit pairs a world robot with its unit id and a telemetry source.
type monitoredUnit struct {
	robot *raydir.Robot
	unit  string
	telem func(t float64) []Signal
	t     float64
	last  Assessment
}

// RobotMonitor drives a set of world robots from one MonitoringEngine: each Step
// advances every robot, samples its telemetry, assesses it (each robot keeps its
// own baseline, keyed by unit id), and writes the assessed severity to the robot's
// beacon.
type RobotMonitor struct {
	engine *MonitoringEngine
	units  []*monitoredUnit
}

// NewRobotMonitor returns an empty monitor.
func NewRobotMonitor() *RobotMonitor {
	return &RobotMonitor{engine: NewMonitoringEngine()}
}

// Add registers a robot under unit id, with a telemetry function of the robot's
// own elapsed time (seconds).
func (m *RobotMonitor) Add(r *raydir.Robot, unit string, telem func(t float64) []Signal) {
	m.units = append(m.units, &monitoredUnit{robot: r, unit: unit, telem: telem})
}

// Step advances every robot by dt, assesses its telemetry, and writes the severity
// to its beacon.
func (m *RobotMonitor) Step(dt float64) {
	for _, u := range m.units {
		u.robot.Step(dt)
		u.t += dt
		var sigs []Signal
		if u.telem != nil {
			sigs = u.telem(u.t)
		}
		u.last = m.engine.Assess(Reading{Unit: u.unit, T: u.t, Signals: sigs})
		u.robot.Status = u.last.Severity
	}
}

// Assessments returns the latest assessment of every monitored robot, in add order.
func (m *RobotMonitor) Assessments() []Assessment {
	out := make([]Assessment, 0, len(m.units))
	for _, u := range m.units {
		out = append(out, u.last)
	}
	return out
}
