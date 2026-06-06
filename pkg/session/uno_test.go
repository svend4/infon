package session_test

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func TestUnoDeckComposition(t *testing.T) {
	d := session.UnoDeck()
	if len(d) != 108 {
		t.Fatalf("deck has %d cards, want 108", len(d))
	}
	var wild, wild4, red, nums int
	for _, c := range d {
		switch c.Kind {
		case "wild":
			wild++
		case "wild4":
			wild4++
		case "num":
			nums++
		}
		if c.Color == "R" {
			red++
		}
	}
	if wild != 4 || wild4 != 4 {
		t.Errorf("wild=%d wild4=%d, want 4/4", wild, wild4)
	}
	if red != 25 {
		t.Errorf("red cards=%d, want 25", red)
	}
	if nums != 76 {
		t.Errorf("number cards=%d, want 76", nums)
	}
}

func TestUnoStartValid(t *testing.T) {
	s := session.Uno{Seed: 1}.Start()
	if len(s.Hands[0]) != 7 || len(s.Hands[1]) != 7 {
		t.Fatalf("hands %d/%d, want 7/7", len(s.Hands[0]), len(s.Hands[1]))
	}
	if len(s.Discard) != 1 || s.Discard[0].Kind != "num" {
		t.Fatalf("starting discard must be a single number card, got %v", s.Discard)
	}
	if len(s.Deck) != 93 {
		t.Errorf("deck=%d, want 93 (108-14-1)", len(s.Deck))
	}
	if s.Winner != -1 {
		t.Errorf("winner should be -1 at start")
	}
}

func TestUnoDrawTwoEffect(t *testing.T) {
	s := session.UnoState{
		Hands:   [2][]session.UnoCard{{{Color: "R", Kind: "draw2"}, {Color: "R", Kind: "num", Num: 9}}, {{Color: "B", Kind: "num", Num: 5}, {Color: "G", Kind: "num", Num: 3}}},
		Deck:    []session.UnoCard{{Color: "Y", Kind: "num", Num: 1}, {Color: "Y", Kind: "num", Num: 2}},
		Discard: []session.UnoCard{{Color: "R", Kind: "num", Num: 7}},
		Color:   "R", Turn: 0, Winner: -1,
	}
	ns, err := session.Uno{}.Apply(s, 0, []byte(`{"card_index":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Hands[1]) != 4 {
		t.Errorf("opponent should have drawn 2 (4 cards), has %d", len(ns.Hands[1]))
	}
	if ns.Turn != 0 {
		t.Errorf("Draw-Two should skip the opponent (turn stays 0), got %d", ns.Turn)
	}
	if ns.Color != "R" {
		t.Errorf("active color should be R, got %s", ns.Color)
	}
}

func TestUnoRejectsUnplayable(t *testing.T) {
	s := session.UnoState{
		Hands:   [2][]session.UnoCard{{{Color: "B", Kind: "num", Num: 2}}, {{Color: "G", Kind: "num", Num: 1}}},
		Discard: []session.UnoCard{{Color: "R", Kind: "num", Num: 7}},
		Color:   "R", Turn: 0, Winner: -1,
	}
	if _, err := (session.Uno{}).Apply(s, 0, []byte(`{"card_index":0}`)); err == nil {
		t.Fatal("playing Blue 2 on Red 7 (color Red) should be illegal")
	}
}

func TestUnoWinByEmptyingHand(t *testing.T) {
	s := session.UnoState{
		Hands:   [2][]session.UnoCard{{{Color: "R", Kind: "num", Num: 9}}, {{Color: "G", Kind: "num", Num: 1}}},
		Discard: []session.UnoCard{{Color: "R", Kind: "num", Num: 7}},
		Color:   "R", Turn: 0, Winner: -1,
	}
	ns, err := session.Uno{}.Apply(s, 0, []byte(`{"card_index":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if ns.Winner != 0 {
		t.Fatalf("emptying the hand should win: winner=%d", ns.Winner)
	}
}

// Two reference brains, each speaking game:uno and seeing only their own hand,
// play a full game to a winner on the session runner.
func TestUnoFullGameTwoBrains(t *testing.T) {
	players := []session.Participant{
		session.BrainPlayer{N: "ref-A", B: brain.Local{}},
		session.BrainPlayer{N: "ref-B", B: brain.Local{}},
	}
	r, err := session.Play[session.UnoState](session.Uno{Seed: 7}, players, 3000)
	if err != nil {
		t.Fatal(err)
	}
	if r.Winner < 0 {
		t.Fatalf("no winner after %d turns", r.Turns)
	}
	if len(r.Final.Hands[r.Winner]) != 0 {
		t.Fatalf("winner %d should have an empty hand, has %d", r.Winner, len(r.Final.Hands[r.Winner]))
	}
}
