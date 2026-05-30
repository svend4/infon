// Command aicards renders a glyph "card" message + a truecolor effect via
// pkg/terminal.Frame. With -brain URL it asks any tvcp-ai/1 brain for a
// kind=react reply (-msg) and renders the cards the model chooses.
package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

type rc struct {
	g     rune
	fg    color.RGB
	label string
}

var cardMap = map[string]rc{
	"star":     {'★', color.NewRGB(245, 200, 70), "star"},
	"heart":    {'♥', color.NewRGB(235, 80, 90), "love"},
	"check":    {'✓', color.NewRGB(90, 210, 120), "ok"},
	"ok":       {'✓', color.NewRGB(90, 210, 120), "ok"},
	"thumbsup": {'✓', color.NewRGB(90, 210, 120), "yes"},
	"smile":    {'☺', color.NewRGB(245, 215, 90), "smile"},
	"sad":      {'☹', color.NewRGB(120, 160, 235), "sad"},
	"fire":     {'▲', color.NewRGB(240, 130, 50), "fire"},
	"warning":  {'!', color.NewRGB(240, 180, 60), "warn"},
	"x":        {'✕', color.NewRGB(235, 90, 90), "no"},
	"music":    {'♪', color.NewRGB(170, 130, 230), "music"},
	"sun":      {'☀', color.NewRGB(245, 200, 80), "sun"},
}

func mapCard(s string) rc {
	if v, ok := cardMap[strings.ToLower(strings.TrimSpace(s))]; ok {
		return v
	}
	g := '●'
	for _, r := range s {
		g = r
		break
	}
	lbl := s
	if len(lbl) > 6 {
		lbl = lbl[:6]
	}
	return rc{g, color.NewRGB(220, 220, 235), lbl}
}

func brainReact(url, msg string) ([]rc, bool) {
	hb := brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}
	resp, err := hb.Decide(brain.Request{Kind: "react", Prompt: msg})
	if err != nil || len(resp.Cards) == 0 {
		return nil, false
	}
	seen := map[rune]bool{}
	var out []rc
	for _, name := range resp.Cards {
		card := mapCard(name)
		if seen[card.g] {
			continue
		}
		seen[card.g] = true
		out = append(out, card)
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out, true
}

func hsv(h, s, v float64) color.RGB {
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.NewRGB(uint8((r+m)*255), uint8((g+m)*255), uint8((b+m)*255))
}

func drawCard(f *terminal.Frame, x, y int, rank string, suit rune, red bool) {
	light := color.NewRGB(245, 245, 238)
	ink := color.NewRGB(30, 30, 36)
	if red {
		ink = color.NewRGB(200, 40, 40)
	}
	w, h := 9, 5
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			f.SetBlock(xx, yy, ' ', ink, light)
		}
	}
	f.DrawBox(x, y, w, h, color.NewRGB(120, 120, 130), light)
	f.DrawText(x+1, y+1, rank, ink, light)
	f.SetBlock(x+w/2, y+h/2, suit, ink, light)
	f.DrawText(x+w-1-len(rank), y+h-2, rank, ink, light)
}

func cardsFrame(react []rc, title string) *terminal.Frame {
	const W, H = 60, 14
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(12, 16, 26)
	f.Fill(' ', color.NewRGB(220, 220, 230), bg)
	f.DrawText(2, 0, title, color.NewRGB(235, 235, 245), bg)
	type cd struct {
		r   string
		s   rune
		red bool
	}
	hand := []cd{{"A", '♠', false}, {"K", '♥', true}, {"Q", '♦', true}, {"J", '♣', false}, {"10", '♥', true}}
	x := 2
	for _, c := range hand {
		drawCard(f, x, 2, c.r, c.s, c.red)
		x += 11
	}
	x = 2
	for _, r := range react {
		f.SetBlock(x, 10, r.g, r.fg, bg)
		f.DrawText(x+2, 10, r.label, color.NewRGB(200, 205, 220), bg)
		x += 12
	}
	return f
}

func effectFrame(t, n int) *terminal.Frame {
	const W, H = 60, 9
	f := terminal.NewFrame(W, H)
	bg := color.NewRGB(8, 9, 18)
	f.Fill(' ', color.NewRGB(200, 200, 210), bg)
	f.DrawText(2, 1, "Claude . visual FX  (truecolor sweep)", color.NewRGB(230, 230, 245), bg)
	for x := 2; x < W-2; x++ {
		hue := float64(x*7) + float64(t)*(360.0/float64(n))
		c := hsv(hue, 0.85, 1.0)
		f.SetBlock(x, 4, '█', c, bg)
		f.SetBlock(x, 5, '▀', c, bg)
	}
	stars := [][2]int{{6, 2}, {14, 7}, {22, 2}, {30, 7}, {38, 2}, {46, 7}, {52, 2}, {10, 7}, {34, 2}, {50, 7}}
	glyphs := []rune{'.', '+', '*'}
	for i, s := range stars {
		on := (t + i) % 3
		b := uint8(120 + 45*on)
		f.SetBlock(s[0], s[1], glyphs[on], color.NewRGB(b, b, b), bg)
	}
	f.DrawText(2, 7, "♥  T V C P   .   A I  ♥", color.NewRGB(235, 120, 140), bg)
	return f
}

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai/1 brain endpoint for kind=react cards")
	msg := flag.String("msg", "good game, nice move!", "message the brain reacts to (with -brain)")
	flag.Parse()

	react := []rc{
		{'★', color.NewRGB(245, 200, 70), "nice"},
		{'♥', color.NewRGB(235, 80, 90), "love"},
		{'✓', color.NewRGB(90, 210, 120), "ok"},
		{'!', color.NewRGB(240, 180, 60), "look"},
	}
	title := "Claude . visual message  (cards = a low-bandwidth language)"
	if *brainURL != "" {
		if got, ok := brainReact(*brainURL, *msg); ok {
			react = got
			title = "brain react (tvcp-ai/1) to: " + *msg
		}
	}

	fmt.Print(cardsFrame(react, title).Render())
	fmt.Print("\n@@@FRAME@@@\n")
	const n = 12
	for t := 0; t < n; t++ {
		fmt.Print(effectFrame(t, n).Render())
		fmt.Print("\n@@@FRAME@@@\n")
	}
}
