// Command tournament runs a spectated tic-tac-toe LEAGUE (pkg/tournament over
// pkg/session): several participants — a minimax brain and a few scripted bots —
// play round-robin, home and away, and the standings print out. Any brain, bot or
// human bridge could enter the same way.
//
//	go run ./cmd/tournament
package main

import (
	"encoding/json"
	"fmt"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
	"github.com/svend4/infon/pkg/tournament"
)

func pick(req brain.Request, order [][2]int) brain.Response {
	var st brain.TicTacToeState
	_ = json.Unmarshal(req.State, &st)
	legal := map[[2]int]bool{}
	for _, c := range st.Legal {
		legal[[2]int{c[0], c[1]}] = true
	}
	for _, c := range order {
		if legal[c] {
			return brain.Response{Kind: "move", Move: &brain.Move{Row: c[0], Col: c[1]}}
		}
	}
	if len(st.Legal) > 0 {
		return brain.Response{Kind: "move", Move: &brain.Move{Row: st.Legal[0][0], Col: st.Legal[0][1]}}
	}
	return brain.Response{Kind: "move", Move: &brain.Move{}}
}

var (
	rowMajor = [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}}
	center   = [][2]int{{1, 1}, {0, 0}, {0, 2}, {2, 0}, {2, 2}, {0, 1}, {1, 0}, {1, 2}, {2, 1}}
)

func main() {
	rev := make([][2]int, len(rowMajor))
	for i := range rowMajor {
		rev[i] = rowMajor[len(rowMajor)-1-i]
	}
	players := []session.Participant{
		session.BrainPlayer{N: "optimal", B: brain.Local{}},
		session.FuncPlayer{N: "center", F: func(r brain.Request) brain.Response { return pick(r, center) }},
		session.FuncPlayer{N: "first", F: func(r brain.Request) brain.Response { return pick(r, rowMajor) }},
		session.FuncPlayer{N: "last", F: func(r brain.Request) brain.Response { return pick(r, rev) }},
	}

	tbl := tournament.RoundRobin[session.Board](session.TicTacToe{}, players, 12)

	fmt.Printf("tic-tac-toe league — %d players, %d games (home & away)\n\n", len(players), len(tbl.Matches))
	fmt.Printf("%-10s %3s %3s %3s %4s\n", "player", "W", "L", "D", "pts")
	for _, s := range tbl.Standings {
		fmt.Printf("%-10s %3d %3d %3d %4d\n", s.Name, s.Wins, s.Losses, s.Draws, s.Points)
	}
	fmt.Printf("\nchampion: %s\n", tbl.Standings[0].Name)
}
