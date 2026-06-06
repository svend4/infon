package acl

import "testing"

func TestContractNetAwardsLowest(t *testing.T) {
	bidders := []Bidder{
		FuncBidder{N: "cheap", F: func(t Task) (int, bool) { return t.Size, true }},
		FuncBidder{N: "mid", F: func(t Task) (int, bool) { return t.Size * 2, true }},
		FuncBidder{N: "busy", F: func(t Task) (int, bool) { return 0, false }},
	}
	out, log := ContractNet("boss", Task{ID: "t1", Name: "job", Size: 5}, bidders)
	if !out.Awarded || out.Winner != "cheap" || out.Cost != 5 {
		t.Fatalf("award = %+v, want cheap@5", out)
	}
	if len(log) != 9 { // 3 cfp + 2 propose + 1 refuse + 1 accept + 1 reject + 1 inform
		t.Fatalf("conversation has %d messages, want 9", len(log))
	}
	count := map[Performative]int{}
	for _, m := range log {
		count[m.Perf]++
	}
	if count[CFP] != 3 || count[Propose] != 2 || count[Refuse] != 1 ||
		count[Accept] != 1 || count[Reject] != 1 || count[Inform] != 1 {
		t.Fatalf("performative counts = %v", count)
	}
}

func TestNoBiddersNoAward(t *testing.T) {
	out, _ := ContractNet("boss", Task{ID: "t", Name: "x", Size: 1},
		[]Bidder{FuncBidder{N: "a", F: func(Task) (int, bool) { return 0, false }}})
	if out.Awarded {
		t.Fatal("should not award when everyone refuses")
	}
}

func TestTieBreakByName(t *testing.T) {
	bidders := []Bidder{
		FuncBidder{N: "zeta", F: func(Task) (int, bool) { return 3, true }},
		FuncBidder{N: "alpha", F: func(Task) (int, bool) { return 3, true }},
	}
	out, _ := ContractNet("boss", Task{ID: "t", Name: "x", Size: 1}, bidders)
	if out.Winner != "alpha" {
		t.Fatalf("equal bids should break to the lower name, got %s", out.Winner)
	}
}
