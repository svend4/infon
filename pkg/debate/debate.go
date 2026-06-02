// Package debate stages a judged AI debate: two Debaters argue a topic over
// several rounds (each seeing the opponent's last point), and a Judge scores each
// round; the runner returns a transcript and a verdict. Debaters and the judge
// are interfaces, so a live tvcp-ai brain, a scripted bot, or a human bridge all
// slot in — the spectated-AI-debate the experimental notes imagined, on the same
// open protocol.
package debate

import (
	"strings"

	"github.com/svend4/infon/pkg/brain"
)

// Debater makes an argument given the topic, the round, and the opponent's last
// point.
type Debater interface {
	Name() string
	Argue(topic string, round int, opponentLast string) string
}

// Judge scores a round: 0 = A wins, 1 = B wins, -1 = tie.
type Judge interface {
	Name() string
	Score(topic, argA, argB string) int
}

// Round is one exchange and its verdict.
type Round struct {
	ArgA, ArgB string
	Winner     int
}

// Result is the debate outcome.
type Result struct {
	Topic                string
	A, B, J              string
	Rounds               []Round
	ScoreA, ScoreB, Ties int
	Winner               int // 0 = A, 1 = B, -1 = tie
}

// Run plays the debate.
func Run(topic string, rounds int, a, b Debater, j Judge) Result {
	res := Result{Topic: topic, A: a.Name(), B: b.Name(), J: j.Name()}
	var lastA, lastB string
	for r := 0; r < rounds; r++ {
		aa := a.Argue(topic, r, lastB)
		bb := b.Argue(topic, r, lastA)
		w := j.Score(topic, aa, bb)
		res.Rounds = append(res.Rounds, Round{ArgA: aa, ArgB: bb, Winner: w})
		switch w {
		case 0:
			res.ScoreA++
		case 1:
			res.ScoreB++
		default:
			res.Ties++
		}
		lastA, lastB = aa, bb
	}
	switch {
	case res.ScoreA > res.ScoreB:
		res.Winner = 0
	case res.ScoreB > res.ScoreA:
		res.Winner = 1
	default:
		res.Winner = -1
	}
	return res
}

// FuncDebater adapts a function into a Debater (a scripted bot).
type FuncDebater struct {
	N string
	F func(topic string, round int, opponentLast string) string
}

func (d FuncDebater) Name() string { return d.N }
func (d FuncDebater) Argue(topic string, round int, opp string) string {
	return d.F(topic, round, opp)
}

// FuncJudge adapts a function into a Judge.
type FuncJudge struct {
	N string
	F func(topic, argA, argB string) int
}

func (j FuncJudge) Name() string                       { return j.N }
func (j FuncJudge) Score(topic, a, b string) int        { return j.F(topic, a, b) }

// BrainDebater wraps any tvcp-ai brain as a Debater: it asks the brain to "react"
// to the topic and the opponent's last point, and uses the reply as its argument.
type BrainDebater struct {
	N string
	B brain.Brain
}

func (d BrainDebater) Name() string { return d.N }
func (d BrainDebater) Argue(topic string, round int, opp string) string {
	prompt := topic
	if opp != "" {
		prompt += " || rebut: " + opp
	}
	resp, err := d.B.Decide(brain.Request{Protocol: brain.Protocol, Kind: "react", Prompt: prompt})
	if err != nil {
		return "(no argument)"
	}
	s := strings.TrimSpace(resp.Reasoning)
	if len(resp.Cards) > 0 {
		s = strings.TrimSpace(s + " [" + strings.Join(resp.Cards, " ") + "]")
	}
	if s == "" {
		s = "(no argument)"
	}
	return s
}
