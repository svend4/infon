// Command arenaplan pits the lookahead Planner (blue) against the reactive
// RefCommander (red) on a symmetric board with a central ridge, prints the
// tick-by-tick score, and renders a top-down GIF (via pkg/arenacast) with a
// survivor bar. The planner simulates each move to the end of a reference
// rollout, so it holds ground and out-trades the bot that only reacts.
//
//	go run ./cmd/arenaplan
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/arenacast"
)

func board() *arena.Arena {
	const W, H = 9, 5
	a := &arena.Arena{W: W, H: H, Terrain: make([]uint8, W*H)}
	for y := 0; y < H; y++ {
		a.Terrain[y*W+4] = 3 // central ridge
	}
	for n, k := range []uint8{4, 3, 5} { // owl, swan, crab — mirrored
		y := uint8(1 + n)
		a.Units = append(a.Units,
			arena.Unit{Kind: k, X: 1, Y: y, HP: arena.Bestiary[k].HP, Faction: 0, Alive: true},
			arena.Unit{Kind: k, X: 7, Y: y, HP: arena.Bestiary[k].HP, Faction: 1, Alive: true},
		)
	}
	return a
}

func main() {
	ticks := flag.Int("ticks", 60, "max ticks")
	out := flag.String("out", "_arena", "output directory")
	flag.Parse()
	_ = os.MkdirAll(*out, 0o755)

	a := board()
	g := &gif.GIF{}
	fmt.Println("planner (blue) vs reference (red) — symmetric armies, central ridge")
	for t := 0; t <= *ticks; t++ {
		g.Image = append(g.Image, frame(a))
		g.Delay = append(g.Delay, 18)
		a0, a1 := a.AliveCount(0), a.AliveCount(1)
		fmt.Printf("  tick %2d  blue=%d red=%d\n", t, a0, a1)
		if a0 == 0 || a1 == 0 {
			break
		}
		a.Step(arena.Planner{}, arena.RefCommander{})
	}
	for i := 0; i < 4; i++ { // hold the final frame
		g.Image = append(g.Image, g.Image[len(g.Image)-1])
		g.Delay = append(g.Delay, 120)
	}
	a0, a1 := a.AliveCount(0), a.AliveCount(1)
	winner := "draw"
	if a0 > a1 {
		winner = "blue (planner)"
	} else if a1 > a0 {
		winner = "red (reference)"
	}
	fmt.Printf("winner: %s (%d v %d)\n", winner, a0, a1)

	path := fmt.Sprintf("%s/arenaplan.gif", *out)
	fp, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer fp.Close()
	if err := gif.EncodeAll(fp, g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", path)
}

const cvW, cvH = 520, 320

func frame(a *arena.Arena) *image.Paletted {
	c := image.NewRGBA(image.Rect(0, 0, cvW, cvH))
	fillRect(c, 0, 0, cvW, cvH, color.RGBA{14, 14, 22, 255})

	// the board, top-down, via the same marks bridge used for the semantic call.
	if img, err := arenacast.Spec(a).Image(440, 240); err == nil && img != nil {
		ox, oy := (cvW-440)/2, 60
		draw.Draw(c, image.Rect(ox, oy, ox+440, oy+240), img, img.Bounds().Min, draw.Over)
		outline(c, ox-4, oy-4, 448, 248, color.RGBA{40, 40, 56, 255})
	}

	// survivor bar at the top: blue (planner) vs orange (reference).
	a0, a1 := a.AliveCount(0), a.AliveCount(1)
	total := a0 + a1
	if total == 0 {
		total = 1
	}
	const bx, by, bw, bh = 40, 24, 440, 18
	wBlue := bw * a0 / total
	fillRect(c, bx, by, bw, bh, color.RGBA{60, 30, 30, 255})
	fillRect(c, bx, by, wBlue, bh, color.RGBA{90, 150, 235, 255})
	fillRect(c, bx+wBlue, by, bw-wBlue, bh, color.RGBA{240, 150, 60, 255})

	pi := image.NewPaletted(c.Bounds(), palette.Plan9)
	draw.FloydSteinberg.Draw(pi, c.Bounds(), c, image.Point{})
	return pi
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func outline(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	fillRect(img, x, y, w, 2, c)
	fillRect(img, x, y+h-2, w, 2, c)
	fillRect(img, x, y, 2, h, c)
	fillRect(img, x+w-2, y, 2, h, c)
}
