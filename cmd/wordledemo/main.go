// Command wordledemo shows the reference brain playing WORDLE on the session
// runner (pkg/session) — the first SOLITAIRE game on the same frame that runs
// tic-tac-toe and Connect Four. One player, a hidden word, per-letter feedback;
// the brain speaks the existing tvcp-ai game:wordle protocol unchanged.
//
//	go run ./cmd/wordledemo
package main

import (
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func annotate(marks []string) string {
	b := strings.Builder{}
	for _, m := range marks {
		switch m {
		case "correct":
			b.WriteByte('^') // right letter, right place
		case "present":
			b.WriteByte('+') // right letter, wrong place
		default:
			b.WriteByte('.') // not in the word
		}
	}
	return b.String()
}

func main() {
	target := "CRANE"
	g := session.Wordle{Target: target, MaxGuesses: 6}
	r, err := session.Play[session.WordleState](
		g, []session.Participant{session.BrainPlayer{N: "reference", B: brain.Local{}}}, 20)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("wordle — reference brain vs a hidden %d-letter word\n\n", len(target))
	for i, gx := range r.Final.Guesses {
		fmt.Printf("  %d  %s\n     %s\n", i+1, gx.Word, annotate(gx.Marks))
	}
	fmt.Println()
	if r.Winner == 0 {
		fmt.Printf("solved %q in %d guesses.\n", target, r.Turns)
	} else {
		fmt.Printf("not solved in %d guesses (answer was %q).\n", r.Turns, target)
	}
	fmt.Println("one player, hidden state, feedback — the same runner as tic-tac-toe and Connect Four.")
}
