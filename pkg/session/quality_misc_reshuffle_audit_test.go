package session

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// unoExhaustedState is a game whose draw pile is empty and whose discard pile
// holds 40 distinct cards under the top card, in a known order.
func unoExhaustedState() UnoState {
	var discard []UnoCard
	for _, col := range unoColors {
		for n := 0; n <= 9; n++ {
			discard = append(discard, UnoCard{Color: col, Kind: "num", Num: n})
		}
	}
	discard = append(discard, UnoCard{Color: "R", Kind: "skip"}) // the top card
	return UnoState{Discard: discard, Color: "R", Winner: -1}
}

func unoDrawAll(t *testing.T, g Uno, s UnoState, n int) []UnoCard {
	t.Helper()
	for i := 0; i < n; i++ {
		ns, err := g.Apply(s, 0, []byte(`{"draw":true}`))
		if err != nil {
			t.Fatal(err)
		}
		s = ns
	}
	return s.Hands[0]
}

func unoKeys(cs []UnoCard) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = fmt.Sprintf("%s/%s/%d", c.Color, c.Kind, c.Num)
	}
	return out
}

// When the draw pile runs out, the discard pile (minus its top) is reshuffled
// back into it, as unoDraw documents: the next draws must not simply replay the
// discard pile in reverse order, every recycled card must still be drawn exactly
// once, and the reshuffle stays reproducible for a given seed.
func TestQualityMiscAudit_UnoReshufflesDiscard(t *testing.T) {
	s := unoExhaustedState()
	recycled := s.Discard[:len(s.Discard)-1]

	g := Uno{Seed: 7}
	drawn := unoDrawAll(t, g, s, len(recycled))

	reversed := make([]UnoCard, len(recycled))
	for i, c := range recycled {
		reversed[len(recycled)-1-i] = c
	}
	if reflect.DeepEqual(drawn, reversed) {
		t.Fatalf("after the deck ran out, draws replay the discard pile in reverse order (not reshuffled): %v", unoKeys(drawn))
	}

	got, want := unoKeys(drawn), unoKeys(recycled)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reshuffled draws are not the recycled discard pile:\n got %v\nwant %v", got, want)
	}

	if again := unoDrawAll(t, g, s, len(recycled)); !reflect.DeepEqual(again, drawn) {
		t.Fatalf("same state and seed gave a different reshuffle")
	}
	if s.Deck != nil || len(s.Discard) != len(recycled)+1 {
		t.Fatalf("Apply modified the caller's state")
	}
}
