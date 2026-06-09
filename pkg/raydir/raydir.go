// Package raydir flies a ray-traced camera with a tvcp-ai brain. It reuses the
// existing game:"world" protocol: each tick it sends a compact Brief of the
// camera's orbit / zoom / height / pitch and applies the brain's
// {fold,rise,spin,camera} directives. So the built-in reference brain, a local
// Ollama model, or a cloud LLM all "fly the camera" through the same /v1/decide
// endpoint — closing the loop neural -> bytes -> real 3-D.
package raydir

import (
	"encoding/json"
	"math"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// CamState is the camera's orbit parameterisation.
type CamState struct {
	Angle  float64 // orbit angle (radians)
	Radius float64
	Height float64
	Pitch  float64 // extra pitch on top of the look-at
	Target raytrace.Vec3
}

// Camera converts the orbit state into a raytrace.Camera looking at Target.
func (s CamState) Camera() raytrace.Camera {
	pos := raytrace.Vec3{X: math.Sin(s.Angle) * s.Radius, Y: s.Height, Z: math.Cos(s.Angle) * s.Radius}
	d := s.Target.Sub(pos)
	return raytrace.Camera{
		Pos:   pos,
		Yaw:   math.Atan2(d.X, d.Z),
		Pitch: math.Atan2(d.Y, math.Hypot(d.X, d.Z)) + s.Pitch,
		FOV:   math.Pi / 3,
	}
}

const (
	dAngle  = 0.18
	dRadius = 0.25
	dHeight = 0.20
	dPitch  = 0.05
	rMin    = 3.5
	rMax    = 9.0
	hMin    = 0.5
	hMax    = 5.0
	pMin    = -0.6
	pMax    = 0.6
)

// Director advances the camera state by one tick.
type Director interface {
	Next(s CamState) CamState
}

// directives mirrors the game:"world" reply shape.
type directives struct{ Fold, Rise, Spin, Camera int }

// brief is the camera view sent to the brain; the field names match what the
// reference world brain reads (fold_pct, relief_pct), so brain.Reference works.
type brief struct {
	FoldPct    int    `json:"fold_pct"`
	ReliefPct  int    `json:"relief_pct"`
	CameraDeg  int    `json:"camera_deg"`
	OrbitPhase int    `json:"orbit_phase"`
	Menu       string `json:"menu"`
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampd(v int) int {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}

// apply maps {fold,rise,spin,camera} onto the orbit: spin orbits, fold dollies
// the radius, rise raises the camera, camera tilts the pitch.
func apply(s CamState, d directives) CamState {
	s.Angle += float64(clampd(d.Spin)) * dAngle
	s.Radius = clampf(s.Radius+float64(clampd(d.Fold))*dRadius, rMin, rMax)
	s.Height = clampf(s.Height+float64(clampd(d.Rise))*dHeight, hMin, hMax)
	s.Pitch = clampf(s.Pitch+float64(clampd(d.Camera))*dPitch, pMin, pMax)
	return s
}

// RefDirector is a scripted stand-in: orbit, breathe the radius and height.
type RefDirector struct{}

// Next implements Director.
func (RefDirector) Next(s CamState) CamState {
	d := directives{Spin: 1}
	if s.Radius < (rMin+rMax)/2 {
		d.Fold = 1
	} else {
		d.Fold = -1
	}
	if s.Height < (hMin+hMax)/2 {
		d.Rise = 1
	} else {
		d.Rise = -1
	}
	return apply(s, d)
}

// BrainDirector flies the camera with any tvcp-ai brain via game:"world". On an
// error or empty reply it holds the last directives, so a quiet model still flies.
type BrainDirector struct {
	B    brain.Brain
	last directives
}

// Next implements Director.
func (c *BrainDirector) Next(s CamState) CamState {
	b := brief{
		FoldPct:    int((s.Radius - rMin) / (rMax - rMin) * 100),
		ReliefPct:  int((s.Height - hMin) / (hMax - hMin) * 100),
		CameraDeg:  int(s.Angle*180/math.Pi) % 360,
		OrbitPhase: int(s.Angle*180/math.Pi) % 360,
		Menu:       "reply {fold,rise,spin,camera}, each -1|0|1",
	}
	raw, _ := json.Marshal(b)
	resp, err := c.B.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "world", State: raw})
	d := c.last
	if err == nil && len(resp.World) > 0 {
		var nd directives
		if json.Unmarshal(resp.World, &nd) == nil {
			d = directives{Fold: clampd(nd.Fold), Rise: clampd(nd.Rise), Spin: clampd(nd.Spin), Camera: clampd(nd.Camera)}
			c.last = d
		}
	}
	return apply(s, d)
}

// Fly runs a director for n ticks from init, returning the camera at each tick.
func Fly(d Director, init CamState, n int) []raytrace.Camera {
	cams := make([]raytrace.Camera, 0, n)
	s := init
	for i := 0; i < n; i++ {
		s = d.Next(s)
		cams = append(cams, s.Camera())
	}
	return cams
}
