// Command braindemo proves the open tvcp-ai/1 format end-to-end over real HTTP:
// it starts the reference brain on localhost, then (1) plays a full tic-tac-toe
// where X's moves come from the brain via the protocol, O is random; and (2)
// asks the brain to draw a scene and renders the returned draw-DSL. Any other
// backend (Ollama/OpenAI/this model) drops in by changing the endpoint URL.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/terminal"
)

func other(m string) string {
	if m == "X" {
		return "O"
	}
	return "X"
}

func winner(b [3][3]string) string {
	lines := [8][3][2]int{
		{{0, 0}, {0, 1}, {0, 2}}, {{1, 0}, {1, 1}, {1, 2}}, {{2, 0}, {2, 1}, {2, 2}},
		{{0, 0}, {1, 0}, {2, 0}}, {{0, 1}, {1, 1}, {2, 1}}, {{0, 2}, {1, 2}, {2, 2}},
		{{0, 0}, {1, 1}, {2, 2}}, {{0, 2}, {1, 1}, {2, 0}},
	}
	for _, l := range lines {
		a := b[l[0][0]][l[0][1]]
		if a != "" && a == b[l[1][0]][l[1][1]] && a == b[l[2][0]][l[2][1]] {
			return a
		}
	}
	return ""
}

func full(b [3][3]string) bool {
	for i := range b {
		for j := range b[i] {
			if b[i][j] == "" {
				return false
			}
		}
	}
	return true
}

func empties(b [3][3]string) [][2]int {
	var e [][2]int
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if b[i][j] == "" {
				e = append(e, [2]int{i, j})
			}
		}
	}
	return e
}

func renderBoard(b [3][3]string, moveNo int, state string) *terminal.Frame {
	const W, H = 33, 17
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(14, 16, 28)
	f.Fill(' ', color.NewRGB(220, 220, 230), bg)
	red := color.NewRGB(235, 70, 70)
	cyan := color.NewRGB(70, 205, 235)
	grid := color.NewRGB(90, 95, 120)
	f.DrawText(1, 0, "brain X (tvcp-ai/1) vs random O", color.NewRGB(235, 235, 245), bg)
	f.DrawText(1, 1, fmt.Sprintf("move %d  -  %s", moveNo, state), color.NewRGB(150, 160, 190), bg)
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
			switch b[r][c] {
			case "X":
				f.SetBlock(cx, cy, 'X', red, bg)
			case "O":
				f.SetBlock(cx, cy, 'O', cyan, bg)
			}
		}
	}
	return f
}

func main() {
	brainURL := flag.String("brain", "", "external tvcp-ai/1 brain endpoint (default: in-process reference)")
	promptFlag := flag.String("prompt", "a calm horizon at dusk", "draw prompt")
	flag.Parse()
	var url string
	if *brainURL != "" {
		url = *brainURL
	} else {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/v1/decide", brain.Handler())
		go func() { _ = http.Serve(ln, mux) }()
		url = "http://" + ln.Addr().String() + "/v1/decide"
	}
	fmt.Println("brain endpoint:", url)
	hb := brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}

	gf, _ := os.Create("assets/brain_game.frames.txt")
	mn := 0
	emit := func(b [3][3]string, st string) {
		gf.WriteString(renderBoard(b, mn, st).Render())
		gf.WriteString("\n@@@FRAME@@@\n")
	}
	var b [3][3]string
	emit(b, "Playing")
	turn := "X"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := "Draw"
	for {
		if turn == "X" {
			raw, _ := json.Marshal(brain.TicTacToeState{Board: b, Turn: "X", You: "X", Legal: empties(b)})
			resp, err := hb.Decide(brain.Request{Kind: "move", Game: "tictactoe", State: raw})
			if err != nil || resp.Move == nil {
				fmt.Fprintln(os.Stderr, "brain error:", err, resp.Error)
				gf.Close()
				os.Exit(1)
			}
			r, c := resp.Move.Row, resp.Move.Col
			if r < 0 || r > 2 || c < 0 || c > 2 || b[r][c] != "" {
				e := empties(b) // brain proposed an illegal move; fall back to a legal one
				r, c = e[0][0], e[0][1]
			}
			b[r][c] = "X"
		} else {
			e := empties(b)
			m := e[rng.Intn(len(e))]
			b[m[0]][m[1]] = "O"
		}
		mn++
		if w := winner(b); w != "" {
			result = w + " Wins"
			emit(b, result)
			break
		}
		if full(b) {
			result = "Draw"
			emit(b, result)
			break
		}
		emit(b, "Playing")
		turn = other(turn)
	}
	gf.Close()

	resp, err := hb.Decide(brain.Request{Kind: "draw", Prompt: *promptFlag, Canvas: &brain.Canvas{Width: 64, Height: 20}})
	if err == nil && len(resp.Scene) > 0 {
		var sc scene.Scene
		if json.Unmarshal(resp.Scene, &sc) == nil {
			sc.Width, sc.Height = 64, 20 // enforce canvas
			_ = os.WriteFile("assets/brain_draw.txt", []byte(scene.RenderScene(&sc).Render()), 0o644)
		}
	}
	fmt.Println("game result:", result, "| draw scene bytes:", len(resp.Scene))
}
