// Package acl is a small agent communication layer in the FIPA-ACL spirit:
// agents exchange PERFORMATIVES (cfp, propose, accept, reject, inform...) to
// NEGOTIATE rather than merely compete. ContractNet runs the canonical Contract
// Net Protocol — an initiator calls for proposals, bidders propose or refuse, the
// best bid is awarded and the rest rejected, the winner reports done — returning
// the award plus the full conversation log. The bidders are an interface, so
// scripted strategies or live tvcp-ai brains can take part.
package acl

import (
	"fmt"
	"sort"
)

// Performative is an ACL message type.
type Performative string

const (
	CFP    Performative = "cfp"             // call for proposals
	Propose Performative = "propose"
	Refuse  Performative = "refuse"
	Accept  Performative = "accept-proposal"
	Reject  Performative = "reject-proposal"
	Inform  Performative = "inform"
	Failure Performative = "failure"
)

// Message is one ACL message in a conversation.
type Message struct {
	From, To string
	Perf     Performative
	Convo    string
	Content  string
}

// Task is the work being contracted out.
type Task struct {
	ID   string
	Name string
	Size int
}

// Bidder evaluates a task and either bids a cost or refuses (ok=false).
type Bidder interface {
	Name() string
	Bid(t Task) (cost int, ok bool)
}

// FuncBidder adapts a function into a Bidder (a scripted strategy).
type FuncBidder struct {
	N string
	F func(t Task) (int, bool)
}

func (b FuncBidder) Name() string             { return b.N }
func (b FuncBidder) Bid(t Task) (int, bool)    { return b.F(t) }

// Outcome is the result of a negotiation.
type Outcome struct {
	Task    Task
	Winner  string
	Cost    int
	Awarded bool
}

// ContractNet runs the protocol and returns the award and the conversation log.
func ContractNet(initiator string, t Task, bidders []Bidder) (Outcome, []Message) {
	var log []Message
	emit := func(from, to string, p Performative, content string) {
		log = append(log, Message{From: from, To: to, Perf: p, Convo: t.ID, Content: content})
	}

	cfp := fmt.Sprintf("task %q (size %d): who can do it, and at what cost?", t.Name, t.Size)
	for _, b := range bidders {
		emit(initiator, b.Name(), CFP, cfp)
	}

	type bid struct {
		name string
		cost int
	}
	var bids []bid
	for _, b := range bidders {
		cost, ok := b.Bid(t)
		if ok {
			bids = append(bids, bid{b.Name(), cost})
			emit(b.Name(), initiator, Propose, fmt.Sprintf("cost=%d", cost))
		} else {
			emit(b.Name(), initiator, Refuse, "cannot take it on")
		}
	}

	if len(bids) == 0 {
		return Outcome{Task: t, Awarded: false}, log
	}
	sort.SliceStable(bids, func(i, j int) bool {
		if bids[i].cost != bids[j].cost {
			return bids[i].cost < bids[j].cost
		}
		return bids[i].name < bids[j].name
	})
	win := bids[0]
	emit(initiator, win.name, Accept, fmt.Sprintf("awarded at cost=%d", win.cost))
	for _, b := range bids[1:] {
		emit(initiator, b.name, Reject, "a lower bid won")
	}
	emit(win.name, initiator, Inform, "done")
	return Outcome{Task: t, Winner: win.name, Cost: win.cost, Awarded: true}, log
}
