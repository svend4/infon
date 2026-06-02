package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"github.com/svend4/infon/pkg/brain"
)

// uno.go is the FOURTH worked Game on the session runner and the first with
// HIDDEN INFORMATION and CHANCE: two-player UNO. Each player sees only their own
// hand (Brief redacts the opponent's), the 108-card deck is shuffled from a seed,
// and a move is "play a playable card (choosing a color for a wild) or draw one".
// It speaks the existing tvcp-ai game:uno protocol, so the reference brain plays
// unchanged. With two players Reverse acts as Skip; Skip/Draw-Two/Wild-Draw-Four
// keep the turn with the player who played them.

var unoColors = []string{"R", "G", "B", "Y"}

var unoColorWord = map[string]string{"R": "Red", "G": "Green", "B": "Blue", "Y": "Yellow"}
var unoWordColor = map[string]string{"Red": "R", "Green": "G", "Blue": "B", "Yellow": "Y"}

// UnoCard is one card. Color is "" for wild/wild4; Num applies to kind "num".
type UnoCard struct {
	Color string `json:"color,omitempty"`
	Kind  string `json:"kind"` // num | skip | reverse | draw2 | wild | wild4
	Num   int    `json:"num,omitempty"`
}

// String renders a card with full color words (the tvcp-ai uno vocabulary).
func (c UnoCard) String() string {
	w := unoColorWord[c.Color]
	switch c.Kind {
	case "num":
		return fmt.Sprintf("%s %d", w, c.Num)
	case "skip":
		return w + " Skip"
	case "reverse":
		return w + " Reverse"
	case "draw2":
		return w + " Draw Two"
	case "wild":
		return "Wild"
	case "wild4":
		return "Wild Draw Four"
	}
	return "?"
}

func (c UnoCard) isWild() bool { return c.Kind == "wild" || c.Kind == "wild4" }

// UnoState is the full game state. Both hands are present; Brief redacts the
// opponent's so a player only ever sees its own cards.
type UnoState struct {
	Hands   [2][]UnoCard `json:"hands"`
	Deck    []UnoCard    `json:"deck"`
	Discard []UnoCard    `json:"discard"`
	Color   string       `json:"color"` // active color code
	Turn    int          `json:"turn"`
	Winner  int          `json:"winner"` // -1 until someone empties their hand
}

// UnoDeck builds a standard 108-card deck: per color one 0, two each 1-9, two
// each Skip/Reverse/Draw-Two; plus four Wild and four Wild-Draw-Four.
func UnoDeck() []UnoCard {
	var d []UnoCard
	for _, col := range unoColors {
		d = append(d, UnoCard{Color: col, Kind: "num", Num: 0})
		for n := 1; n <= 9; n++ {
			d = append(d, UnoCard{Color: col, Kind: "num", Num: n})
			d = append(d, UnoCard{Color: col, Kind: "num", Num: n})
		}
		for _, k := range []string{"skip", "reverse", "draw2"} {
			d = append(d, UnoCard{Color: col, Kind: k})
			d = append(d, UnoCard{Color: col, Kind: k})
		}
	}
	for i := 0; i < 4; i++ {
		d = append(d, UnoCard{Kind: "wild"})
		d = append(d, UnoCard{Kind: "wild4"})
	}
	return d
}

// Uno implements Game[UnoState]. Seed drives the deterministic shuffle.
type Uno struct {
	Seed int64
}

func (Uno) Name() string { return "uno" }

func (u Uno) Start() UnoState {
	d := UnoDeck()
	r := rand.New(rand.NewSource(u.Seed))
	r.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	s := UnoState{Winner: -1}
	for k := 0; k < 7; k++ { // deal 7 each, popping from the end
		for p := 0; p < 2; p++ {
			s.Hands[p] = append(s.Hands[p], d[len(d)-1])
			d = d[:len(d)-1]
		}
	}
	for { // starting discard: first number card (rotate actions/wilds to bottom)
		top := d[len(d)-1]
		d = d[:len(d)-1]
		if top.Kind == "num" {
			s.Discard = []UnoCard{top}
			s.Color = top.Color
			break
		}
		d = append([]UnoCard{top}, d...)
	}
	s.Deck = d
	return s
}

func unoPlayable(card, top UnoCard, color string) bool {
	switch {
	case card.isWild():
		return true
	case card.Color == color:
		return true
	case top.Kind == "num" && card.Kind == "num" && card.Num == top.Num:
		return true
	case card.Kind == top.Kind && (card.Kind == "skip" || card.Kind == "reverse" || card.Kind == "draw2"):
		return true
	}
	return false
}

func unoLegal(hand []UnoCard, top UnoCard, color string) []int {
	var l []int
	for i, c := range hand {
		if unoPlayable(c, top, color) {
			l = append(l, i)
		}
	}
	return l
}

func (Uno) Turn(s UnoState) int {
	if s.Winner >= 0 {
		return -1
	}
	return s.Turn
}

func (Uno) Over(s UnoState) (bool, int) {
	if s.Winner >= 0 {
		return true, s.Winner
	}
	return false, -1
}

func (Uno) Brief(s UnoState, p int) brain.Request {
	top := s.Discard[len(s.Discard)-1]
	hand := make([]string, len(s.Hands[p]))
	for i, c := range s.Hands[p] {
		hand[i] = c.String()
	}
	st := struct {
		Hand     []string `json:"hand"`
		Color    string   `json:"color"`
		Top      string   `json:"top"`
		OppCount int      `json:"opp_count"`
		Playable []int    `json:"playable"`
	}{
		Hand:     hand,
		Color:    unoColorWord[s.Color],
		Top:      top.String(),
		OppCount: len(s.Hands[1-p]),
		Playable: unoLegal(s.Hands[p], top, s.Color),
	}
	data, _ := json.Marshal(st)
	return brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "uno", State: data}
}

type unoMove struct {
	CardIndex *int   `json:"card_index,omitempty"`
	Draw      bool   `json:"draw,omitempty"`
	Color     string `json:"color,omitempty"`
}

func (Uno) Parse(resp brain.Response) (Move, error) {
	if resp.Move == nil {
		return nil, errors.New("no move in response")
	}
	if resp.Move.CardIndex == nil && !resp.Move.Draw {
		return nil, errors.New("uno move is neither a card_index nor draw")
	}
	return json.Marshal(unoMove{CardIndex: resp.Move.CardIndex, Draw: resp.Move.Draw, Color: resp.Move.Color})
}

func unoCloneHands(h [2][]UnoCard) [2][]UnoCard {
	var out [2][]UnoCard
	for p := 0; p < 2; p++ {
		out[p] = append([]UnoCard(nil), h[p]...)
	}
	return out
}

// unoDraw pops n cards from the deck into hand p, reshuffling the discard (minus
// its top) back into the deck when the deck runs out.
func unoDraw(s *UnoState, p, n int) {
	for k := 0; k < n; k++ {
		if len(s.Deck) == 0 {
			if len(s.Discard) <= 1 {
				return // nothing left to draw
			}
			top := s.Discard[len(s.Discard)-1]
			s.Deck = append([]UnoCard(nil), s.Discard[:len(s.Discard)-1]...)
			s.Discard = []UnoCard{top}
		}
		s.Hands[p] = append(s.Hands[p], s.Deck[len(s.Deck)-1])
		s.Deck = s.Deck[:len(s.Deck)-1]
	}
}

func (Uno) Apply(s UnoState, p int, m Move) (UnoState, error) {
	var v unoMove
	if json.Unmarshal(m, &v) != nil {
		return s, errors.New("bad move encoding")
	}
	top := s.Discard[len(s.Discard)-1]
	ns := s
	ns.Hands = unoCloneHands(s.Hands)
	ns.Deck = append([]UnoCard(nil), s.Deck...)
	ns.Discard = append([]UnoCard(nil), s.Discard...)
	opp := 1 - p

	if v.Draw {
		unoDraw(&ns, p, 1)
		ns.Turn = opp
		return ns, nil
	}
	if v.CardIndex == nil {
		return s, errors.New("no card index")
	}
	idx := *v.CardIndex
	if idx < 0 || idx >= len(ns.Hands[p]) {
		return s, fmt.Errorf("card index %d out of range", idx)
	}
	card := ns.Hands[p][idx]
	if !unoPlayable(card, top, s.Color) {
		return s, fmt.Errorf("card %q not playable on %q (color %s)", card, top, unoColorWord[s.Color])
	}
	ns.Hands[p] = append(ns.Hands[p][:idx], ns.Hands[p][idx+1:]...)
	ns.Discard = append(ns.Discard, card)
	if card.isWild() {
		col := unoWordColor[v.Color]
		if col == "" {
			col = unoPickColor(ns.Hands[p])
		}
		ns.Color = col
	} else {
		ns.Color = card.Color
	}
	if len(ns.Hands[p]) == 0 {
		ns.Winner, ns.Turn = p, -1
		return ns, nil
	}
	switch card.Kind {
	case "skip", "reverse": // 2 players: opponent is skipped
		ns.Turn = p
	case "draw2":
		unoDraw(&ns, opp, 2)
		ns.Turn = p
	case "wild4":
		unoDraw(&ns, opp, 4)
		ns.Turn = p
	default: // num, wild
		ns.Turn = opp
	}
	return ns, nil
}

// unoPickColor chooses the most common color in a hand (fallback for a wild with
// no explicit color), defaulting to Red.
func unoPickColor(hand []UnoCard) string {
	count := map[string]int{}
	for _, c := range hand {
		if c.Color != "" {
			count[c.Color]++
		}
	}
	best, bn := "R", -1
	for _, col := range unoColors {
		if count[col] > bn {
			bn, best = count[col], col
		}
	}
	return best
}
