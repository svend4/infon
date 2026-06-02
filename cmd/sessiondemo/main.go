// Command sessiondemo shows the unified game session (pkg/session): the same
// runner plays one Game (tic-tac-toe) between any Participants — here a tvcp-ai
// brain against another brain, then a brain against a scripted bot. Human, bot
// and brain are interchangeable.
//
//	go run ./cmd/sessiondemo
package main

import (
	"encoding/json"
	"fmt"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func firstLegal(req brain.Request) brain.Response {
	var st brain.TicTacToeState
	_ = json.Unmarshal(req.State, &st)
	if len(st.Legal) > 0 {
		return brain.Response{Kind: "move", Move: &brain.Move{Row: st.Legal[0][0], Col: st.Legal[0][1]}}
	}
	return brain.Response{Kind: "move", Move: &brain.Move{}}
}

func who(w int) string {
	switch w {
	case 0:
		return "X"
	case 1:
		return "O"
	default:
		return "draw"
	}
}

func printBoard(b session.Board) {
	for i := 0; i < 3; i++ {
		row := ""
		for j := 0; j < 3; j++ {
			c := b[i][j]
			if c == "" {
				c = "."
			}
			row += " " + c
		}
		fmt.Println("   " + row)
	}
}

func run(label string, players []session.Participant) {
	r, err := session.Play[session.Board](session.TicTacToe{}, players, 12)
	if err != nil {
		fmt.Printf("%s: error: %v\n", label, err)
		return
	}
	fmt.Printf("%s:\n", label)
	printBoard(r.Final)
	fmt.Printf("   winner=%s turns=%d\n\n", who(r.Winner), r.Turns)
}

func main() {
	run("brain vs brain", []session.Participant{
		session.BrainPlayer{N: "ref-X", B: brain.Local{}},
		session.BrainPlayer{N: "ref-O", B: brain.Local{}},
	})
	run("optimal brain (X) vs first-legal bot (O)", []session.Participant{
		session.BrainPlayer{N: "ref-X", B: brain.Local{}},
		session.FuncPlayer{N: "bot-O", F: firstLegal},
	})
	fmt.Println("any Participant — brain, bot, or a human bridge — plays any Game through one session.")
}
