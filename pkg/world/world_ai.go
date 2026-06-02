package world

import (
	"encoding/json"
	"math"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/scene3d"
)

// Directives are the per-tick controls a brain returns for game "world": each in
// {-1, 0, 1}. This is the small, model-friendly action space - the LLM picks
// directions, the world physics (Apply) turns them into the next State.
type Directives struct {
	Fold   int `json:"fold"`   // -1 close, +1 open
	Rise   int `json:"rise"`   // -1 lower the relief, +1 raise it
	Spin   int `json:"spin"`   // orbit direction
	Camera int `json:"camera"` // camera yaw direction
}

// Brief is the compact, model-readable view of the world sent as Request.State.
type Brief struct {
	FoldPct    int    `json:"fold_pct"`
	ReliefPct  int    `json:"relief_pct"`
	CameraDeg  int    `json:"camera_deg"`
	OrbitPhase int    `json:"orbit_phase"`
	Menu       string `json:"menu"`
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Init returns a sensible starting world (scene in a ring, fold and relief mid).
func Init() State {
	var s State
	for i := 0; i < 7; i++ {
		s.Fold[i] = 80
	}
	s.Relief = 40
	for i := 0; i < 7; i++ {
		ang := float64(i) * (2 * math.Pi / 7)
		s.Scene[i] = scene3d.Item{Piece: byte(i), X: clamp(128 + 70*math.Cos(ang)), Y: clamp(128 + 70*math.Sin(ang)), Z: 128, Yaw: byte(i % 8), Color: byte(i)}
	}
	return s
}

// MakeBrief summarises a state for the brain.
func MakeBrief(s State) Brief {
	fold := 0
	for i := 0; i < 7; i++ {
		fold += int(s.Fold[i])
	}
	return Brief{
		FoldPct:    fold / 7 * 100 / 240,
		ReliefPct:  int(s.Relief) * 100 / 200,
		CameraDeg:  int(s.Cam) * 360 / 256,
		OrbitPhase: int(s.Scene[0].X),
		Menu:       "reply {fold,rise,spin,camera}, each -1|0|1",
	}
}

// Apply evolves the state by one tick under the directives.
func Apply(s State, d Directives) State {
	ns := s
	ns.Cam = byte((int(s.Cam) + clampi(d.Camera, -1, 1)*6 + 256) % 256)
	ns.Relief = clamp(float64(clampi(int(s.Relief)+clampi(d.Rise, -1, 1)*12, 8, 200)))
	for i := 0; i < 7; i++ {
		ns.Fold[i] = clamp(float64(clampi(int(s.Fold[i])+clampi(d.Fold, -1, 1)*12, 0, 240)))
	}
	ang0 := math.Atan2(float64(int(s.Scene[0].Y)-128), float64(int(s.Scene[0].X)-128)) + float64(clampi(d.Spin, -1, 1))*0.30
	for i := 0; i < 7; i++ {
		ang := ang0 + float64(i)*(2*math.Pi/7)
		ns.Scene[i] = scene3d.Item{
			Piece: byte(i),
			X:     clamp(128 + 70*math.Cos(ang)),
			Y:     clamp(128 + 70*math.Sin(ang)),
			Z:     clamp(120 + 90*math.Sin(ang)),
			Yaw:   byte((i + int(ns.Cam)/32) % 8),
			Color: byte(i),
		}
	}
	return ns
}

// Controller decides the next directives from the current state. An LLM, the
// reference brain, or a script all implement it.
type Controller interface {
	Decide(State) Directives
}

// RefController is a scripted stand-in: it reads the state and breathes the fold,
// rises/falls the relief, and keeps the scene spinning and the camera panning.
type RefController struct{}

// Decide returns directives from the current state.
func (RefController) Decide(s State) Directives {
	d := Directives{Spin: 1, Camera: 1}
	if s.Fold[0] < 200 {
		d.Fold = 1
	} else {
		d.Fold = -1
	}
	if s.Relief < 150 {
		d.Rise = 1
	} else {
		d.Rise = -1
	}
	return d
}

// BrainController bridges any tvcp-ai brain into a Controller: it sends the
// Brief as a game "world" request and parses Response.World into Directives.
// Swap brain.Local for brain.HTTPBrain{URL: ...} and a live LLM drives the world.
type BrainController struct {
	B    brain.Brain
	Last Directives
}

// Decide asks the brain for the next directives (holding the last on any error).
func (c *BrainController) Decide(s State) Directives {
	raw, _ := json.Marshal(MakeBrief(s))
	resp, err := c.B.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "world", State: raw})
	if err != nil || len(resp.World) == 0 {
		return c.Last
	}
	var d Directives
	if json.Unmarshal(resp.World, &d) != nil {
		return c.Last
	}
	d.Fold, d.Rise = clampi(d.Fold, -1, 1), clampi(d.Rise, -1, 1)
	d.Spin, d.Camera = clampi(d.Spin, -1, 1), clampi(d.Camera, -1, 1)
	c.Last = d
	return d
}

// LoopC runs a controller for n ticks, returning the states and the directives.
func LoopC(c Controller, init State, n int) ([]State, []Directives) {
	states := make([]State, 0, n)
	dirs := make([]Directives, 0, n)
	s := init
	for i := 0; i < n; i++ {
		d := c.Decide(s)
		s = Apply(s, d)
		states = append(states, s)
		dirs = append(dirs, d)
	}
	return states, dirs
}

// DirByte packs four directives into one byte (two bits each).
func DirByte(d Directives) byte {
	return byte((d.Fold+1)&3 | ((d.Rise+1)&3)<<2 | ((d.Spin+1)&3)<<4 | ((d.Camera+1)&3)<<6)
}
