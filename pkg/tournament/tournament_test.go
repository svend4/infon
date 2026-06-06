package tournament

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func firstLegal(req brain.Request) brain.Response {
	var st brain.TicTacToeState
	_ = json.Unmarshal(req.State, &st)
	if len(st.Legal) > 0 {
		return brain.Response{Kind: "move", Move: &brain.Move{Row: st.Legal[0][0], Col: st.Legal[0][1]}}
	}
	return brain.Response{Kind: "move", Move: &brain.Move{}}
}

func lastLegal(req brain.Request) brain.Response {
	var st brain.TicTacToeState
	_ = json.Unmarshal(req.State, &st)
	if n := len(st.Legal); n > 0 {
		return brain.Response{Kind: "move", Move: &brain.Move{Row: st.Legal[n-1][0], Col: st.Legal[n-1][1]}}
	}
	return brain.Response{Kind: "move", Move: &brain.Move{}}
}

func TestRoundRobinInvariants(t *testing.T) {
	players := []session.Participant{
		session.BrainPlayer{N: "optimal", B: brain.Local{}},
		session.FuncPlayer{N: "first", F: firstLegal},
		session.FuncPlayer{N: "last", F: lastLegal},
	}
	tbl := RoundRobin[session.Board](session.TicTacToe{}, players, 12)

	n := len(players)
	if len(tbl.Standings) != n {
		t.Fatalf("standings %d, want %d", len(tbl.Standings), n)
	}
	if len(tbl.Matches) != n*(n-1) {
		t.Fatalf("matches %d, want %d (each pair home+away)", len(tbl.Matches), n*(n-1))
	}
	games := 2 * (n - 1)
	for _, s := range tbl.Standings {
		if s.Wins+s.Losses+s.Draws != games {
			t.Fatalf("%s played %d games, want %d", s.Name, s.Wins+s.Losses+s.Draws, games)
		}
		if s.Name == "optimal" && s.Losses != 0 {
			t.Fatalf("optimal lost %d games (minimax should never lose)", s.Losses)
		}
	}
}
