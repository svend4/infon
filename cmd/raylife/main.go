// Command raylife runs a small artificial-life world that lives WITHOUT a player
// (Block B: alife). A population of creatures forages a grid of food, gains energy by
// eating, spends it moving, reproduces when fed and dies when starved — so the
// population booms, crashes and migrates on its own. Food regrows at a rate set by
// the HexCA climate (rain zones feed fast, snow slow), so the weather automaton drives
// the ecology. It steps the world for a while, ray-traces overhead snapshots into a
// contact sheet, and prints population and standing-food curves as sparklines.
//
//	go run ./cmd/raylife -seed 7 -ticks 240
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
		out   = flag.String("out", "life", "output basename")
		seed  = flag.Int64("seed", 7, "world seed (climate + population)")
		gw    = flag.Int("gw", 12, "grid width (cells)")
		gh    = flag.Int("gh", 12, "grid height (cells)")
		start = flag.Int("n", 28, "starting population")
		ticks = flag.Int("ticks", 240, "ticks to simulate")
		snaps = flag.Int("snaps", 4, "overhead snapshots in the contact sheet")
		w     = flag.Int("w", 300, "panel width")
		h     = flag.Int("h", 220, "panel height")
		spp   = flag.Int("spp", 40, "samples per pixel")
	)
	flag.Parse()

	clim := raydir.NewClimate(*seed)
	eco := raydir.NewEcosystem(*gw, *gh, *start, clim, *seed)
	sw, sd := eco.Span()
	at := raytrace.Vec3{X: -sw / 2, Y: 0, Z: -sd / 2} // centre the arena on the origin

	// an overhead camera that frames the whole arena, looking down and forward (an
	// oblique top-down — straight down with a bright sky blows the exposure).
	diag := math.Hypot(sw, sd)
	cam := raytrace.Camera{
		Pos:   raytrace.Vec3{X: 0, Y: diag * 0.82, Z: -diag * 0.6},
		Pitch: -0.98, FOV: math.Pi / 2.3,
	}

	scene := func() *raytrace.Scene {
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
		s.Objects = append(s.Objects, eco.Objects(at)...)
		s.BuildBVH()
		return s
	}

	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 4, Seed: 5, NEE: true, MIS: true, Sobol: true}

	// step the world, recording the curves and grabbing snapshots at even intervals.
	popCurve := make([]float64, 0, *ticks+1)
	foodCurve := make([]float64, 0, *ticks+1)
	record := func() {
		popCurve = append(popCurve, float64(eco.Population()))
		foodCurve = append(foodCurve, eco.TotalFood())
	}
	record()
	var panels []panel
	snapAt := map[int]bool{}
	for i := 0; i < *snaps; i++ {
		snapAt[i*(*ticks)/maxi(*snaps-1, 1)] = true // first .. last tick, evenly spaced
	}
	for t := 0; t <= *ticks; t++ {
		if snapAt[t] {
			img := raytrace.PostProcess(raytrace.PathRender(scene(), cam, *w, *h, opt), 1.0, 0.9, 0.4)
			panels = append(panels, panel{img, fmt.Sprintf("tick %d  pop %d", t, eco.Population())})
		}
		if t < *ticks {
			eco.Step()
		}
		record()
	}

	writePNG(*out+".png", montage(panels))

	peakPop, peakAt := peak(popCurve)
	fmt.Printf("alife world seed %d — %d×%d grid, %d ticks, climate=%s\n", *seed, *gw, *gh, *ticks, clim.Kind())
	fmt.Printf("  population: start %d  peak %d (tick %d)  end %d\n", *start, peakPop, peakAt, int(popCurve[len(popCurve)-1]))
	fmt.Printf("  pop  %s\n", sparkline(popCurve))
	fmt.Printf("  food %s\n", sparkline(foodCurve))
	fmt.Printf("wrote %s.png\n", *out)
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func peak(xs []float64) (int, int) {
	best, at := 0.0, 0
	for i, x := range xs {
		if x > best {
			best, at = x, i
		}
	}
	return int(best), at
}

// sparkline renders a series as block characters scaled to its own range.
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
	// subsample to ~60 columns so the line stays terminal-width
	const cols = 60
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
	return fmt.Sprintf("%s  [%g..%g]", b.String(), lo, hi)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	if len(panels) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	pw := panels[0].img.Bounds().Dx()
	ph := panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	W := len(panels)*pw + (len(panels)-1)*gap
	H := ph + labelH
	out := image.NewRGBA(image.Rect(0, 0, W, H))
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
