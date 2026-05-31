//go:build experimental

// Command aiplay connects an external "brain" to the repo's tic-tac-toe engine
// (experimental/games/board) and renders every board as a colored Frame.
//
//	X moves: supplied by the LLM (Claude)      O moves: repo minimax (GetAIMove)
//	engine: board.TicTacToe   renderer: pkg/terminal.Frame
//
// Build (note the experimental tag) and run:
//
//	go run -tags experimental ./cmd/aiplay
package main

import (
	"fmt"
	"os"

	"github.com/svend4/infon/experimental/games/board"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// xMovesは the LLM's chosen line for X (a draw vs perfect play).
var xMoves = [][2]int{{1, 1}, {0, 2}, {1, 0}, {0, 1}, {2, 2}}

func renderBoard(g *board.TicTacToe, moveNo int) *terminal.Frame {
	const W, H = 31, 17
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(14, 16, 28)
	f.Fill(' ', color.NewRGB(220, 220, 230), bg)
	red := color.NewRGB(235, 70, 70)
	cyan := color.NewRGB(70, 205, 235)
	grid := color.NewRGB(90, 95, 120)
	white := color.NewRGB(235, 235, 245)

	f.DrawText(1, 0, "Claude (X)  vs  minimax (O)", white, bg)
	f.DrawText(1, 1, fmt.Sprintf("move %d  -  %s", moveNo, g.State.String()), color.NewRGB(150, 160, 190), bg)

	ox, oy, cw, chh := 2, 3, 9, 4
	for r := 0; r <= 3; r++ {
		yy := oy + r*chh
		for x := 0; x <= 3*cw; x++ {
			f.SetBlock(ox+x, yy, '─', grid, bg)
		}
	}
	for c := 0; c <= 3; c++ {
		xx := ox + c*cw
		for y := 0; y <= 3*chh; y++ {
			f.SetBlock(xx, oy+y, '│', grid, bg)
		}
	}
	for r := 0; r <= 3; r++ {
		for c := 0; c <= 3; c++ {
			f.SetBlock(ox+c*cw, oy+r*chh, '┼', grid, bg)
		}
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			cx := ox + c*cw + cw/2
			cy := oy + r*chh + chh/2
			switch g.Board[r][c] {
			case board.MarkX:
				f.SetBlock(cx, cy, 'X', red, bg)
			case board.MarkO:
				f.SetBlock(cx, cy, 'O', cyan, bg)
			}
		}
	}
	return f
}

func main() {
	g := board.NewTicTacToe()
	g.SetupVsAI("claude", "Claude", 2) // X = Claude (scripted), O = minimax (Hard)
	if err := g.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	emit := func(n int) { fmt.Print(renderBoard(g, n).Render()); fmt.Print("\n@@@FRAME@@@\n") }
	emit(0)
	xi, moveNo := 0, 0
	for g.State == board.StatePlaying {
		moveNo++
		if g.Turn == 0 {
			m := xMoves[xi]
			xi++
			if err := g.MakeMove(m[0], m[1]); err != nil {
				fmt.Fprintln(os.Stderr, "x move err:", err)
				return
			}
		} else {
			r, c, err := g.GetAIMove()
			if err != nil {
				fmt.Fprintln(os.Stderr, "ai err:", err)
				return
			}
			g.MakeMove(r, c)
		}
		emit(moveNo)
	}
	fmt.Fprintln(os.Stderr, "final:", g.State.String())
}
