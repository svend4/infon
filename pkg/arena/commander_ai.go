package arena

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/brain"
)

// RpgBrief is the battlefield view sent to a brain for game "rpg".
type RpgBrief struct {
	You     int       `json:"you"`
	W       int       `json:"w"`
	H       int       `json:"h"`
	Units   []RpgUnit `json:"units"`
	Enemies []RpgFoe  `json:"enemies"`
	Menu    string    `json:"menu"`
}

// RpgUnit is one of the brain's own units in the Brief.
type RpgUnit struct {
	ID   int    `json:"id"`
	Kind string `json:"kind"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	HP   int    `json:"hp"`
}

// RpgFoe is a visible enemy in the Brief.
type RpgFoe struct {
	X  int `json:"x"`
	Y  int `json:"y"`
	HP int `json:"hp"`
}

// RpgMove is one unit's step, dx/dy in -1..1.
type RpgMove struct {
	ID int `json:"id"`
	DX int `json:"dx"`
	DY int `json:"dy"`
}

// Brief builds the battlefield view for a faction.
func (a *Arena) Brief(faction uint8) RpgBrief {
	b := RpgBrief{You: int(faction), W: a.W, H: a.H,
		Menu: "reply rpg: a list of {id,dx,dy}, dx/dy each -1,0,1, to move your units toward enemies"}
	for i := range a.Units {
		u := a.Units[i]
		if !u.Alive {
			continue
		}
		if u.Faction == faction {
			b.Units = append(b.Units, RpgUnit{ID: i, Kind: Bestiary[int(u.Kind)%len(Bestiary)].Name, X: int(u.X), Y: int(u.Y), HP: int(u.HP)})
		} else {
			b.Enemies = append(b.Enemies, RpgFoe{X: int(u.X), Y: int(u.Y), HP: int(u.HP)})
		}
	}
	return b
}

// BrainCommander drives a faction with any tvcp-ai brain via game "rpg". On an
// empty or failed reply it falls back to the reference charge, so a quiet model
// still fights.
type BrainCommander struct {
	B brain.Brain
}

func clampDir(v int) int {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}

// Plan implements Commander by asking the brain for per-unit moves.
func (c BrainCommander) Plan(a *Arena, faction uint8) []Action {
	raw, _ := json.Marshal(a.Brief(faction))
	resp, err := c.B.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "rpg", State: raw})
	if err != nil || len(resp.Rpg) == 0 {
		return RefCommander{}.Plan(a, faction)
	}
	var moves []RpgMove
	if json.Unmarshal(resp.Rpg, &moves) != nil {
		return RefCommander{}.Plan(a, faction)
	}
	var acts []Action
	for _, m := range moves {
		if m.ID < 0 || m.ID >= len(a.Units) || !a.Units[m.ID].Alive || a.Units[m.ID].Faction != faction {
			continue
		}
		acts = append(acts, Action{Unit: m.ID, DX: clampDir(m.DX), DY: clampDir(m.DY), Attack: -1})
	}
	if len(acts) == 0 {
		return RefCommander{}.Plan(a, faction)
	}
	return acts
}
