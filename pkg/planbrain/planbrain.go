// Package planbrain exposes the arena lookahead Planner as a tvcp-ai/1 brain, so
// "the planner is reachable by URL". For a game:rpg request carrying a rich
// RpgState (pkg/arena), it rebuilds the board, runs arena.Planner, and returns
// the planned per-unit moves; every other kind is delegated to the reference
// policy, so planbrain.Brain is a complete drop-in backend.
package planbrain

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
)

// Move is one unit's planned step on the wire (matches the rpg reply shape).
type Move struct {
	ID int `json:"id"`
	DX int `json:"dx"`
	DY int `json:"dy"`
}

// Brain is the planner as a tvcp-ai/1 backend.
type Brain struct{}

// Decide implements brain.Brain.
func (Brain) Decide(req brain.Request) (brain.Response, error) {
	if req.Kind != "move" || req.Game != "rpg" {
		return brain.Reference(req), nil
	}
	var bd arena.RpgBoard
	if err := json.Unmarshal(req.State, &bd); err != nil {
		return brain.Response{Protocol: brain.Protocol, Kind: "move", Error: "planbrain: bad rpg state: " + err.Error()}, nil
	}
	board := bd.Arena()
	acts := arena.Planner{}.Plan(board, bd.You)
	moves := make([]Move, 0, len(acts))
	for _, a := range acts {
		moves = append(moves, Move{ID: a.Unit, DX: a.DX, DY: a.DY})
	}
	data, _ := json.Marshal(moves)
	return brain.Response{Protocol: brain.Protocol, Kind: "move", Rpg: data, Reasoning: "planbrain: rollout planner"}, nil
}
