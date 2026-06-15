package republic_test

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/republic"
)

func cost(n int) func(int) int { return func(int) int { return n } }

// The republic must self-organize: register everyone, award the two seats to the
// two lowest bidders (distinct), then actually fight a match.
func TestConveneNegotiatesAndFights(t *testing.T) {
	cands := []republic.Candidate{
		{Name: "scout", Brain: brain.Local{}, Cost: cost(3)},
		{Name: "planner", Brain: brain.Local{}, Cost: cost(4)},
		{Name: "warden", Brain: brain.Local{}, Cost: cost(5)},
	}
	r := republic.Convene(cands, 7)

	if len(r.Registered) != 3 {
		t.Fatalf("registered %d, want 3", len(r.Registered))
	}
	if len(r.Seats) != 2 || !r.Seats[0].Awarded || !r.Seats[1].Awarded {
		t.Fatalf("both seats must be awarded: %+v", r.Seats)
	}
	if r.Seats[0].Brain == r.Seats[1].Brain {
		t.Fatalf("the two factions must get different commanders")
	}
	if r.Seats[0].Brain != "scout" {
		t.Errorf("faction 0 should go to scout (lowest bid 3), got %q", r.Seats[0].Brain)
	}
	if r.Seats[1].Brain != "planner" {
		t.Errorf("faction 1 should go to planner (next lowest bid 4), got %q", r.Seats[1].Brain)
	}
	if r.Ticks <= 0 {
		t.Error("the battle did not run")
	}
	if r.Final == nil || len(r.Final.Units) == 0 {
		t.Error("arena was not built")
	}
}
