package session_test

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func TestHanabiDeckComposition(t *testing.T) {
	d := session.HanabiDeck()
	if len(d) != 50 {
		t.Fatalf("deck has %d cards, want 50", len(d))
	}
	perColor := map[string]int{}
	perValue := map[int]int{}
	for _, c := range d {
		perColor[c.Color]++
		perValue[c.Num]++
	}
	for _, col := range []string{"R", "G", "B", "Y", "W"} {
		if perColor[col] != 10 {
			t.Errorf("colour %s has %d cards, want 10", col, perColor[col])
		}
	}
	if perValue[1] != 15 || perValue[5] != 5 {
		t.Errorf("value counts: ones=%d fives=%d, want 15/5", perValue[1], perValue[5])
	}
}

// the view shape Brief emits (mirrors session's internal hanView json tags)
type tvCard struct {
	Color      string `json:"color"`
	Num        int    `json:"num"`
	KnownColor string `json:"known_color"`
	KnownNum   int    `json:"known_num"`
}
type tvView struct {
	You      int            `json:"you"`
	Hints    int            `json:"hints"`
	Fuses    int            `json:"fuses"`
	Deck     int            `json:"deck"`
	Stacks   map[string]int `json:"stacks"`
	YourHand []tvCard       `json:"your_hand"`
	Mate     struct {
		Player int      `json:"player"`
		Hand   []tvCard `json:"hand"`
	} `json:"mate"`
}

// The defining property of Hanabi: a player sees the teammate's cards but NOT
// their own. Brief must redact the moving player's hand to clues only.
func TestHanabiBriefHidesYourOwnHand(t *testing.T) {
	s := session.Hanabi{Seed: 1}.Start()
	req := session.Hanabi{}.Brief(s, 0)
	var v tvView
	if err := json.Unmarshal(req.State, &v); err != nil {
		t.Fatal(err)
	}
	if v.Deck != 40 || v.Hints != 8 || v.Fuses != 3 {
		t.Errorf("deck=%d hints=%d fuses=%d, want 40/8/3", v.Deck, v.Hints, v.Fuses)
	}
	if len(v.YourHand) != 5 {
		t.Fatalf("your hand has %d slots, want 5", len(v.YourHand))
	}
	for i, c := range v.YourHand {
		if c.Color != "" || c.Num != 0 {
			t.Errorf("your card %d leaks identity (%s%d): you must not see your own cards", i, c.Color, c.Num)
		}
	}
	if len(v.Mate.Hand) != 5 {
		t.Fatalf("mate hand has %d cards, want 5", len(v.Mate.Hand))
	}
	for i, c := range v.Mate.Hand {
		if c.Color == "" || c.Num < 1 || c.Num > 5 {
			t.Errorf("mate card %d is not fully visible: %q %d", i, c.Color, c.Num)
		}
	}
}

// Two reference brains, each blind to its own hand, cooperate over the clue
// channel to build the fireworks — without ever blowing a fuse.
func TestHanabiSelfPlayCooperates(t *testing.T) {
	players := []session.Participant{
		session.BrainPlayer{N: "ref-A", B: brain.Local{}},
		session.BrainPlayer{N: "ref-B", B: brain.Local{}},
	}
	r, err := session.Play[session.HanabiState](session.Hanabi{Seed: 7}, players, 400)
	if err != nil {
		t.Fatal(err)
	}
	if done, _ := (session.Hanabi{}).Over(r.Final); !done {
		t.Fatalf("game did not finish in %d turns", r.Turns)
	}
	score := session.HanabiScore(r.Final)
	if score <= 0 {
		t.Fatalf("cooperating reference brains scored %d, want > 0", score)
	}
	if r.Final.Fuses != 3 {
		t.Errorf("reference policy blew %d fuse(s); it should only play fully-known cards", 3-r.Final.Fuses)
	}
}

func TestHanabiApplyRules(t *testing.T) {
	base := session.HanabiState{
		Hands: [2][]session.HanCard{
			{{Color: "R", Num: 1}, {Color: "G", Num: 2}},
			{{Color: "R", Num: 1}, {Color: "B", Num: 3}},
		},
		Known:  [2][]session.HanKnown{{{}, {}}, {{}, {}}},
		Stacks: map[string]int{"R": 0, "G": 0, "B": 0, "Y": 0, "W": 0},
		Hints:  8, Fuses: 3, Turn: 0, Last: -1,
		Deck: []session.HanCard{{Color: "W", Num: 1}},
	}
	g := session.Hanabi{}

	// a legal colour clue marks matching cards and spends a token
	ns, err := g.Apply(base, 0, []byte(`{"action":"hint","to":1,"kind":"color","color":"R"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ns.Known[1][0].Color != "R" || ns.Hints != 7 {
		t.Errorf("colour clue did not register: known=%q hints=%d", ns.Known[1][0].Color, ns.Hints)
	}

	// clue to yourself is illegal
	if _, err := g.Apply(base, 0, []byte(`{"action":"hint","to":0,"kind":"color","color":"R"}`)); err == nil {
		t.Error("cluing your own hand should be illegal")
	}
	// a clue touching zero cards is illegal
	if _, err := g.Apply(base, 0, []byte(`{"action":"hint","to":1,"kind":"color","color":"Y"}`)); err == nil {
		t.Error("a clue touching no cards should be illegal")
	}

	// an in-order play raises the stack with no fuse cost
	ns, err = g.Apply(base, 0, []byte(`{"action":"play","card":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if ns.Stacks["R"] != 1 || ns.Fuses != 3 || len(ns.Hands[0]) != 2 {
		t.Errorf("good play: stackR=%d fuses=%d hand=%d, want 1/3/2", ns.Stacks["R"], ns.Fuses, len(ns.Hands[0]))
	}
	// an out-of-order play blows a fuse
	ns, err = g.Apply(base, 0, []byte(`{"action":"play","card":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if ns.Fuses != 2 || ns.Stacks["G"] != 0 {
		t.Errorf("misplay: fuses=%d stackG=%d, want 2/0", ns.Fuses, ns.Stacks["G"])
	}
}
