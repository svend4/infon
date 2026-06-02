package planbrain

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
)

func board() *arena.Arena {
	a := &arena.Arena{W: 8, H: 6, Terrain: make([]uint8, 48)}
	for x := 0; x < 8; x++ {
		a.Terrain[2*8+x] = 2
	}
	a.Units = []arena.Unit{
		{Kind: 4, X: 1, Y: 1, HP: 5, Faction: 0, Alive: true},
		{Kind: 3, X: 1, Y: 3, HP: 5, Faction: 0, Alive: true},
		{Kind: 0, X: 6, Y: 1, HP: 6, Faction: 1, Alive: true},
		{Kind: 5, X: 6, Y: 3, HP: 8, Faction: 1, Alive: true},
	}
	return a
}

// The brain must reproduce the in-process Planner's decision after a JSON round
// trip through the rich rpg state.
func TestPlanbrainMatchesPlanner(t *testing.T) {
	a := board()
	st := arena.MarshalRpgBoard(a, 0)
	stJSON, _ := json.Marshal(st)

	resp, err := Brain{}.Decide(brain.Request{Kind: "move", Game: "rpg", State: stJSON})
	if err != nil {
		t.Fatal(err)
	}
	var got []Move
	if json.Unmarshal(resp.Rpg, &got) != nil {
		t.Fatalf("bad rpg reply: %s", resp.Rpg)
	}

	want := arena.Planner{}.Plan(st.Arena(), 0)
	if len(got) != len(want) {
		t.Fatalf("got %d moves, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].Unit || got[i].DX != want[i].DX || got[i].DY != want[i].DY {
			t.Fatalf("move %d = %+v, want id=%d dx=%d dy=%d", i, got[i], want[i].Unit, want[i].DX, want[i].DY)
		}
	}
	if len(got) == 0 {
		t.Fatal("planner returned no moves for a live faction")
	}
}

func TestRpgStateRoundTrip(t *testing.T) {
	a := board()
	rb := arena.MarshalRpgBoard(a, 1).Arena()
	if len(rb.Units) != len(a.Units) || rb.W != a.W || rb.H != a.H {
		t.Fatalf("rebuilt board mismatch")
	}
	for i := range a.Units {
		if rb.Units[i] != a.Units[i] {
			t.Fatalf("unit %d: rebuilt %+v != %+v", i, rb.Units[i], a.Units[i])
		}
	}
}

func TestDelegatesNonRpg(t *testing.T) {
	st, _ := json.Marshal(brain.TicTacToeState{Turn: "X", You: "X", Legal: [][2]int{{0, 0}}})
	resp, err := Brain{}.Decide(brain.Request{Kind: "move", Game: "tictactoe", State: st})
	if err != nil || resp.Move == nil {
		t.Fatalf("delegate to reference failed: %+v err=%v", resp, err)
	}
}
