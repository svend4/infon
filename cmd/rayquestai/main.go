// Command rayquestai is an agent that PLAYS the Q6 roguelike (Block H: agent × maze).
// Where rayquest solves the maze omnisciently, this agent navigates with fog of war:
// it cannot see a collapsed room until it tries to step into one and bumps. It heads
// for the goal along the shortest route it currently believes exists (unseen rooms
// assumed open), re-plans whenever a bump reveals a wall, and remembers every wall it
// finds. Replayed on the same maze its cost collapses toward the omniscient optimum as
// its map fills in. It draws the cube map with the agent's first (wandering) route and
// its final (learned) route, and a bar chart of cost (steps + wall-bumps) per episode
// falling to the optimum — the agent learning the dungeon by playing it.
//
//	go run ./cmd/rayquestai -seed 16 -walls 0.6 -episodes 12
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
)

func main() {
	var (
		out      = flag.String("out", "questai", "output basename")
		seed     = flag.Int64("seed", 16, "maze seed")
		walls    = flag.Float64("walls", 0.6, "fraction of rooms collapsed (0..0.9)")
		episodes = flag.Int("episodes", 12, "episodes to replay (learning)")
	)
	flag.Parse()

	q := raydir.NewQuest(*seed, *walls)
	opt, ok := q.Solve()
	if !ok {
		fmt.Fprintln(os.Stderr, "no route")
		os.Exit(1)
	}
	optimal := len(opt) - 1

	a := raydir.NewQuestAgent(q)
	var steps, cost []int // steps walked, and total actions (steps + wall bumps)
	var first, final []raydir.Hexagram
	bumps0 := 0
	for e := 0; e < *episodes; e++ {
		before := a.Bumps
		r, _ := a.Run(1024)
		s := len(r) - 1
		bp := a.Bumps - before
		steps = append(steps, s)
		cost = append(cost, s+bp)
		if e == 0 {
			first, bumps0 = r, bp
		}
		final = r
	}

	converged := *episodes
	for i, s := range steps {
		if s == optimal {
			converged = i + 1
			break
		}
	}
	fmt.Printf("Q6 quest AI seed %d — %d rooms collapsed, optimum %d steps\n", *seed, q.WallCount(), optimal)
	fmt.Printf("  episode 1 (fog of war): %d steps + %d wall-bumps (cost %d)\n", steps[0], bumps0, cost[0])
	fmt.Printf("  final: %d steps + 0 bumps   optimum: %d steps\n", steps[len(steps)-1], optimal)
	fmt.Printf("  converged to the optimum at episode %d; explored %d/64 rooms\n", converged, a.Explored())
	fmt.Printf("  cost/episode %s\n", sparkline(cost))

	writePNG(*out+".png", vstack(
		drawMap(q, first, final), "cube map: walls=red start=green goal=gold  first route=orange  learned=white",
		drawCurve(cost, optimal), fmt.Sprintf("cost per episode = steps + wall-bumps (falling to the optimum %d)", optimal),
	))
	fmt.Printf("wrote %s.png\n", *out)
}

// drawMap renders the 8×8 cube (col = lower trigram, row = upper trigram) with the
// agent's first (wandering) route faint and its final (learned) route bright.
func drawMap(q *raydir.Quest, first, final []raydir.Hexagram) image.Image {
	const cell, margin = 34, 12
	W := 8*cell + 2*margin
	img := image.NewRGBA(image.Rect(0, 0, W, W))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	onFinal := map[int]bool{}
	for _, h := range final {
		onFinal[h.Number()] = true
	}
	center := func(n int) (int, int) { return margin + (n&7)*cell + cell/2, margin + (n>>3)*cell + cell/2 }
	for n := 0; n < 64; n++ {
		x0, y0 := margin+(n&7)*cell, margin+(n>>3)*cell
		c := color.RGBA{R: 44, G: 48, B: 60, A: 255}
		switch {
		case q.IsWall(raydir.HexagramFromNumber(n)):
			c = color.RGBA{R: 120, G: 40, B: 44, A: 255}
		case onFinal[n]:
			c = color.RGBA{R: 60, G: 90, B: 150, A: 255}
		}
		if n == q.Start.Number() {
			c = color.RGBA{R: 70, G: 170, B: 90, A: 255}
		}
		if n == q.Goal.Number() {
			c = color.RGBA{R: 210, G: 170, B: 60, A: 255}
		}
		fillRect(img, x0+1, y0+1, cell-2, cell-2, c)
	}
	orange := color.RGBA{R: 230, G: 150, B: 60, A: 255}
	for i := 1; i < len(first); i++ { // the wandering first route, thin
		x0, y0 := center(first[i-1].Number())
		x1, y1 := center(first[i].Number())
		drawLine(img, x0, y0, x1, y1, orange, false)
	}
	white := color.RGBA{R: 245, G: 245, B: 250, A: 255}
	for i := 1; i < len(final); i++ { // the learned route, bold
		x0, y0 := center(final[i-1].Number())
		x1, y1 := center(final[i].Number())
		drawLine(img, x0, y0, x1, y1, white, true)
	}
	return img
}

// drawCurve draws a bar chart of steps per episode with a reference line at optimal.
func drawCurve(steps []int, optimal int) image.Image {
	const h, pad = 150, 12
	bw := 8
	if len(steps) > 0 && len(steps)*bw > 360 {
		bw = 360 / len(steps)
	}
	if bw < 2 {
		bw = 2
	}
	W := len(steps)*bw + 2*pad
	img := image.NewRGBA(image.Rect(0, 0, W, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	mx := optimal
	for _, s := range steps {
		if s > mx {
			mx = s
		}
	}
	if mx < 1 {
		mx = 1
	}
	base := h - pad
	scale := float64(h-2*pad) / float64(mx)
	// optimal reference line
	oy := base - int(float64(optimal)*scale)
	for x := pad; x < W-pad; x++ {
		img.SetRGBA(x, oy, color.RGBA{R: 90, G: 200, B: 120, A: 255})
	}
	for i, s := range steps {
		bh := int(float64(s) * scale)
		x0 := pad + i*bw
		col := color.RGBA{R: 90, G: 140, B: 220, A: 255}
		if s == optimal {
			col = color.RGBA{R: 90, G: 210, B: 130, A: 255} // reached the optimum
		}
		fillRect(img, x0, base-bh, bw-1, bh, col)
	}
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if image.Pt(i, j).In(img.Bounds()) {
				img.SetRGBA(i, j, c)
			}
		}
	}
}

// drawLine is Bresenham; bold draws a 2px-wide line.
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, bold bool) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	for {
		img.SetRGBA(x0, y0, c)
		if bold {
			img.SetRGBA(x0+1, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func sign(a int) int {
	switch {
	case a > 0:
		return 1
	case a < 0:
		return -1
	default:
		return 0
	}
}

func sparkline(xs []int) string {
	const ramp = " ▁▂▃▄▅▆▇█"
	lo, hi := xs[0], xs[0]
	for _, x := range xs {
		if x < lo {
			lo = x
		}
		if x > hi {
			hi = x
		}
	}
	rng := hi - lo
	if rng == 0 {
		rng = 1
	}
	var b strings.Builder
	for _, x := range xs {
		lvl := (x - lo) * 8 / rng
		if lvl < 0 {
			lvl = 0
		}
		if lvl > 8 {
			lvl = 8
		}
		b.WriteRune([]rune(ramp)[lvl])
	}
	return fmt.Sprintf("%s  [%d..%d]", b.String(), lo, hi)
}

func vstack(a image.Image, la string, b image.Image, lb string) image.Image {
	const gap, labelH = 8, 16
	aw, ah := a.Bounds().Dx(), a.Bounds().Dy()
	bw, bh := b.Bounds().Dx(), b.Bounds().Dy()
	W := aw
	if bw > W {
		W = bw
	}
	H := labelH + ah + gap + labelH + bh
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	white := color.RGBA{R: 230, G: 230, B: 235, A: 255}
	microfont.Draw(out, 4, 3, 1, la, white)
	draw.Draw(out, image.Rect((W-aw)/2, labelH, (W-aw)/2+aw, labelH+ah), a, a.Bounds().Min, draw.Src)
	y := labelH + ah + gap
	microfont.Draw(out, 4, y+3, 1, lb, white)
	draw.Draw(out, image.Rect((W-bw)/2, y+labelH, (W-bw)/2+bw, y+labelH+bh), b, b.Bounds().Min, draw.Src)
	return out
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
