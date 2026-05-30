// Command aiplayi is an INTERACTIVE tic-tac-toe: X reads moves from stdin (so a
// live LLM can play through the terminal via interact_with_process), O is the
// repo engine (random here for a winnable game). Each board is also logged as a
// colored Frame to assets/aiplayi.frames.txt for a GIF.
//
//	go run -tags experimental ./cmd/aiplayi
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/svend4/infon/experimental/games/board"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func renderBoard(g *board.TicTacToe, moveNo int) *terminal.Frame {
	const W, H = 31, 17
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(14, 16, 28)
	f.Fill(' ', color.NewRGB(220, 220, 230), bg)
	red := color.NewRGB(235, 70, 70)
	cyan := color.NewRGB(70, 205, 235)
	grid := color.NewRGB(90, 95, 120)
	f.DrawText(1, 0, "Claude (X)  vs  random (O)", color.NewRGB(235, 235, 245), bg)
	f.DrawText(1, 1, fmt.Sprintf("move %d  -  %s", moveNo, g.State.String()), color.NewRGB(150, 160, 190), bg)
	ox, oy, cw, chh := 2, 3, 9, 4
	for r := 0; r <= 3; r++ {
		for x := 0; x <= 3*cw; x++ {
			f.SetBlock(ox+x, oy+r*chh, '─', grid, bg)
		}
	}
	for c := 0; c <= 3; c++ {
		for y := 0; y <= 3*chh; y++ {
			f.SetBlock(ox+c*cw, oy+y, '│', grid, bg)
		}
	}
	for r := 0; r <= 3; r++ {
		for c := 0; c <= 3; c++ {
			f.SetBlock(ox+c*cw, oy+r*chh, '┼', grid, bg)
		}
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			cx, cy := ox+c*cw+cw/2, oy+r*chh+chh/2
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

func plain(g *board.TicTacToe) string {
	var b strings.Builder
	b.WriteString("===BOARD===\n   c0 c1 c2\n")
	for r := 0; r < 3; r++ {
		b.WriteString("r" + strconv.Itoa(r))
		for c := 0; c < 3; c++ {
			ch := "."
			switch g.Board[r][c] {
			case board.MarkX:
				ch = "X"
			case board.MarkO:
				ch = "O"
			}
			b.WriteString("  " + ch)
		}
		b.WriteString("\n")
	}
	turn := "X"
	if g.Turn == 1 {
		turn = "O"
	}
	b.WriteString(fmt.Sprintf("STATE: %s  TURN: %s\n", g.State.String(), turn))
	return b.String()
}

func main() {
	g := board.NewTicTacToe()
	g.SetupVsAI("claude", "Claude", 0) // O = random
	_ = g.Start()
	f, _ := os.Create("assets/aiplayi.frames.txt")
	defer f.Close()
	moveNo := 0
	emit := func() { f.WriteString(renderBoard(g, moveNo).Render()); f.WriteString("\n@@@FRAME@@@\n"); _ = f.Sync() }
	emit()
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(plain(g))
		if g.State != board.StatePlaying {
			fmt.Printf("RESULT: %s\n<<<END>>>\n", g.State.String())
			break
		}
		if g.Turn == 0 {
			fmt.Println("AWAIT_X")
			if !sc.Scan() {
				break
			}
			parts := strings.Fields(strings.TrimSpace(sc.Text()))
			if len(parts) < 2 {
				fmt.Println("ERR need: row col")
				continue
			}
			r, e1 := strconv.Atoi(parts[0])
			c, e2 := strconv.Atoi(parts[1])
			if e1 != nil || e2 != nil {
				fmt.Println("ERR not ints")
				continue
			}
			if err := g.MakeMove(r, c); err != nil {
				fmt.Println("ERR", err)
				continue
			}
			moveNo++
			emit()
		} else {
			r, c, err := g.GetAIMove()
			if err != nil {
				fmt.Println("ERR ai", err)
				break
			}
			_ = g.MakeMove(r, c)
			fmt.Printf("O_PLAYED: %d %d\n", r, c)
			moveNo++
			emit()
		}
	}
}
