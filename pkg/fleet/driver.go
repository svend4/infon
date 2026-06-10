package fleet

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

// driver.go lets a tvcp-ai/1 brain dispatch the robots: whenever a robot is ready
// for a new destination, the brain is asked which of its stations to head to next,
// given the robot's position and live status — so a hot robot can be sent back to
// base to cool down while healthy ones keep working. It falls back to a round-robin
// patrol when there is no brain or the brain declines, so it always runs offline.

type station struct {
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Z    float64 `json:"z"`
}

type driverUnit struct {
	robot    *raydir.Robot
	unit     string
	stations []raytrace.Vec3
	names    []string
	status   func() float64
	rr       int
	lastIdx  int
}

// BrainDriver dispatches a set of robots from a brain (or round-robin if nil).
type BrainDriver struct {
	brain brain.Brain
	units []*driverUnit
}

// NewBrainDriver returns a driver that asks b for each dispatch; pass nil to run
// the offline round-robin patrol.
func NewBrainDriver(b brain.Brain) *BrainDriver { return &BrainDriver{brain: b} }

// Add registers a robot with its candidate stations (names optional, parallel to
// stations) and a status source (may be nil).
func (d *BrainDriver) Add(r *raydir.Robot, unit string, stations []raytrace.Vec3, names []string, status func() float64) {
	d.units = append(d.units, &driverUnit{robot: r, unit: unit, stations: stations, names: names, status: status})
}

// Step advances every robot and, when one is ready for a new destination, asks the
// brain (or the fallback) which station to send it to.
func (d *BrainDriver) Step(dt float64) {
	for _, u := range d.units {
		u.robot.Step(dt)
		if u.robot.AtGoal() && len(u.stations) > 0 {
			u.lastIdx = d.choose(u)
			u.robot.GoTo(u.stations[u.lastIdx])
		}
	}
}

func (d *BrainDriver) choose(u *driverUnit) int {
	if d.brain != nil {
		if i, ok := d.ask(u); ok {
			return i
		}
	}
	u.rr = (u.rr + 1) % len(u.stations)
	return u.rr
}

func (d *BrainDriver) ask(u *driverUnit) (int, bool) {
	st := 0.0
	if u.status != nil {
		st = u.status()
	}
	stns := make([]station, len(u.stations))
	for i, s := range u.stations {
		name := ""
		if i < len(u.names) {
			name = u.names[i]
		}
		stns[i] = station{Name: name, X: s.X, Z: s.Z}
	}
	state, _ := json.Marshal(struct {
		Unit     string    `json:"unit"`
		X        float64   `json:"x"`
		Z        float64   `json:"z"`
		Status   float64   `json:"status"`
		Stations []station `json:"stations"`
	}{u.unit, u.robot.Pos.X, u.robot.Pos.Z, st, stns})
	resp, err := d.brain.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "robot", State: state})
	if err != nil || resp.Move == nil || resp.Move.CardIndex == nil {
		return 0, false
	}
	i := *resp.Move.CardIndex
	if i < 0 || i >= len(u.stations) {
		return 0, false
	}
	return i, true
}

// Decisions reports the station each unit was last dispatched to (by name).
func (d *BrainDriver) Decisions() map[string]string {
	out := map[string]string{}
	for _, u := range d.units {
		name := ""
		if u.lastIdx < len(u.names) {
			name = u.names[u.lastIdx]
		}
		out[u.unit] = name
	}
	return out
}
