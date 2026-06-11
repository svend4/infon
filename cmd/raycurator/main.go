// Command raycurator is a curator agent that searches the Q6 hypercube for the world a
// viewer loves MOST (Block K: curator = engagement × maze × agent). Its goal is a
// feeling, not a coordinate: it evaluates the open worlds adjacent to where it stands
// and climbs toward the most-loved, detouring around collapsed worlds, settling on the
// best reachable one. It draws the engagement landscape over all 64 worlds (a heatmap),
// the walls, the curator's path, where it started, the world it chose, and the viewer's
// true global favourite — showing whether the maze forced a compromise.
//
//	go run ./cmd/raycurator -seed 16 -walls 0.5 -taste 110010
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
)

func main() {
	var (
		out    = flag.String("out", "curator", "output basename")
		seed   = flag.Int64("seed", 16, "maze + taste seed")
		walls  = flag.Float64("walls", 0.5, "fraction of worlds collapsed (0..0.9)")
		taste  = flag.String("taste", "", "viewer's taste as a hexagram (6 bits); empty = random from seed")
		budget = flag.Int("budget", 64, "worlds the curator may visit")
		sigma  = flag.Float64("sigma", 0.55, "viewer's taste tolerance (larger = a gentler landscape)")
	)
	flag.Parse()

	q := raydir.NewQuest(*seed, *walls)
	var pref raydir.SceneVector
	if hex, ok := raydir.ParseHexagram(*taste); ok {
		pref = raydir.VectorFromHexagram(hex)
		pref[raydir.AxWarm] = 0.9 // nudge so the peak corner is unambiguous
	} else {
		rng := rand.New(rand.NewSource(*seed))
		for i := range pref {
			pref[i] = rng.Float64()
		}
	}
	vm := raydir.ViewerModel{Pref: pref, Sigma: *sigma}
	c := raydir.NewCurator(q, vm)

	best, bestE, visited := c.Curate(q.Start, *budget)
	global := pref.Hexagram()

	fmt.Printf("Q6 curator seed %d — %d worlds collapsed; viewer's favourite is %06b (mood %q)\n",
		*seed, q.WallCount(), global.Number(), pref.Mood())
	fmt.Printf("  curator visited %d worlds, settled on %06b %q (mood %q, engagement %.3f)\n",
		len(visited), best.Number(), best.Name(), raydir.VectorFromHexagram(best).Mood(), bestE)
	switch {
	case best == global:
		fmt.Printf("  reached the viewer's global favourite ✓\n")
	case q.IsWall(global):
		fmt.Printf("  the global favourite is collapsed — curator found the best reachable world\n")
	default:
		fmt.Printf("  the maze forced a compromise (global favourite engagement %.3f, walled off)\n",
			c.Engagement(global))
	}

	writePNG(*out+".png", drawHeat(q, c, visited, q.Start, best, global))
	fmt.Printf("wrote %s.png\n", *out)
}

// heatColor maps engagement (0..1) from a cold dark blue to a warm gold.
func heatColor(e float64) color.RGBA {
	if e < 0 {
		e = 0
	}
	if e > 1 {
		e = 1
	}
	lo := [3]float64{42, 50, 82}
	hi := [3]float64{245, 210, 90}
	return color.RGBA{
		R: uint8(lo[0] + (hi[0]-lo[0])*e),
		G: uint8(lo[1] + (hi[1]-lo[1])*e),
		B: uint8(lo[2] + (hi[2]-lo[2])*e),
		A: 255,
	}
}

// drawHeat paints the engagement landscape over the 64 worlds (col = lower trigram,
// row = upper trigram), with walls, the curator's path, start, chosen world and the
// viewer's global favourite.
func drawHeat(q *raydir.Quest, c *raydir.Curator, visited []raydir.Hexagram, start, best, global raydir.Hexagram) image.Image {
	const cell, margin, labelH = 40, 14, 14
	W := 8*cell + 2*margin
	H := W + labelH
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	microfont.Draw(img, 4, 3, 1, "viewer's love over all 64 worlds: dark=cold gold=loved  walls=red  path=white  start=green box  chosen=gold box  favourite=cyan",
		color.RGBA{R: 225, G: 225, B: 230, A: 255})
	cen := func(n int) (int, int) { return margin + (n&7)*cell + cell/2, labelH + margin + (n>>3)*cell + cell/2 }
	for n := 0; n < 64; n++ {
		x0, y0 := margin+(n&7)*cell, labelH+margin+(n>>3)*cell
		hx := raydir.HexagramFromNumber(n)
		col := heatColor(c.Engagement(hx))
		if q.IsWall(hx) {
			col = color.RGBA{R: 110, G: 36, B: 40, A: 255}
		}
		fillRect(img, x0+1, y0+1, cell-2, cell-2, col)
	}
	// the worlds the curator visited, dotted (best-first hops jump across the cube, so
	// a connecting line would be spaghetti — the dots show what it sampled).
	white := color.RGBA{R: 250, G: 250, B: 252, A: 255}
	for _, h := range visited {
		cx, cy := cen(h.Number())
		fillRect(img, cx-2, cy-2, 4, 4, white)
	}
	box := func(n int, c color.RGBA, t int) {
		x0, y0 := margin+(n&7)*cell, labelH+margin+(n>>3)*cell
		for k := 0; k < t; k++ {
			rectOutline(img, x0+1+k, y0+1+k, cell-2-2*k, cell-2-2*k, c)
		}
	}
	box(global.Number(), color.RGBA{R: 90, G: 220, B: 240, A: 255}, 2) // cyan: viewer's favourite
	box(start.Number(), color.RGBA{R: 80, G: 220, B: 110, A: 255}, 2)  // green: start
	box(best.Number(), color.RGBA{R: 245, G: 205, B: 70, A: 255}, 3)   // gold: chosen
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

func rectOutline(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for i := x; i < x+w; i++ {
		img.SetRGBA(i, y, c)
		img.SetRGBA(i, y+h-1, c)
	}
	for j := y; j < y+h; j++ {
		img.SetRGBA(x, j, c)
		img.SetRGBA(x+w-1, j, c)
	}
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
