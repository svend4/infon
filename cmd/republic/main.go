// Command republic runs the self-organizing agent-world: brains advertise in a
// registry, an ACL Contract-Net round awards the two faction commands to the
// lowest bidders, and the arena is fought out with those awarded brains as
// commanders (game:rpg). Discover -> negotiate -> fight, with nobody hard-coding
// who leads which side.
//
//	go run ./cmd/republic
//	go run ./cmd/republic -seed 12 -json
//	go run ./cmd/republic -brain0 http://127.0.0.1:8092/v1/decide   # a live brain bids for (and wins) a seat
//
// With -brain0/-brain1 a live tvcp-ai brain (Haiku, OpenAI, a local Ollama model)
// enters the auction at a winning bid and commands its faction.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/republic"
)

func fixed(n int) func(int) int { return func(int) int { return n } }

func main() {
	seed := flag.Int64("seed", 7, "battlefield seed")
	brain0 := flag.String("brain0", "", "tvcp-ai/1 URL of a live brain to enter the auction (bids to win)")
	brain1 := flag.String("brain1", "", "a second live brain URL")
	asJSON := flag.Bool("json", false, "emit the Result as JSON")
	turns := flag.Int("turns", 200, "cap the battle length in ticks (keeps a live duel bounded)")
	flag.Parse()

	// The standing candidates: reference brains with distinct bids. Live brains,
	// if supplied, enter with the lowest bids and so win their seats.
	cands := []republic.Candidate{
		{Name: "scout", Brain: brain.Local{}, Cost: fixed(3)},
		{Name: "planner", Brain: brain.Local{}, Cost: fixed(4)},
		{Name: "warden", Brain: brain.Local{}, Cost: fixed(5)},
		{Name: "ranger", Brain: brain.Local{}, Cost: fixed(6)},
	}
	if *brain0 != "" {
		cands = append(cands, republic.Candidate{Name: "live-haiku", Brain: brain.HTTPBrain{URL: *brain0}, Cost: fixed(1)})
	}
	if *brain1 != "" {
		cands = append(cands, republic.Candidate{Name: "live-openai", Brain: brain.HTTPBrain{URL: *brain1}, Cost: fixed(2)})
	}

	res := republic.ConveneN(cands, *seed, *turns)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}

	fmt.Println("TVCP republic — discover ▸ negotiate ▸ fight")
	fmt.Println("registered (game:rpg): " + strings.Join(res.Registered, ", "))
	fmt.Println("\nnegotiation (Contract Net — the lowest bid wins each seat):")
	for _, s := range res.Seats {
		mark := "✓"
		who := s.Brain
		if !s.Awarded {
			mark, who = "✗", "(no award)"
		}
		fmt.Printf("  faction %d ⇐ %-12s (bid %d)  %s\n", s.Faction, who, s.Cost, mark)
	}

	verdict := "draw"
	switch res.Winner {
	case 0:
		verdict = "faction 0 holds the field"
	case 1:
		verdict = "faction 1 holds the field"
	}
	fmt.Printf("\nbattle: %d ticks → %s  (%d vs %d alive)\n", res.Ticks, verdict, res.Alive0, res.Alive1)
	fmt.Println("\nfinal field (B = faction 0, R = faction 1):")
	fmt.Print(field(res.Final))
}

func field(a *arena.Arena) string {
	if a == nil {
		return ""
	}
	cells := make([][]rune, a.H)
	for y := range cells {
		cells[y] = make([]rune, a.W)
		for x := range cells[y] {
			cells[y][x] = '·'
		}
	}
	for _, u := range a.Units {
		if !u.Alive || int(u.Y) >= a.H || int(u.X) >= a.W {
			continue
		}
		r := 'B'
		if u.Faction == 1 {
			r = 'R'
		}
		cells[u.Y][u.X] = r
	}
	var b strings.Builder
	for _, row := range cells {
		b.WriteString("  ")
		b.WriteString(string(row))
		b.WriteByte('\n')
	}
	return b.String()
}
