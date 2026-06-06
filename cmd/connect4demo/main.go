// Command connect4demo shows a SECOND game on the same session runner
// (pkg/session): Connect Four, a win/block tactician bot against a naive leftmost
// bot. It proves the Game/Participant frame is game-agnostic — the same runner,
// tournament and debate machinery now work for connect4 too.
//
//	go run ./cmd/connect4demo
package main

import (
	"encoding/json"
	"fmt"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func mv(c int) brain.Response { return brain.Response{Kind: "move", Move: &brain.Move{Col: c}} }

func parse(req brain.Request) session.CFState {
	var st session.CFState
	_ = json.Unmarshal(req.State, &st)
	return st
}

func naive(req brain.Request) brain.Response {
	st := parse(req)
	if len(st.Legal) > 0 {
		return mv(st.Legal[0])
	}
	return mv(0)
}

func tactician(req brain.Request) brain.Response {
	st := parse(req)
	me := st.You
	opp := "X"
	if me == "X" {
		opp = "O"
	}
	// win if you can
	for _, c := range st.Legal {
		if nb, ok := session.CFPlace(st.Board, c, me); ok && session.CFWinner(nb) == me {
			return mv(c)
		}
	}
	// else block an immediate threat
	for _, c := range st.Legal {
		if nb, ok := session.CFPlace(st.Board, c, opp); ok && session.CFWinner(nb) == opp {
			return mv(c)
		}
	}
	// else prefer the center
	for _, c := range []int{3, 2, 4, 1, 5, 0, 6} {
		for _, l := range st.Legal {
			if l == c {
				return mv(c)
			}
		}
	}
	if len(st.Legal) > 0 {
		return mv(st.Legal[0])
	}
	return mv(0)
}

func printBoard(b session.CFBoard) {
	for r := 5; r >= 0; r-- {
		row := ""
		for c := 0; c < 7; c++ {
			x := b[r][c]
			if x == "" {
				x = "."
			}
			row += " " + x
		}
		fmt.Println("   " + row)
	}
	fmt.Println("    0 1 2 3 4 5 6")
}

func main() {
	players := []session.Participant{
		session.FuncPlayer{N: "tactician", F: tactician},
		session.FuncPlayer{N: "naive", F: naive},
	}
	r, err := session.Play[session.CFBoard](session.ConnectFour{}, players, 60)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("connect four — tactician (X) vs naive-leftmost (O):")
	printBoard(r.Final)
	who := "draw"
	switch r.Winner {
	case 0:
		who = "tactician (X)"
	case 1:
		who = "naive (O)"
	}
	fmt.Printf("   winner: %s · turns: %d\n", who, r.Turns)
	fmt.Println("a second game on the same runner — bots, brains and humans interchangeable.")
}
