// Package republic is the self-organizing agent-world: it wires three previously
// separate pieces into one act. Brains advertise themselves in a registry; an
// ACL Contract-Net round awards the two faction commands to the brains that bid
// for them; then the arena is fought out with those awarded brains as the
// commanders (each speaking the tvcp-ai game:rpg protocol). Nobody hard-codes who
// commands what — the world organizes itself: discover -> negotiate -> fight.
package republic

import (
	"fmt"
	"math/rand"

	"github.com/svend4/infon/pkg/acl"
	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/registry"
	"github.com/svend4/infon/pkg/tangram"
)

// Candidate is a brain offering to command, with the cost it bids for a seat
// (lower bids win the Contract-Net auction).
type Candidate struct {
	Name  string
	Brain brain.Brain
	Cost  func(faction int) int
}

// Seat is one faction command, as awarded by negotiation.
type Seat struct {
	Faction int    `json:"faction"`
	Brain   string `json:"brain"`
	Cost    int    `json:"cost"`
	Awarded bool   `json:"awarded"`
}

// Result is the outcome of convening the republic.
type Result struct {
	Registered []string     `json:"registered"`
	Seats      []Seat       `json:"seats"`
	Log        []string     `json:"log"`
	Winner     int          `json:"winner"` // faction index, or -1 for a draw
	Ticks      int          `json:"ticks"`
	Alive0     int          `json:"alive0"`
	Alive1     int          `json:"alive1"`
	Final      *arena.Arena `json:"-"`
}

// Convene runs the whole act: register, negotiate the two seats, then fight.
func Convene(cands []Candidate, seed int64) Result {
	var res Result
	res.Winner = -1

	// 1) Discovery: every candidate advertises a game:rpg capability.
	reg := registry.New()
	byName := map[string]Candidate{}
	for _, c := range cands {
		reg.Register(registry.Entry{Name: c.Name, URL: "local://" + c.Name, Kinds: []string{"move"}, Games: []string{"rpg"}})
		byName[c.Name] = c
	}
	commanders := reg.Find("move", "rpg") // who can command a faction
	avail := map[string]bool{}
	for _, e := range commanders {
		avail[e.Name] = true
		res.Registered = append(res.Registered, e.Name)
	}

	// 2) Negotiation: a Contract-Net round per faction. The lowest bidder still
	//    available wins each seat (and is then removed from the pool).
	for f := 0; f < 2; f++ {
		var bidders []acl.Bidder
		for _, c := range cands {
			if !avail[c.Name] {
				continue
			}
			cc, ff := c, f
			bidders = append(bidders, acl.FuncBidder{N: cc.Name, F: func(acl.Task) (int, bool) { return cc.Cost(ff), true }})
		}
		task := acl.Task{ID: fmt.Sprintf("faction-%d", f), Name: "command a faction (game:rpg)", Size: 10}
		out, msgs := acl.ContractNet("republic", task, bidders)
		for _, m := range msgs {
			res.Log = append(res.Log, fmt.Sprintf("%s -> %s : %s", m.From, m.To, m.Perf))
		}
		res.Seats = append(res.Seats, Seat{Faction: f, Brain: out.Winner, Cost: out.Cost, Awarded: out.Awarded})
		if out.Awarded {
			avail[out.Winner] = false
		}
	}

	// 3) The fight: the awarded brains command the arena via game:rpg.
	a := buildArena(seed)
	commander := func(f int) arena.Commander {
		if f < len(res.Seats) && res.Seats[f].Awarded {
			if c, ok := byName[res.Seats[f].Brain]; ok {
				return arena.BrainCommander{B: c.Brain}
			}
		}
		return arena.RefCommander{}
	}
	c0, c1 := commander(0), commander(1)
	ticks := 0
	for ticks < 200 && a.AliveCount(0) > 0 && a.AliveCount(1) > 0 {
		a.Step(c0, c1)
		ticks++
	}
	res.Ticks = ticks
	res.Alive0, res.Alive1 = a.AliveCount(0), a.AliveCount(1)
	switch {
	case res.Alive0 > res.Alive1:
		res.Winner = 0
	case res.Alive1 > res.Alive0:
		res.Winner = 1
	default:
		res.Winner = -1
	}
	res.Final = a
	return res
}

// buildArena seats two summoned armies of tangram-creatures on a small field.
func buildArena(seed int64) *arena.Arena {
	a := &arena.Arena{W: 14, H: 9, Terrain: make([]uint8, 14*9)}
	cat := tangram.Catalog()
	r := rand.New(rand.NewSource(seed))
	roster := []string{"cat", "fox", "owl", "crab", "turtle", "swan"}
	add := func(name string, f, x, y uint8) {
		if fig, ok := cat[name]; ok {
			if u, _, ok2 := arena.Summon(fig, f, x, y); ok2 {
				a.Units = append(a.Units, u)
			}
		}
	}
	for i := 0; i < 4; i++ {
		add(roster[r.Intn(len(roster))], 0, 2, uint8(1+i*2))
		add(roster[r.Intn(len(roster))], 1, 11, uint8(1+i*2))
	}
	return a
}
