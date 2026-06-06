// Command debate stages a judged AI debate (pkg/debate): two debaters argue a
// topic over several rounds and a judge scores each one. Here the debaters and
// judge are scripted stand-ins (deterministic), but a live tvcp-ai brain plugs
// into the exact same Debater/Judge interfaces (see BrainDebater).
//
//	go run ./cmd/debate
package main

import (
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/debate"
)

func main() {
	pro := debate.FuncDebater{N: "PRO", F: func(_ string, r int, _ string) string {
		args := []string{
			"Bandwidth is scarce: a few hundred bytes of meaning beat megabytes of pixels.",
			"It is verifiable — the spec IS the image: diffable, signable, replayable.",
			"It survives packet loss and reaches the blind and the bandwidth-poor alike, as one artifact.",
		}
		return args[r%len(args)]
	}}
	con := debate.FuncDebater{N: "CON", F: func(_ string, r int, _ string) string {
		args := []string{
			"Pixels are universal; meaning needs a shared model to reconstruct.",
			"Symbolic vocabularies smuggle in a culture — whose 'sky', whose 'home'?",
			"Fidelity matters: a sketch is not a photograph.",
		}
		return args[r%len(args)]
	}}
	// a stand-in judge: the more elaborated argument (more words) wins the round.
	judge := debate.FuncJudge{N: "JUDGE", F: func(_, a, b string) int {
		switch wa, wb := len(strings.Fields(a)), len(strings.Fields(b)); {
		case wa > wb:
			return 0
		case wb > wa:
			return 1
		default:
			return -1
		}
	}}

	topic := "Should media stream MEANING, not pixels?"
	res := debate.Run(topic, 3, pro, con, judge)

	fmt.Printf("DEBATE: %q\n  %s vs %s, judged by %s\n\n", res.Topic, res.A, res.B, res.J)
	for i, rd := range res.Rounds {
		fmt.Printf("round %d\n  PRO: %s\n  CON: %s\n  -> %s\n\n", i+1, rd.ArgA, rd.ArgB, verdict(rd.Winner, res.A, res.B))
	}
	fmt.Printf("score: %s %d — %d %s (ties %d)\n", res.A, res.ScoreA, res.ScoreB, res.B, res.Ties)
	fmt.Printf("verdict: %s\n", winner(res))
}

func verdict(w int, a, b string) string {
	switch w {
	case 0:
		return a + " takes the round"
	case 1:
		return b + " takes the round"
	default:
		return "a draw"
	}
}

func winner(res debate.Result) string {
	switch res.Winner {
	case 0:
		return res.A + " wins the debate"
	case 1:
		return res.B + " wins the debate"
	default:
		return "the debate is a draw"
	}
}
