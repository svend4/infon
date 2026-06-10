// Command rayquest plays the Q6 hypercube as a roguelike (Block D). The 64 hexagrams
// are rooms; the only move is to change one line — flip a single trait of the world —
// stepping to an adjacent room. Some rooms have collapsed (walls), so reaching the
// goal is a maze through six-dimensional space, not the Hamming bee-line. It solves
// the maze (breadth-first over the line-flip graph), draws the 8×8 map of the cube
// with the walls and the route, prints the route as its changing lines (the I-Ching
// moving lines), and ray-traces the worlds you travel through — each room authored
// from its hexagram's Q6 vector, so the maze is also a tour of real places.
//
//	go run ./cmd/rayquest -seed 7 -walls 0.5
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out   = flag.String("out", "quest", "output basename")
		seed  = flag.Int64("seed", 7, "maze seed")
		walls = flag.Float64("walls", 0.5, "fraction of rooms collapsed (0..0.9)")
		rooms = flag.Int("rooms", 4, "worlds to ray-trace along the route")
		w     = flag.Int("w", 200, "world thumbnail width")
		h     = flag.Int("h", 150, "world thumbnail height")
		spp   = flag.Int("spp", 48, "samples per pixel")
	)
	flag.Parse()

	q := raydir.NewQuest(*seed, *walls)
	path, ok := q.Solve()
	if !ok {
		fmt.Fprintln(os.Stderr, "no route (should not happen — generator guarantees one)")
		os.Exit(1)
	}

	hamming := q.Start.Hamming(q.Goal)
	fmt.Printf("Q6 quest seed %d — %d rooms collapsed, %d of 64 reachable\n", *seed, q.WallCount(), q.Reachable())
	fmt.Printf("start %s (%06b)  ->  goal %s (%06b)\n",
		q.Start.Name(), q.Start.Number(), q.Goal.Name(), q.Goal.Number())
	fmt.Printf("route: %d steps (the straight line is %d — the maze adds %d)\n", len(path)-1, hamming, len(path)-1-hamming)
	for i := 1; i < len(path); i++ {
		m := raydir.Move(path[i-1], path[i])
		dir := "yin"
		if m.ToYang {
			dir = "yang"
		}
		fmt.Printf("  step %2d: line %d -> %-4s (%-7s)  now %06b %s\n",
			i, m.Line+1, dir, m.Trait, path[i].Number(), path[i].Name())
	}

	// the map of the cube, then the worlds along the route, stacked.
	mapImg := drawMap(q, path)
	worlds := drawWorlds(path, pickRooms(len(path), *rooms), *w, *h, *spp)
	writePNG(*out+".png", vstack(mapImg, "Q6 maze: walls=red start=green goal=gold route=white", worlds, "worlds along the route"))
	fmt.Printf("wrote %s.png\n", *out)
}

// pickRooms chooses up to k indices along a path of length n, always including the
// first and last.
func pickRooms(n, k int) []int {
	if k >= n {
		k = n
	}
	if k <= 1 {
		return []int{0}
	}
	out := make([]int, k)
	for i := 0; i < k; i++ {
		out[i] = i * (n - 1) / (k - 1)
	}
	return out
}

func drawWorlds(path []raydir.Hexagram, idx []int, w, h, spp int) image.Image {
	opt := raytrace.PathOptions{Samples: spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	const gap, labelH = 8, 16
	W := len(idx)*w + (len(idx)-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, h+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for j, i := range idx {
		v := raydir.VectorFromHexagram(path[i])
		img := raytrace.PostProcess(raytrace.PathRender(raydir.BuildScene(v.SceneSpec()), cam, w, h, opt), 1.0, 0.9, 0.4)
		x := j * (w + gap)
		draw.Draw(out, image.Rect(x, labelH, x+w, labelH+h), img, img.Bounds().Min, draw.Src)
		label := fmt.Sprintf("%06b %s", path[i].Number(), v.Mood())
		microfont.Draw(out, x+4, 3, 1, label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
	return out
}

// drawMap renders the 64-room cube as an 8×8 grid (col = lower trigram, row = upper
// trigram), colouring walls, start, goal and the route, with the route drawn as a
// polyline (some links jump — those are the hypercube's third-dimension edges).
func drawMap(q *raydir.Quest, path []raydir.Hexagram) image.Image {
	const cell, margin = 34, 12
	W := 8*cell + 2*margin
	img := image.NewRGBA(image.Rect(0, 0, W, W))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)

	onPath := map[int]bool{}
	for _, hx := range path {
		onPath[hx.Number()] = true
	}
	cellAt := func(n int) (int, int) { // centre of the room's cell
		col, rowi := n&7, n>>3
		return margin + col*cell + cell/2, margin + rowi*cell + cell/2
	}
	for n := 0; n < 64; n++ {
		col, rowi := n&7, n>>3
		x0, y0 := margin+col*cell, margin+rowi*cell
		c := color.RGBA{R: 44, G: 48, B: 60, A: 255} // open room
		switch {
		case q.IsWall(raydir.HexagramFromNumber(n)):
			c = color.RGBA{R: 120, G: 40, B: 44, A: 255}
		case onPath[n]:
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
	// the route as a white polyline through cell centres
	for i := 1; i < len(path); i++ {
		x0, y0 := cellAt(path[i-1].Number())
		x1, y1 := cellAt(path[i].Number())
		drawLine(img, x0, y0, x1, y1, color.RGBA{R: 240, G: 240, B: 245, A: 255})
	}
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			img.SetRGBA(i, j, c)
		}
	}
}

// drawLine is a 2-pixel-wide Bresenham line (so the route reads over the cells).
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	for {
		img.SetRGBA(x0, y0, c)
		img.SetRGBA(x0+1, y0, c)
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

// vstack stacks two labelled panels, centred, on a dark canvas.
func vstack(a image.Image, la string, b image.Image, lb string) image.Image {
	const gap, labelH = 8, 16
	aw, ah := a.Bounds().Dx(), a.Bounds().Dy()
	bw, bh := b.Bounds().Dx(), b.Bounds().Dy()
	W := maxi(aw, bw)
	H := labelH + ah + gap + labelH + bh
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	white := color.RGBA{R: 230, G: 230, B: 235, A: 255}
	microfont.Draw(out, 4, 3, 1, la, white)
	ax := (W - aw) / 2
	draw.Draw(out, image.Rect(ax, labelH, ax+aw, labelH+ah), a, a.Bounds().Min, draw.Src)
	y := labelH + ah + gap
	microfont.Draw(out, 4, y+3, 1, lb, white)
	bx := (W - bw) / 2
	draw.Draw(out, image.Rect(bx, y+labelH, bx+bw, y+labelH+bh), b, b.Bounds().Min, draw.Src)
	return out
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
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
