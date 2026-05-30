// Command aiplayf is a LIVE tic-tac-toe driven over a shared folder, so an
// external brain (Claude) can play move-by-move while the repo engine + renderer
// run for real. X = the brain (reads assets/ttt/mvK.txt); O = repo engine
// (random, winnable). Boards go to assets/ttt/board.txt; colored frames are
// logged to assets/aiplayf.frames.txt for a GIF.
//
//	go run -tags experimental ./cmd/aiplayf
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	f.DrawText(1, 0, "Claude (X)  vs  random (O)  - LIVE", color.NewRGB(235, 235, 245), bg)
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
	g.SetupVsAI("claude", "Claude", 0) // O = random (winnable)
	_ = g.Start()
	dir := "assets/ttt"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	ff, _ := os.Create("assets/aiplayf.frames.txt")
	defer ff.Close()
	moveNo := 0
	emit := func() { ff.WriteString(renderBoard(g, moveNo).Render()); ff.WriteString("\n@@@FRAME@@@\n"); _ = ff.Sync() }
	writeBoard := func(extra string) { _ = os.WriteFile(dir+"/board.txt", []byte(plain(g)+extra), 0o644) }
	emit()
	k := 0
	deadline := time.Now().Add(240 * time.Second)
	for {
		if g.State != board.StatePlaying {
			writeBoard("RESULT: " + g.State.String() + "\nDONE\n")
			_ = os.WriteFile(dir+"/done.txt", []byte(g.State.String()), 0o644)
			break
		}
		if g.Turn == 0 {
			writeBoard(fmt.Sprintf("AWAIT k=%d\n", k))
			mv := fmt.Sprintf("%s/mv%d.txt", dir, k)
			applied := false
			for !applied {
				if time.Now().After(deadline) {
					_ = os.WriteFile(dir+"/done.txt", []byte("timeout"), 0o644)
					return
				}
				if data, err := os.ReadFile(mv); err == nil {
					parts := strings.Fields(strings.TrimSpace(string(data)))
					if len(parts) >= 2 {
						r, e1 := strconv.Atoi(parts[0])
						c, e2 := strconv.Atoi(parts[1])
						if e1 == nil && e2 == nil && g.MakeMove(r, c) == nil {
							applied = true
							break
						}
					}
					_ = os.WriteFile(dir+"/err.txt", []byte("invalid; rewrite "+mv), 0o644)
					_ = os.Remove(mv)
				}
				time.Sleep(150 * time.Millisecond)
			}
			k++
			moveNo++
			emit()
		} else {
			r, c, err := g.GetAIMove()
			if err != nil {
				break
			}
			_ = g.MakeMove(r, c)
			_ = os.WriteFile(dir+"/olast.txt", []byte(fmt.Sprintf("%d %d", r, c)), 0o644)
			moveNo++
			emit()
		}
	}
}
