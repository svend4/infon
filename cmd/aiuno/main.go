// Command aiuno plays UNO where a tvcp-ai/1 brain controls one player, using the
// repo's experimental/games/cards engine. Demonstrates the protocol on a card
// game (Move.CardIndex / Draw / Color).
//
//	go run -tags experimental ./cmd/aiuno -brain http://127.0.0.1:8088/v1/decide
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/svend4/infon/experimental/games/cards"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func colorName(c cards.UnoColor) string {
	switch c {
	case cards.ColorRed:
		return "Red"
	case cards.ColorBlue:
		return "Blue"
	case cards.ColorGreen:
		return "Green"
	case cards.ColorYellow:
		return "Yellow"
	default:
		return "Wild"
	}
}

func colorRGB(c cards.UnoColor) color.RGB {
	switch c {
	case cards.ColorRed:
		return color.NewRGB(220, 70, 70)
	case cards.ColorBlue:
		return color.NewRGB(70, 120, 235)
	case cards.ColorGreen:
		return color.NewRGB(70, 190, 95)
	case cards.ColorYellow:
		return color.NewRGB(230, 200, 70)
	default:
		return color.NewRGB(150, 150, 160)
	}
}

func cardLabel(c *cards.UnoCard) string {
	if c == nil {
		return "-"
	}
	if c.Value == cards.ValueWild || c.Value == cards.ValueWildDraw4 {
		return c.Value.String()
	}
	return colorName(c.Color) + " " + c.Value.String()
}

func parseColor(name string) cards.UnoColor {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "blue":
		return cards.ColorBlue
	case "green":
		return cards.ColorGreen
	case "yellow":
		return cards.ColorYellow
	default:
		return cards.ColorRed
	}
}

func canPlay(c, top *cards.UnoCard, cur cards.UnoColor) bool {
	if c.Value == cards.ValueWild || c.Value == cards.ValueWildDraw4 {
		return true
	}
	return c.Color == cur || c.Value == top.Value
}

func playable(p *cards.UnoPlayer, top *cards.UnoCard, cur cards.UnoColor) []int {
	var idx []int
	for i, c := range p.Hand {
		if canPlay(c, top, cur) {
			idx = append(idx, i)
		}
	}
	return idx
}

func bestColor(p *cards.UnoPlayer) cards.UnoColor {
	counts := map[cards.UnoColor]int{}
	for _, c := range p.Hand {
		if c.Color != cards.ColorWild {
			counts[c.Color]++
		}
	}
	best, bn := cards.ColorRed, -1
	for _, col := range []cards.UnoColor{cards.ColorRed, cards.ColorBlue, cards.ColorGreen, cards.ColorYellow} {
		if counts[col] > bn {
			bn, best = counts[col], col
		}
	}
	return best
}

func renderUno(g *cards.UnoGame, last string) *terminal.Frame {
	const W, H = 58, 15
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(14, 16, 26)
	f.Fill(' ', color.NewRGB(225, 225, 235), bg)
	f.DrawText(1, 0, "UNO - brain vs bot  (tvcp-ai/1)", color.NewRGB(235, 235, 245), bg)
	top := g.GetTopCard()
	cc := colorRGB(g.CurrentColor)
	f.DrawText(2, 2, "current color:", color.NewRGB(160, 170, 200), bg)
	f.SetBlock(16, 2, ' ', cc, cc)
	f.SetBlock(17, 2, ' ', cc, cc)
	f.DrawText(19, 2, colorName(g.CurrentColor), cc, bg)
	f.DrawText(2, 3, "top card:", color.NewRGB(160, 170, 200), bg)
	lbl := cardLabel(top)
	tc := color.NewRGB(200, 200, 210)
	if top != nil {
		tc = colorRGB(top.Color)
	}
	for i := 0; i < len(lbl)+2; i++ {
		f.SetBlock(12+i, 3, ' ', tc, tc)
	}
	f.DrawText(13, 3, lbl, color.NewRGB(20, 20, 26), tc)
	bn, bt := 0, 0
	for _, p := range g.Players {
		if p.ID == "brain" {
			bn = len(p.Hand)
		} else {
			bt = len(p.Hand)
		}
	}
	f.DrawText(2, 5, fmt.Sprintf("Brain: %d cards     Bot: %d cards", bn, bt), color.NewRGB(225, 225, 235), bg)
	cur := g.GetCurrentPlayer()
	who := "-"
	if cur != nil {
		who = cur.Name
	}
	f.DrawText(2, 6, "turn: "+who, color.NewRGB(150, 200, 235), bg)
	f.DrawText(2, 8, "last: "+last, color.NewRGB(235, 220, 150), bg)
	// brain hand preview
	if cur != nil {
		var hp *cards.UnoPlayer
		for _, p := range g.Players {
			if p.ID == "brain" {
				hp = p
			}
		}
		if hp != nil {
			x := 2
			f.DrawText(2, 10, "brain hand:", color.NewRGB(160, 170, 200), bg)
			y := 11
			for i, c := range hp.Hand {
				if i >= 7 {
					break
				}
				cl := cardLabel(c)
				cr := colorRGB(c.Color)
				for j := 0; j < len(cl)+2; j++ {
					f.SetBlock(x+j, y, ' ', cr, cr)
				}
				f.DrawText(x+1, y, cl, color.NewRGB(20, 20, 26), cr)
				x += len(cl) + 3
			}
		}
	}
	if g.State == cards.UnoStateFinished && g.Winner != nil {
		f.DrawText(2, 1, "WINNER: "+g.Winner.Name, color.NewRGB(90, 230, 120), bg)
	}
	return f
}

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai/1 brain controlling the 'brain' player")
	flag.Parse()

	g := cards.NewUnoGame("u")
	_ = g.AddPlayer("brain", "Brain")
	_ = g.AddPlayer("bot", "Bot")
	_ = g.Start()
	hb := brain.HTTPBrain{URL: *brainURL, HTTP: &http.Client{Timeout: 120 * time.Second}}

	ff, _ := os.Create("assets/uno.frames.txt")
	emit := func(last string) { ff.WriteString(renderUno(g, last).Render()); ff.WriteString("\n@@@FRAME@@@\n") }
	emit("deal")

	turns := 0
	for g.State == cards.UnoStatePlaying && turns < 80 {
		turns++
		cur := g.GetCurrentPlayer()
		if cur == nil {
			break
		}
		top := g.GetTopCard()
		idxs := playable(cur, top, g.CurrentColor)
		last := ""
		if cur.ID == "brain" && *brainURL != "" || cur.ID == "brain" {
			var hand []string
			for _, c := range cur.Hand {
				hand = append(hand, cardLabel(c))
			}
			state, _ := json.Marshal(map[string]interface{}{
				"hand": hand, "top": cardLabel(top), "color": colorName(g.CurrentColor),
				"playable": idxs, "you": "brain",
			})
			var resp brain.Response
			var derr error
			if *brainURL != "" {
				resp, derr = hb.Decide(brain.Request{Kind: "move", Game: "uno", State: state})
			} else {
				resp = brain.Reference(brain.Request{Kind: "move", Game: "uno", State: state})
			}
			done := false
			if derr == nil && resp.Move != nil && !resp.Move.Draw && resp.Move.CardIndex != nil {
				i := *resp.Move.CardIndex
				valid := false
				for _, v := range idxs {
					if v == i {
						valid = true
					}
				}
				if valid {
					c := cur.Hand[i]
					if g.PlayCard("brain", i, parseColor(resp.Move.Color)) == nil {
						last, done = "brain played "+cardLabel(c), true
					}
				}
			}
			if !done {
				if len(idxs) > 0 {
					c := cur.Hand[idxs[0]]
					if g.PlayCard("brain", idxs[0], bestColor(cur)) == nil {
						last, done = "brain played "+cardLabel(c), true
					}
				}
				if !done {
					_ = g.DrawCard("brain")
					last = "brain drew a card"
				}
			}
		} else {
			if len(idxs) > 0 {
				c := cur.Hand[idxs[0]]
				_ = g.PlayCard("bot", idxs[0], bestColor(cur))
				last = "bot played " + cardLabel(c)
			} else {
				_ = g.DrawCard("bot")
				last = "bot drew a card"
			}
		}
		emit(last)
	}
	winner := "nobody"
	if g.Winner != nil {
		winner = g.Winner.Name
	}
	emit("GAME OVER - winner: " + winner)
	ff.Close()
	fmt.Println("winner:", winner, "turns:", turns)
}
