//go:build experimental

// Command aiturn plays ONE turn per run against the repo engine, persisting the
// game in a shared folder so an external brain (you, or another model) drives X
// move-by-move. O is the repo random by default, or — with -brain URL — any
// tvcp-ai/1 brain (e.g. Haiku), so you can play AGAINST a model in the terminal.
//
//	aiturn 1 1                                   # O random
//	aiturn -brain http://127.0.0.1:8092/v1/decide 1 1   # O = the brain
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/svend4/infon/experimental/games/board"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func renderBoard(g *board.TicTacToe, moveNo int, who string) *terminal.Frame {
	const W, H = 33, 17
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(14, 16, 28)
	f.Fill(' ', color.NewRGB(220, 220, 230), bg)
	red := color.NewRGB(235, 70, 70)
	cyan := color.NewRGB(70, 205, 235)
	grid := color.NewRGB(90, 95, 120)
	f.DrawText(1, 0, "you (X)  vs  "+who+" (O)  - LIVE", color.NewRGB(235, 235, 245), bg)
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

func newGame() *board.TicTacToe {
	g := board.NewTicTacToe()
	g.SetupVsAI("you", "You", 0)
	_ = g.Start()
	return g
}

func boardStr(g *board.TicTacToe) [3][3]string {
	var b [3][3]string
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			switch g.Board[r][c] {
			case board.MarkX:
				b[r][c] = "X"
			case board.MarkO:
				b[r][c] = "O"
			}
		}
	}
	return b
}

func emptiesB(g *board.TicTacToe) [][2]int {
	var e [][2]int
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if g.Board[r][c] == board.MarkEmpty {
				e = append(e, [2]int{r, c})
			}
		}
	}
	return e
}

// oMove returns O's move from the brain (validated) or the engine fallback.
func oMove(g *board.TicTacToe, url string) (int, int, bool) {
	if url != "" {
		hb := brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}
		raw, _ := json.Marshal(brain.TicTacToeState{Board: boardStr(g), Turn: "O", You: "O", Legal: emptiesB(g)})
		if resp, err := hb.Decide(brain.Request{Kind: "move", Game: "tictactoe", State: raw}); err == nil && resp.Move != nil {
			r, c := resp.Move.Row, resp.Move.Col
			if r >= 0 && r < 3 && c >= 0 && c < 3 && g.Board[r][c] == board.MarkEmpty {
				return r, c, true
			}
		}
	}
	if r, c, err := g.GetAIMove(); err == nil {
		return r, c, true
	}
	return 0, 0, false
}

func main() {
	brainURL := flag.String("brain", "", "O is played by this tvcp-ai/1 brain (default: random)")
	flag.Parse()
	args := flag.Args()
	who := "random"
	if *brainURL != "" {
		who = "brain"
	}

	dir := "assets/ttt2"
	_ = os.MkdirAll(dir, 0o755)
	histPath := dir + "/history.txt"
	var hist [][2]int
	if data, err := os.ReadFile(histPath); err == nil {
		for _, ln := range strings.Split(string(data), "\n") {
			fs := strings.Fields(strings.TrimSpace(ln))
			if len(fs) >= 2 {
				r, _ := strconv.Atoi(fs[0])
				c, _ := strconv.Atoi(fs[1])
				hist = append(hist, [2]int{r, c})
			}
		}
	}
	g := newGame()
	for _, m := range hist {
		_ = g.MakeMove(m[0], m[1])
	}
	msg := ""
	if len(args) >= 2 && g.State == board.StatePlaying && g.Turn == 0 {
		r, e1 := strconv.Atoi(args[0])
		c, e2 := strconv.Atoi(args[1])
		if e1 == nil && e2 == nil && g.MakeMove(r, c) == nil {
			hist = append(hist, [2]int{r, c})
			if g.State == board.StatePlaying && g.Turn == 1 {
				if or, oc, ok := oMove(g, *brainURL); ok {
					_ = g.MakeMove(or, oc)
					hist = append(hist, [2]int{or, oc})
					msg = fmt.Sprintf("O_PLAYED: %d %d\n", or, oc)
				}
			}
		} else {
			msg = "ERR invalid move\n"
		}
	}
	var sb strings.Builder
	for _, m := range hist {
		sb.WriteString(fmt.Sprintf("%d %d\n", m[0], m[1]))
	}
	_ = os.WriteFile(histPath, []byte(sb.String()), 0o644)

	g2 := newGame()
	ff, _ := os.Create("assets/aiturn.frames.txt")
	mn := 0
	emit := func(gg *board.TicTacToe) {
		ff.WriteString(renderBoard(gg, mn, who).Render())
		ff.WriteString("\n@@@FRAME@@@\n")
	}
	emit(g2)
	for _, m := range hist {
		_ = g2.MakeMove(m[0], m[1])
		mn++
		emit(g2)
	}
	ff.Close()

	extra := msg
	if g.State != board.StatePlaying {
		extra += "RESULT: " + g.State.String() + "\nDONE\n"
	} else {
		extra += fmt.Sprintf("AWAIT k=%d\n", len(hist))
	}
	_ = os.WriteFile(dir+"/board.txt", []byte(plain(g)+extra), 0o644)
	fmt.Print(plain(g) + extra)
}
