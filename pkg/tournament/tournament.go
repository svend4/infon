// Package tournament runs a round-robin league over pkg/session: any set of
// Participants — brains, bots, or human bridges — play a Game both home and away,
// and the package tallies standings (win 3, draw 1) and a match log. It is the
// "spectated AI tournament" the experimental notes envisioned, built on the
// unified session runner.
package tournament

import (
	"sort"

	"github.com/svend4/infon/pkg/session"
)

// MatchResult records one game. Winner: 0 = A (home), 1 = B (away), -1 = draw.
type MatchResult struct {
	A, B   string
	Winner int
	Turns  int
}

// Standing is a participant's record.
type Standing struct {
	Name                  string
	Wins, Losses, Draws   int
	Points                int
}

// Table is the league outcome: standings (sorted) and every match played.
type Table struct {
	Standings []Standing
	Matches   []MatchResult
}

// RoundRobin plays every pair both ways (to neutralise first-move advantage) and
// returns the standings and match log.
func RoundRobin[S any](g session.Game[S], players []session.Participant, maxTurns int) Table {
	n := len(players)
	st := make([]Standing, n)
	for i := range players {
		st[i].Name = players[i].Name()
	}
	var matches []MatchResult

	play := func(home, away int) {
		r, _ := session.Play[S](g, []session.Participant{players[home], players[away]}, maxTurns)
		matches = append(matches, MatchResult{A: st[home].Name, B: st[away].Name, Winner: r.Winner, Turns: r.Turns})
		switch r.Winner {
		case 0:
			st[home].Wins++
			st[home].Points += 3
			st[away].Losses++
		case 1:
			st[away].Wins++
			st[away].Points += 3
			st[home].Losses++
		default:
			st[home].Draws++
			st[away].Draws++
			st[home].Points++
			st[away].Points++
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			play(i, j)
			play(j, i)
		}
	}

	out := append([]Standing(nil), st...)
	sort.SliceStable(out, func(x, y int) bool {
		if out[x].Points != out[y].Points {
			return out[x].Points > out[y].Points
		}
		if out[x].Wins != out[y].Wins {
			return out[x].Wins > out[y].Wins
		}
		return out[x].Name < out[y].Name
	})
	return Table{Standings: out, Matches: matches}
}
