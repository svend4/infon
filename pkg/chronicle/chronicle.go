// Package chronicle turns an arena match into an emergent NARRATIVE. It replays
// the match and, by diffing the few-byte state between ticks, records a compact
// byte-log of events (a strike, a fall, the close). The log is the "byte
// chronicle"; Narrate retells it as prose. It closes the project's chain on one
// tiny alphabet: number -> image -> world -> story.
package chronicle

import (
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/arena"
)

// Kind enumerates chronicle events.
type Kind uint8

const (
	Open Kind = iota
	Strike
	Fall
	Close
)

// Event is one beat. Fields are reused by kind: Strike{Actor,Target,Amt};
// Fall{Target}; Open/Close{Actor=blue count, Target=red count, Faction=winner}.
type Event struct {
	Tick    int
	Kind    Kind
	Actor   int
	Target  int
	Amt     int
	Faction int
}

// UnitInfo names a unit index for retelling.
type UnitInfo struct {
	Name    string
	Faction int
}

// Log is a self-contained chronicle: the roster plus the event stream.
type Log struct {
	W, H   int
	Roster []UnitInfo
	Events []Event
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// striker attributes damage to the strongest enemy that had unit i within its
// range at the given (pre-step) positions — the likely author of the blow.
func striker(a *arena.Arena, i int, px, py []int) int {
	best, bestAtk := -1, -1
	ux, uy := px[i], py[i]
	for j := range a.Units {
		e := a.Units[j]
		if e.Faction == a.Units[i].Faction {
			continue
		}
		k := arena.Bestiary[int(e.Kind)%len(arena.Bestiary)]
		if abs(px[j]-ux)+abs(py[j]-uy) <= int(k.Range) && int(k.Atk) > bestAtk {
			bestAtk, best = int(k.Atk), j
		}
	}
	return best
}

// Record replays a match between two commanders and returns its chronicle.
func Record(a *arena.Arena, c0, c1 arena.Commander, maxTicks int) Log {
	l := Log{W: a.W, H: a.H}
	for _, u := range a.Units {
		l.Roster = append(l.Roster, UnitInfo{
			Name:    arena.Bestiary[int(u.Kind)%len(arena.Bestiary)].Name,
			Faction: int(u.Faction),
		})
	}
	l.Events = append(l.Events, Event{Tick: 0, Kind: Open, Actor: a.AliveCount(0), Target: a.AliveCount(1)})

	for t := 1; t <= maxTicks; t++ {
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		n := len(a.Units)
		preHP := make([]int, n)
		preAlive := make([]bool, n)
		px := make([]int, n)
		py := make([]int, n)
		for i := range a.Units {
			preHP[i] = int(a.Units[i].HP)
			preAlive[i] = a.Units[i].Alive
			px[i] = int(a.Units[i].X)
			py[i] = int(a.Units[i].Y)
		}
		a.Step(c0, c1)
		for i := range a.Units {
			if preAlive[i] && int(a.Units[i].HP) < preHP[i] {
				l.Events = append(l.Events, Event{
					Tick: t, Kind: Strike,
					Actor: striker(a, i, px, py), Target: i,
					Amt: preHP[i] - int(a.Units[i].HP),
				})
			}
			if preAlive[i] && !a.Units[i].Alive {
				l.Events = append(l.Events, Event{Tick: t, Kind: Fall, Target: i})
			}
		}
	}
	a0, a1 := a.AliveCount(0), a.AliveCount(1)
	win := -1
	if a0 > a1 {
		win = 0
	} else if a1 > a0 {
		win = 1
	}
	l.Events = append(l.Events, Event{Tick: a.Tick, Kind: Close, Actor: a0, Target: a1, Faction: win})
	return l
}

// Bytes encodes the event stream compactly: 6 bytes per event (tick, kind,
// actor, target, amt, faction), clamped to a byte each — the whole war in a
// handful of bytes.
func (l Log) Bytes() []byte {
	b := make([]byte, 0, len(l.Events)*6)
	cl := func(v int) byte {
		if v < 0 {
			return 255
		}
		if v > 255 {
			return 255
		}
		return byte(v)
	}
	for _, e := range l.Events {
		b = append(b, cl(e.Tick), byte(e.Kind), cl(e.Actor), cl(e.Target), cl(e.Amt), cl(e.Faction))
	}
	return b
}

func host(f int) string {
	switch f {
	case 0:
		return "the blue"
	case 1:
		return "the red"
	default:
		return "an unknown"
	}
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Narrate retells the chronicle as prose.
func (l Log) Narrate() string {
	name := func(i int) string {
		if i < 0 || i >= len(l.Roster) {
			return "a shadow"
		}
		return host(l.Roster[i].Faction) + " " + l.Roster[i].Name
	}
	var b strings.Builder
	for _, e := range l.Events {
		switch e.Kind {
		case Open:
			fmt.Fprintf(&b, "Two hosts met upon the field — %d of the blue against %d of the red.\n", e.Actor, e.Target)
		case Strike:
			fmt.Fprintf(&b, "  t%d — %s struck %s for %d.\n", e.Tick, title(name(e.Actor)), name(e.Target), e.Amt)
		case Fall:
			fmt.Fprintf(&b, "  t%d — %s fell.\n", e.Tick, title(name(e.Target)))
		case Close:
			switch e.Faction {
			case 0:
				fmt.Fprintf(&b, "And so the blue host held the field, %d to %d.\n", e.Actor, e.Target)
			case 1:
				fmt.Fprintf(&b, "And so the red host held the field, %d to %d.\n", e.Target, e.Actor)
			default:
				fmt.Fprintf(&b, "And so neither host yielded, %d to %d.\n", e.Actor, e.Target)
			}
		}
	}
	return b.String()
}
