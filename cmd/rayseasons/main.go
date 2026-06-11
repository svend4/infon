// Command rayseasons runs a living world through the turning year (Block L: seasons).
// Its weather both cycles (wet and dry) and MOVES (a rain front sweeping across the
// land), so the food follows the seasons and the creatures migrate after the rain —
// booming in the wet, thinning in the dry. It ray-traces overhead snapshots across the
// seasons (the green food and the creatures cluster wherever the rain currently falls,
// and that band drifts from snapshot to snapshot) and prints the population curve.
//
//	go run ./cmd/rayseasons -years 4 -year 120
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
		out   = flag.String("out", "seasons", "output basename")
		seed  = flag.Int64("seed", 7, "world seed")
		gw    = flag.Int("gw", 14, "grid width")
		gh    = flag.Int("gh", 14, "grid height")
		start = flag.Int("n", 30, "starting population")
		year  = flag.Int("year", 120, "ticks per year")
		years = flag.Float64("years", 4, "years to simulate")
		snaps = flag.Int("snaps", 6, "snapshots across the run")
		w     = flag.Int("w", 240, "panel width")
		h     = flag.Int("h", 180, "panel height")
		spp   = flag.Int("spp", 36, "samples per pixel")
	)
	flag.Parse()

	sw := raydir.NewSeasonalWorld(*gw, *gh, *start, *seed, *year)
	ticks := int(*years * float64(*year))
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 4, Seed: 5, NEE: true, MIS: true, Sobol: true}

	snapAt := map[int]bool{}
	for i := 0; i < *snaps; i++ {
		snapAt[i*ticks/maxi(*snaps-1, 1)] = true
	}
	popCurve := make([]float64, 0, ticks+1)
	var panels []panel
	for t := 0; t <= ticks; t++ {
		popCurve = append(popCurve, float64(sw.Population()))
		if snapAt[t] {
			scene, cam := arenaScene(sw.Eco())
			img := raytrace.PostProcess(raytrace.PathRender(scene, cam, *w, *h, opt), 1.0, 0.9, 0.4)
			panels = append(panels, panel{img, fmt.Sprintf("y%d %s wet%.0f%% pop%d",
				t / *year, sw.SeasonName(), sw.Wetness()*100, sw.Population())})
		}
		if t < ticks {
			sw.Step()
		}
	}

	writePNG(*out+".png", montage(panels))
	peak, peakAt := 0.0, 0
	for i, p := range popCurve {
		if p > peak {
			peak, peakAt = p, i
		}
	}
	fmt.Printf("seasonal world seed %d — %.0f years of %d ticks, %d×%d\n", *seed, *years, *year, *gw, *gh)
	fmt.Printf("  population: start %d  peak %d (tick %d, year %d %s)  end %d\n",
		*start, int(peak), peakAt, peakAt / *year, seasonAtTick(peakAt, *year), int(popCurve[len(popCurve)-1]))
	fmt.Printf("  pop %s\n", sparkline(popCurve))
	fmt.Printf("wrote %s.png\n", *out)
}

// seasonAtTick names the season at a tick (for the report).
func seasonAtTick(tick, year int) string {
	switch int(math.Mod(float64(tick)/float64(year), 1) * 4) {
	case 0:
		return "spring"
	case 1:
		return "summer"
	case 2:
		return "autumn"
	default:
		return "winter"
	}
}

// arenaScene builds an overhead, ray-traceable view of the living world.
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

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	return fmt.Sprintf("%s  [%.0f..%.0f]", b.String(), lo, hi)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	if len(panels) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	pw, ph := panels[0].img.Bounds().Dx(), panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	cols := (len(panels) + 1) / 2
	if len(panels) <= 3 {
		cols = len(panels)
	}
	rows := (len(panels) + cols - 1) / cols
	W := cols*pw + (cols-1)*gap
	H := rows*(ph+labelH) + (rows-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		r, c := i/cols, i%cols
		x := c * (pw + gap)
		y := r * (ph + labelH + gap)
		draw.Draw(out, image.Rect(x, y+labelH, x+pw, y+labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, y+3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
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
