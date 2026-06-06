// Command unodemo runs two reference brains through a full game of UNO on the
// session runner (pkg/session) — the first game on the frame with HIDDEN
// INFORMATION (each player sees only its own hand) and CHANCE (a seed-shuffled
// deck). Both players speak the existing tvcp-ai game:uno protocol unchanged.
//
//	go run ./cmd/unodemo
package main

import (
	"fmt"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func main() {
	players := []session.Participant{
		session.BrainPlayer{N: "ref-A", B: brain.Local{}},
		session.BrainPlayer{N: "ref-B", B: brain.Local{}},
	}
	r, err := session.Play[session.UnoState](session.Uno{Seed: 7}, players, 3000)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("uno — two reference brains, each seeing only its own hand")
	fmt.Printf("  deck shuffled from a seed; %d turns played\n", r.Turns)
	top := r.Final.Discard[len(r.Final.Discard)-1]
	fmt.Printf("  final discard top: %s (active color %s)\n", top, top.Color)
	fmt.Printf("  final hand sizes: ref-A=%d, ref-B=%d\n", len(r.Final.Hands[0]), len(r.Final.Hands[1]))
	switch r.Winner {
	case 0:
		fmt.Println("  winner: ref-A (emptied its hand)")
	case 1:
		fmt.Println("  winner: ref-B (emptied its hand)")
	default:
		fmt.Println("  no winner within the turn budget")
	}
	fmt.Println("hidden information and chance on the same runner as tic-tac-toe, Connect Four and Wordle.")
}
