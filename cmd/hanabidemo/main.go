// Command hanabidemo plays a cooperative game of Hanabi on the session runner and
// narrates it. Hanabi is the project's thesis as a game: two players see every
// hand BUT their own, and the ONLY way knowledge crosses between them is a single
// colour-or-number clue. Watch two brains build the fireworks by speaking through
// that ~30-byte channel.
//
//	go run ./cmd/hanabidemo                 # two reference brains cooperate
//	go run ./cmd/hanabidemo -seed 12        # a different deal
//	go run ./cmd/hanabidemo -brain0 http://127.0.0.1:8092/v1/decide   # a live LLM on side 0
//
// With -brain0/-brain1 either partner becomes any tvcp-ai/1 brain (Haiku, OpenAI,
// a local Ollama model) that implements game:hanabi.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

var colors = []string{"R", "G", "B", "Y", "W"}

func mkBrain(url string) brain.Brain {
	if url == "" {
		return brain.Local{}
	}
	return brain.HTTPBrain{URL: url}
}

// fireworks renders the five stacks as rows of pips, e.g. R ●●●·· .
func fireworks(s session.HanabiState) string {
	var b strings.Builder
	for _, col := range colors {
		v := s.Stacks[col]
		b.WriteString(fmt.Sprintf("%s %s  ", col, strings.Repeat("●", v)+strings.Repeat("·", 5-v)))
	}
	return strings.TrimRight(b.String(), " ")
}

type move struct {
	Action string `json:"action"`
	Card   int    `json:"card"`
	To     int    `json:"to"`
	Kind   string `json:"kind"`
	Color  string `json:"color"`
	Num    int    `json:"num"`
}

// narrate describes a move from the omniscient view (before it is applied).
func narrate(s session.HanabiState, p int, m move) string {
	switch m.Action {
	case "play":
		c := s.Hands[p][m.Card]
		ok := s.Stacks[c.Color]+1 == c.Num
		mark := "✓"
		tail := fmt.Sprintf("%s→%d", c.Color, c.Num)
		if !ok {
			mark = "✗ MISPLAY — fuse!"
			tail = "wasted"
		}
		return fmt.Sprintf("P%d plays slot %d = %s  %s  (%s)", p, m.Card, c.Code(), mark, tail)
	case "discard":
		c := s.Hands[p][m.Card]
		return fmt.Sprintf("P%d discards slot %d = %s  (clue token recovered)", p, m.Card, c.Code())
	case "hint":
		var slots []int
		for i, c := range s.Hands[m.To] {
			if (m.Kind == "color" && c.Color == m.Color) || (m.Kind == "number" && c.Num == m.Num) {
				slots = append(slots, i)
			}
		}
		what := m.Color
		if m.Kind == "number" {
			what = fmt.Sprintf("number %d", m.Num)
		} else {
			what = "colour " + m.Color
		}
		return fmt.Sprintf("P%d ⇒ P%d : %-9s — touches slots %v", p, m.To, what, slots)
	}
	return "?"
}

func rating(score int) string {
	switch {
	case score >= 25:
		return "legendary"
	case score >= 20:
		return "excellent"
	case score >= 15:
		return "honourable"
	case score >= 10:
		return "solid"
	case score >= 5:
		return "mediocre"
	default:
		return "poor"
	}
}

func main() {
	seed := flag.Int64("seed", 7, "deal seed")
	brain0 := flag.String("brain0", "", "tvcp-ai/1 URL for player 0 (default: reference brain)")
	brain1 := flag.String("brain1", "", "tvcp-ai/1 URL for player 1 (default: reference brain)")
	maxTurns := flag.Int("turns", 400, "safety cap on turns")
	flag.Parse()

	g := session.Hanabi{Seed: *seed}
	players := []brain.Brain{mkBrain(*brain0), mkBrain(*brain1)}
	name := func(i int) string {
		if (i == 0 && *brain0 != "") || (i == 1 && *brain1 != "") {
			return "live"
		}
		return "ref"
	}

	fmt.Println("Hanabi — cooperative fireworks over a clue-only channel (tvcp-ai/1 game:hanabi)")
	fmt.Printf("Player 0: %s    Player 1: %s    seed %d\n", name(0), name(1), *seed)
	fmt.Println("Each player sees the OTHER's hand, never their own; clues are the only bridge.")
	fmt.Println(strings.Repeat("─", 72))

	s := g.Start()
	var plays, discards, hints, bytes int
	for step := 0; step < *maxTurns; step++ {
		if done, _ := g.Over(s); done {
			break
		}
		p := g.Turn(s)
		if p < 0 {
			break
		}
		resp, err := players[p].Decide(g.Brief(s, p))
		if err != nil {
			fmt.Printf("player %d transport error: %v\n", p, err)
			return
		}
		mb, err := g.Parse(resp)
		if err != nil {
			fmt.Printf("player %d produced an illegal move: %v\n", p, err)
			return
		}
		bytes += len(mb)
		var m move
		_ = json.Unmarshal(mb, &m)
		switch m.Action {
		case "play":
			plays++
		case "discard":
			discards++
		case "hint":
			hints++
		}
		line := narrate(s, p, m)
		ns, err := g.Apply(s, p, mb)
		if err != nil {
			fmt.Printf("player %d illegal move: %v\n", p, err)
			return
		}
		s = ns
		fmt.Printf("%2d. %-52s | score %2d  clues %d  fuses %d\n", step+1, line, session.HanabiScore(s), s.Hints, s.Fuses)
	}

	fmt.Println(strings.Repeat("─", 72))
	fmt.Println("Final fireworks:")
	fmt.Println("  " + fireworks(s))
	score := session.HanabiScore(s)
	moves := plays + discards + hints
	avg := 0.0
	if moves > 0 {
		avg = float64(bytes) / float64(moves)
	}
	fmt.Printf("\nScore %d/25 (%s).  Fuses left %d/3.\n", score, rating(score), s.Fuses)
	fmt.Printf("The whole conversation was %d moves / %d bytes — %.1f bytes per turn, the entire signal between two minds.\n", moves, bytes, avg)
	fmt.Printf("Breakdown: %d plays, %d clues, %d discards.\n", plays, hints, discards)
}
