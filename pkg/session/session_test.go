package session

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func mustMove(r, c int) Move {
	m, _ := json.Marshal(rc{Row: r, Col: c})
	return m
}

func firstLegal(req brain.Request) brain.Response {
	var st brain.TicTacToeState
	_ = json.Unmarshal(req.State, &st)
	if len(st.Legal) > 0 {
		return brain.Response{Kind: "move", Move: &brain.Move{Row: st.Legal[0][0], Col: st.Legal[0][1]}}
	}
	return brain.Response{Kind: "move", Move: &brain.Move{}}
}

func TestDrawBetweenBrains(t *testing.T) {
	players := []Participant{
		BrainPlayer{N: "X", B: brain.Local{}},
		BrainPlayer{N: "O", B: brain.Local{}},
	}
	r, err := Play[Board](TicTacToe{}, players, 12)
	if err != nil {
		t.Fatal(err)
	}
	if r.Winner != -1 {
		t.Fatalf("two optimal players should draw, got winner %d", r.Winner)
	}
	if r.Turns != 9 {
		t.Fatalf("a full draw is 9 turns, got %d", r.Turns)
	}
}

func TestApplyRejectsOccupied(t *testing.T) {
	g := TicTacToe{}
	b := g.Start()
	b, err := g.Apply(b, 0, mustMove(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(b, 1, mustMove(1, 1)); err == nil {
		t.Fatal("an occupied cell was accepted")
	}
}

func TestBotDoesNotBeatOptimal(t *testing.T) {
	players := []Participant{
		BrainPlayer{N: "X", B: brain.Local{}}, // optimal
		FuncPlayer{N: "firstlegal", F: firstLegal},
	}
	r, err := Play[Board](TicTacToe{}, players, 12)
	if err != nil {
		t.Fatal(err)
	}
	if done, _ := (TicTacToe{}).Over(r.Final); !done {
		t.Fatal("game did not finish")
	}
	if r.Winner == 1 {
		t.Fatal("optimal X should never lose to a first-legal bot")
	}
}
