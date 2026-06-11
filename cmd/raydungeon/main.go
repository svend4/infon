// Command raydungeon plays the Q6 maze as a LIVING dungeon (Block E: alife × rogue).
// Every room on the solved route holds its own ecosystem — fed by a climate seeded
// from the room's hexagram and a fertility set by the hexagram (lush when sunny and
// dense, barren when foggy and sunless) — and the rooms are coupled: each tick some
// creatures migrate along the route's edges. So a barren room is kept alive by
// immigration from lush neighbours (source-sink dynamics) while it would crash alone.
// It runs the metapopulation, ray-traces the living arenas of the rooms along the
// route, and reports each room's fertility, population and the migration between them.
//
//	go run ./cmd/raydungeon -seed 16 -walls 0.6 -ticks 200
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
	"strings"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out   = flag.String("out", "dungeon", "output basename")
		seed  = flag.Int64("seed", 16, "maze seed")
		walls = flag.Float64("walls", 0.6, "fraction of rooms collapsed (0..0.9)")
		ticks = flag.Int("ticks", 200, "ticks to simulate")
		migr  = flag.Float64("migr", 0.06, "per-edge migration fraction each tick")
		gw    = flag.Int("gw", 10, "room grid width")
		gh    = flag.Int("gh", 10, "room grid height")
		pop0  = flag.Int("n", 22, "starting population per room")
		rooms = flag.Int("rooms", 5, "rooms to ray-trace along the route")
		w     = flag.Int("w", 240, "arena thumbnail width")
		h     = flag.Int("h", 180, "arena thumbnail height")
		spp   = flag.Int("spp", 40, "samples per pixel")
	)
	flag.Parse()

	q := raydir.NewQuest(*seed, *walls)
	route, ok := q.Solve()
	if !ok {
		fmt.Fprintln(os.Stderr, "no route")
		os.Exit(1)
	}
	d := raydir.NewLivingDungeon(route, *gw, *gh, *pop0, *migr, *seed)
	start := d.Populations()

	totalCurve := make([]float64, 0, *ticks+1)
	totalCurve = append(totalCurve, sum(start))
	for t := 0; t < *ticks; t++ {
		d.Step()
		totalCurve = append(totalCurve, sum(d.Populations()))
	}
	end := d.Populations()

	fmt.Printf("living dungeon seed %d — %d rooms on the route, %d ticks, migration %.0f%%/edge/tick\n",
		*seed, len(route), *ticks, *migr*100)
	for i, hx := range route {
		fmt.Printf("  room %2d %06b %-20s  fertility %.2f  pop %2d -> %2d\n",
			i, hx.Number(), hx.Name(), d.Richness(i), start[i], end[i])
	}
	fmt.Printf("  total population %s\n", sparkline(totalCurve))
	fmt.Printf("  migrants moved across edges: %d\n", d.Migrated())

	// ray-trace the living arenas of rooms spread along the route.
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 4, Seed: 5, NEE: true, MIS: true, Sobol: true}
	idx := pickRooms(len(route), *rooms)
	var panels []panel
	for _, i := range idx {
		scene, cam := arenaScene(d.Room(i))
		img := raytrace.PostProcess(raytrace.PathRender(scene, cam, *w, *h, opt), 1.0, 0.9, 0.4)
		panels = append(panels, panel{img, fmt.Sprintf("%06b rich%.1f pop%d", route[i].Number(), d.Richness(i), end[i])})
	}
	writePNG(*out+".png", montage(panels))
	fmt.Printf("wrote %s.png\n", *out)
}

func sum(xs []int) float64 {
	s := 0.0
	for _, x := range xs {
		s += float64(x)
	}
	return s
}

// arenaScene builds an overhead, ray-traceable view of one room's living world.
func arenaScene(e *raydir.Ecosystem) (*raytrace.Scene, raytrace.Camera) {
	sw, sd := e.Span()
	at := raytrace.Vec3{X: -sw / 2, Y: 0, Z: -sd / 2}
	diag := math.Hypot(sw, sd)
	s := &raytrace.Scene{
		Light:     raytrace.Vec3{X: sw, Y: diag * 1.4, Z: -diag},
		LightInt:  1.5,
		Ambient:   0.28,
		SkyTop:    raytrace.Vec3{X: 0.22, Y: 0.36, Z: 0.62},
		SkyBottom: raytrace.Vec3{X: 0.62, Y: 0.74, Z: 0.9},
	}
	s.Objects = append(s.Objects, raytrace.Plane{
		Y: 0, Size: diag * 4,
		C1:  raytrace.Vec3{X: 0.10, Y: 0.12, Z: 0.10},
		C2:  raytrace.Vec3{X: 0.07, Y: 0.09, Z: 0.07},
		Mat: raytrace.Material{Rough: 0.95},
	})
	s.Objects = append(s.Objects, e.Objects(at)...)
	s.BuildBVH()
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: diag * 0.82, Z: -diag * 0.6}, Pitch: -0.98, FOV: math.Pi / 2.3}
	return s, cam
}

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

func sparkline(xs []float64) string {
	const ramp = " ▁▂▃▄▅▆▇█"
	lo, hi := xs[0], xs[0]
	for _, x := range xs {
		lo, hi = math.Min(lo, x), math.Max(hi, x)
	}
	rng := hi - lo
	if rng == 0 {
		rng = 1
	}
	const cols = 56
	stepF := math.Max(1, float64(len(xs))/cols)
	var b strings.Builder
	for f := 0.0; int(f) < len(xs); f += stepF {
		lvl := int((xs[int(f)] - lo) / rng * 8)
		if lvl < 0 {
			lvl = 0
		}
		if lvl > 8 {
			lvl = 8
		}
		b.WriteRune([]rune(ramp)[lvl])
	}
	return fmt.Sprintf("%s  [%.0f..%.0f]", b.String(), lo, hi)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	pw := panels[0].img.Bounds().Dx()
	ph := panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	W := len(panels)*pw + (len(panels)-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, ph+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		x := i * (pw + gap)
		draw.Draw(out, image.Rect(x, labelH, x+pw, labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, 3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
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
