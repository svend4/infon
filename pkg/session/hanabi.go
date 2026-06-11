package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/svend4/infon/pkg/brain"
)

// hanabi.go is the FIFTH worked Game on the session runner and the first that is
// COOPERATIVE: two players build five firework stacks together and share one
// score (0..25). It is the purest expression of this whole project's thesis —
// transmitting MEANING through a tiny constrained channel. A player sees every
// hand BUT their own; the only way knowledge crosses between hands is a single
// colour-or-number clue, of which there are at most eight in flight. Speaking
// well through that ~30-byte channel is the entire game.
//
// It speaks a new tvcp-ai protocol, game:hanabi, so the reference brain (and any
// LLM that implements it) plays unchanged. The reference policy never risks a
// fuse: it only plays cards it fully knows and spends clues to make the partner's
// playable cards knowable.

var hanabiColors = []string{"R", "G", "B", "Y", "W"}

var hanabiColorWord = map[string]string{"R": "Red", "G": "Green", "B": "Blue", "Y": "Yellow", "W": "White"}

// hanMult[v] is how many copies of value v exist per colour (1s are common, 5s
// unique): three 1s, two each of 2/3/4, one 5 — the standard Hanabi deck.
var hanMult = []int{0, 3, 2, 2, 2, 1}

// HanCard is one Hanabi card: a colour (R/G/B/Y/W) and a value 1..5.
type HanCard struct {
	Color string `json:"color"`
	Num   int    `json:"num"`
}

// Code renders a card as e.g. "R1" (used in the discard log and demos).
func (c HanCard) Code() string { return c.Color + strconv.Itoa(c.Num) }

// HanKnown is what a player has been TOLD about one of their own cards (clues are
// public, so both players track this identically). Empty fields mean "not told".
type HanKnown struct {
	Color string `json:"color,omitempty"`
	Num   int    `json:"num,omitempty"`
}

// HanabiState is the full game. Hands are present for the engine; Brief redacts
// the moving player's own cards down to HanKnown so they only ever see clues.
type HanabiState struct {
	Hands   [2][]HanCard   `json:"hands"`
	Known   [2][]HanKnown  `json:"known"`
	Deck    []HanCard      `json:"deck"`
	Discard []HanCard      `json:"discard"`
	Stacks  map[string]int `json:"stacks"` // colour -> highest value played (0 = empty)
	Hints   int            `json:"hints"`  // clue tokens available (0..8)
	Fuses   int            `json:"fuses"`  // mistakes remaining (start 3)
	Turn    int            `json:"turn"`
	Last    int            `json:"last"` // -1 until the deck empties, then a final-turn countdown
}

// HanabiScore is the sum of the firework stacks (max 25).
func HanabiScore(s HanabiState) int {
	n := 0
	for _, col := range hanabiColors {
		n += s.Stacks[col]
	}
	return n
}

// HanabiDeck builds the standard 50-card deck.
func HanabiDeck() []HanCard {
	var d []HanCard
	for _, col := range hanabiColors {
		for n := 1; n <= 5; n++ {
			for k := 0; k < hanMult[n]; k++ {
				d = append(d, HanCard{Color: col, Num: n})
			}
		}
	}
	return d
}

// Hanabi implements Game[HanabiState]. Seed drives the deterministic shuffle.
type Hanabi struct {
	Seed int64
}

func (Hanabi) Name() string { return "hanabi" }

func (h Hanabi) Start() HanabiState {
	d := HanabiDeck()
	r := rand.New(rand.NewSource(h.Seed))
	r.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	s := HanabiState{Stacks: map[string]int{}, Hints: 8, Fuses: 3, Last: -1}
	for _, col := range hanabiColors {
		s.Stacks[col] = 0
	}
	for k := 0; k < 5; k++ { // deal 5 each, popping from the end
		for p := 0; p < 2; p++ {
			s.Hands[p] = append(s.Hands[p], d[len(d)-1])
			s.Known[p] = append(s.Known[p], HanKnown{})
			d = d[:len(d)-1]
		}
	}
	s.Deck = d
	return s
}

func hanDone(s HanabiState) bool {
	return HanabiScore(s) >= 25 || s.Fuses <= 0 || s.Last == 0
}

func (Hanabi) Turn(s HanabiState) int {
	if hanDone(s) {
		return -1
	}
	return s.Turn
}

// Over reports completion. Hanabi is cooperative, so there is no winning player:
// the winner field is always -1 and the shared score lives in the final state.
func (Hanabi) Over(s HanabiState) (bool, int) {
	return hanDone(s), -1
}

// ---- the Brief: see everyone but yourself ---------------------------------

type hanViewCard struct {
	Color      string `json:"color,omitempty"`       // teammate cards only (you can see them)
	Num        int    `json:"num,omitempty"`         // teammate cards only
	KnownColor string `json:"known_color,omitempty"` // what clues have revealed (public)
	KnownNum   int    `json:"known_num,omitempty"`
}

type hanViewMate struct {
	Player int           `json:"player"`
	Hand   []hanViewCard `json:"hand"`
}

type hanView struct {
	You      int            `json:"you"`
	Score    int            `json:"score"`
	Hints    int            `json:"hints"`
	Fuses    int            `json:"fuses"`
	Deck     int            `json:"deck"`
	Stacks   map[string]int `json:"stacks"`
	YourHand []hanViewCard  `json:"your_hand"`
	Mate     hanViewMate    `json:"mate"`
	Discard  []string       `json:"discard"`
}

func (Hanabi) Brief(s HanabiState, p int) brain.Request {
	mate := 1 - p
	yh := make([]hanViewCard, len(s.Hands[p])) // own cards: only what clues revealed
	for i := range s.Hands[p] {
		yh[i] = hanViewCard{KnownColor: s.Known[p][i].Color, KnownNum: s.Known[p][i].Num}
	}
	mh := make([]hanViewCard, len(s.Hands[mate])) // teammate: full cards + public knowledge
	for i := range s.Hands[mate] {
		mh[i] = hanViewCard{
			Color: s.Hands[mate][i].Color, Num: s.Hands[mate][i].Num,
			KnownColor: s.Known[mate][i].Color, KnownNum: s.Known[mate][i].Num,
		}
	}
	ds := make([]string, len(s.Discard))
	for i, c := range s.Discard {
		ds[i] = c.Code()
	}
	v := hanView{
		You: p, Score: HanabiScore(s), Hints: s.Hints, Fuses: s.Fuses, Deck: len(s.Deck),
		Stacks: s.Stacks, YourHand: yh, Mate: hanViewMate{Player: mate, Hand: mh}, Discard: ds,
	}
	data, _ := json.Marshal(v)
	return brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "hanabi", State: data}
}

// ---- moves ----------------------------------------------------------------

type hanMove struct {
	Action string `json:"action"`          // play | discard | hint
	Card   int    `json:"card,omitempty"`  // index for play/discard
	To     int    `json:"to,omitempty"`    // hint target
	Kind   string `json:"kind,omitempty"`  // color | number
	Color  string `json:"color,omitempty"` // colour code for a colour clue
	Num    int    `json:"num,omitempty"`   // value for a number clue
}

func (Hanabi) Parse(resp brain.Response) (Move, error) {
	m := resp.Move
	if m == nil {
		return nil, errors.New("no move in response")
	}
	switch m.Action {
	case "play", "discard":
		if m.CardIndex == nil {
			return nil, fmt.Errorf("hanabi %s needs a card_index", m.Action)
		}
		return json.Marshal(hanMove{Action: m.Action, Card: *m.CardIndex})
	case "hint":
		if m.Hint == nil {
			return nil, errors.New("hanabi hint needs a hint")
		}
		return json.Marshal(hanMove{Action: "hint", To: m.Hint.To, Kind: m.Hint.Kind, Color: m.Hint.Color, Num: m.Hint.Num})
	default:
		return nil, fmt.Errorf("hanabi action %q is not play/discard/hint", m.Action)
	}
}

func hanClone(s HanabiState) HanabiState {
	ns := s
	for p := 0; p < 2; p++ {
		ns.Hands[p] = append([]HanCard(nil), s.Hands[p]...)
		ns.Known[p] = append([]HanKnown(nil), s.Known[p]...)
	}
	ns.Deck = append([]HanCard(nil), s.Deck...)
	ns.Discard = append([]HanCard(nil), s.Discard...)
	ns.Stacks = map[string]int{}
	for k, v := range s.Stacks {
		ns.Stacks[k] = v
	}
	return ns
}

func hanRemove(ns *HanabiState, p, i int) {
	ns.Hands[p] = append(ns.Hands[p][:i], ns.Hands[p][i+1:]...)
	ns.Known[p] = append(ns.Known[p][:i], ns.Known[p][i+1:]...)
}

func hanDrawCard(ns *HanabiState, p int) {
	if len(ns.Deck) == 0 {
		return
	}
	ns.Hands[p] = append(ns.Hands[p], ns.Deck[len(ns.Deck)-1])
	ns.Known[p] = append(ns.Known[p], HanKnown{})
	ns.Deck = ns.Deck[:len(ns.Deck)-1]
}

func (Hanabi) Apply(s HanabiState, p int, m Move) (HanabiState, error) {
	var v hanMove
	if json.Unmarshal(m, &v) != nil {
		return s, errors.New("bad move encoding")
	}
	ns := hanClone(s)
	switch v.Action {
	case "play":
		if v.Card < 0 || v.Card >= len(ns.Hands[p]) {
			return s, fmt.Errorf("play index %d out of range", v.Card)
		}
		card := ns.Hands[p][v.Card]
		hanRemove(&ns, p, v.Card)
		if ns.Stacks[card.Color]+1 == card.Num { // a correct, in-order play
			ns.Stacks[card.Color] = card.Num
			if card.Num == 5 && ns.Hints < 8 { // a finished firework returns a clue
				ns.Hints++
			}
		} else { // a misplay: the card is wasted and a fuse blows
			ns.Discard = append(ns.Discard, card)
			ns.Fuses--
		}
		hanDrawCard(&ns, p)
	case "discard":
		if v.Card < 0 || v.Card >= len(ns.Hands[p]) {
			return s, fmt.Errorf("discard index %d out of range", v.Card)
		}
		card := ns.Hands[p][v.Card]
		hanRemove(&ns, p, v.Card)
		ns.Discard = append(ns.Discard, card)
		if ns.Hints < 8 {
			ns.Hints++
		}
		hanDrawCard(&ns, p)
	case "hint":
		if ns.Hints <= 0 {
			return s, errors.New("no clue tokens left")
		}
		if v.To != 1-p {
			return s, fmt.Errorf("a clue must go to the teammate (player %d), not %d", 1-p, v.To)
		}
		touched := 0
		for i := range ns.Hands[v.To] {
			c := ns.Hands[v.To][i]
			switch v.Kind {
			case "color":
				if c.Color == v.Color {
					ns.Known[v.To][i].Color = v.Color
					touched++
				}
			case "number":
				if c.Num == v.Num {
					ns.Known[v.To][i].Num = v.Num
					touched++
				}
			default:
				return s, fmt.Errorf("clue kind %q must be color or number", v.Kind)
			}
		}
		if touched == 0 {
			return s, errors.New("a clue must touch at least one of the teammate's cards")
		}
		ns.Hints--
	default:
		return s, fmt.Errorf("action %q is not play/discard/hint", v.Action)
	}
	ns.Turn = 1 - p
	if len(ns.Deck) == 0 { // endgame: once the deck empties each player gets one last turn
		if ns.Last < 0 {
			ns.Last = 2
		}
		if ns.Last > 0 {
			ns.Last--
		}
	}
	return ns, nil
}
