// Command aiwordle plays Wordle where a tvcp-ai/1 brain guesses, using the
// repo's experimental/games/words engine for validation/feedback. Shows the
// protocol generalizing beyond tic-tac-toe to a word game (Move.Word).
//
//	go run -tags experimental ./cmd/aiwordle -brain http://127.0.0.1:8088/v1/decide -word WATER
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/svend4/infon/experimental/games/words"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func mark(s words.LetterStatus) string {
	switch s {
	case words.LetterCorrect:
		return "correct"
	case words.LetterMisplaced:
		return "present"
	default:
		return "absent"
	}
}

func fallbackGuess(g *words.WordleGame, used map[string]bool) string {
	for _, w := range g.Dictionary {
		u := strings.ToUpper(w)
		if len(u) == g.WordLength && !used[u] {
			return u
		}
	}
	return strings.Repeat("A", g.WordLength)
}

func renderWordle(g *words.WordleGame, note string) *terminal.Frame {
	const W, H = 40, 26
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(16, 18, 28)
	f.Fill(' ', color.NewRGB(225, 225, 235), bg)
	f.DrawText(1, 0, "Wordle - brain guesses (tvcp-ai/1)", color.NewRGB(235, 235, 245), bg)
	f.DrawText(1, 1, note, color.NewRGB(160, 170, 200), bg)
	ox, oy, cw, chh := 2, 3, 7, 3
	ink := color.NewRGB(245, 245, 245)
	for r := 0; r < g.MaxAttempts; r++ {
		for c := 0; c < g.WordLength; c++ {
			x, y := ox+c*cw, oy+r*chh
			letter := ' '
			cellbg := color.NewRGB(40, 42, 55)
			if r < len(g.Attempts) {
				a := g.Attempts[r]
				if c < len(a.Word) {
					letter = rune(a.Word[c])
				}
				switch a.Result[c] {
				case words.LetterCorrect:
					cellbg = color.NewRGB(70, 160, 80)
				case words.LetterMisplaced:
					cellbg = color.NewRGB(205, 170, 60)
				default:
					cellbg = color.NewRGB(70, 72, 85)
				}
			}
			for yy := y; yy < y+chh-1+1; yy++ {
				for xx := x; xx < x+cw-1; xx++ {
					f.SetBlock(xx, yy, ' ', ink, cellbg)
				}
			}
			f.SetBlock(x+(cw-1)/2, y+1, letter, ink, cellbg)
		}
	}
	return f
}

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai/1 brain that guesses")
	word := flag.String("word", "WATER", "the secret word")
	flag.Parse()

	g := words.NewWordleGame("w", *word)
	_ = g.AddPlayer("brain", "Brain")
	_ = g.Start()
	hb := brain.HTTPBrain{URL: *brainURL, HTTP: &http.Client{Timeout: 120 * time.Second}}

	ff, _ := os.Create("assets/wordle.frames.txt")
	emit := func(note string) { ff.WriteString(renderWordle(g, note).Render()); ff.WriteString("\n@@@FRAME@@@\n") }
	emit("start - secret is " + fmt.Sprintf("%d letters", g.WordLength))

	note := ""
	for attempt := 0; attempt < g.MaxAttempts; attempt++ {
		used := map[string]bool{}
		var gs []map[string]interface{}
		for _, a := range g.Attempts {
			used[strings.ToUpper(a.Word)] = true
			marks := make([]string, len(a.Result))
			for i, r := range a.Result {
				marks[i] = mark(r)
			}
			gs = append(gs, map[string]interface{}{"word": a.Word, "marks": marks})
		}
		state, _ := json.Marshal(map[string]interface{}{
			"length": g.WordLength, "attempts_left": g.MaxAttempts - len(g.Attempts),
			"guesses": gs, "you": "guesser",
		})
		var resp brain.Response
		var derr error
		if *brainURL != "" {
			resp, derr = hb.Decide(brain.Request{Kind: "move", Game: "wordle", State: state})
		} else {
			resp = brain.Reference(brain.Request{Kind: "move", Game: "wordle", State: state})
		}
		guess := ""
		if derr == nil && resp.Move != nil {
			guess = strings.ToUpper(strings.TrimSpace(resp.Move.Word))
		}
		if len(guess) != g.WordLength || used[guess] {
			guess = fallbackGuess(g, used)
		}
		att, err := g.MakeGuess("brain", guess)
		if err != nil {
			guess = fallbackGuess(g, used)
			if att, err = g.MakeGuess("brain", guess); err != nil {
				note = "error: " + err.Error()
				emit(note)
				break
			}
		}
		_ = att
		if g.Players["brain"].Won {
			note = fmt.Sprintf("SOLVED '%s' in %d!", guess, len(g.Attempts))
			emit(note)
			break
		}
		note = fmt.Sprintf("guess %d: %s", len(g.Attempts), guess)
		emit(note)
	}
	if !g.Players["brain"].Won {
		note = "out of tries - the word was " + strings.ToUpper(*word)
		emit(note)
	}
	ff.Close()
	fmt.Println(note)
}
