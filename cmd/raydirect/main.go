// Command raydirect shows meaning-driven direction in both directions (Block C). A
// hidden viewer has a taste; the LEARNING director cannot see it, so it starts with a
// neutral world and, round after round, watches an engagement signal (how long the
// viewer lingers — dwell — blended with mood) and climbs toward worlds that hold
// attention, converging on the viewer's taste. Then the READER runs the other way:
// given the finished, ray-traced world it recovers the six Q6 coordinates, the
// hexagram and the mood the scene carries — meaning read back out of a world.
//
// It ray-traces three worlds side by side (neutral start, what the director learned,
// the viewer's actual taste), prints the engagement climb as a sparkline, and reports
// the reader's read-back of the learned world.
//
//	go run ./cmd/raydirect -seed 3 -rounds 250
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out    = flag.String("out", "direct", "output basename")
		seed   = flag.Int64("seed", 3, "seed (viewer taste + director exploration)")
		taste  = flag.String("taste", "", "viewer's taste as a hexagram (6 bits); empty = random from seed")
		rounds = flag.Int("rounds", 250, "learning rounds")
		w      = flag.Int("w", 300, "panel width")
		h      = flag.Int("h", 220, "panel height")
		spp    = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()

	// the hidden viewer: a taste the director never sees directly.
	var pref raydir.SceneVector
	if hex, ok := raydir.ParseHexagram(*taste); ok {
		pref = raydir.VectorFromHexagram(hex)
	} else {
		rng := rand.New(rand.NewSource(*seed))
		for i := range pref {
			pref[i] = rng.Float64()
		}
	}
	viewer := raydir.ViewerModel{Pref: pref, Sigma: 0.3}

	// the learning director: climbs the engagement signal toward the viewer's taste.
	d := raydir.NewLearningDirector(*seed)
	curve := make([]float64, 0, *rounds+1)
	curve = append(curve, d.Engagement())
	for i := 0; i < *rounds; i++ {
		d.Round(viewer.Engagement)
		curve = append(curve, d.Engagement())
	}
	curve[0] = curve[1] // the first sample is -Inf until the neutral world is scored

	neutral := raydir.SceneVector{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	learned := d.Best()

	// the reader: recover meaning from the finished, authored world.
	learnedSpec := learned.SceneSpec()
	read := raydir.ReadScene(learnedSpec)

	fmt.Printf("viewer's hidden taste: hexagram %q (%06b), mood %q\n",
		pref.Hexagram().Name(), pref.Hexagram().Number(), pref.Mood())
	fmt.Printf("director, after %d rounds: hexagram %q (%06b), mood %q, engagement %.3f\n",
		*rounds, learned.Hexagram().Name(), learned.Hexagram().Number(), learned.Mood(), d.Engagement())
	fmt.Printf("  engagement climb %s\n", sparkline(curve))
	hit := "✗"
	if learned.Hexagram() == pref.Hexagram() {
		hit = "✓"
	}
	fmt.Printf("  learned hexagram matches the viewer's taste: %s\n", hit)
	fmt.Printf("reader on the learned world: hexagram %q (%06b), mood %q\n",
		read.Hexagram().Name(), read.Hexagram().Number(), read.Mood())
	rt := "✗"
	if read.Hexagram() == learned.Hexagram() {
		rt = "✓"
	}
	fmt.Printf("  reader recovers the director's hexagram (round trip): %s\n", rt)

	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	render := func(v raydir.SceneVector) image.Image {
		img := raytrace.PathRender(raydir.BuildScene(v.SceneSpec()), cam, *w, *h, opt)
		return raytrace.PostProcess(img, 1.0, 0.9, 0.4)
	}

	sheet := montage([]panel{
		{render(neutral), "neutral start"},
		{render(learned), fmt.Sprintf("director learned (%s)", learned.Mood())},
		{render(pref), fmt.Sprintf("viewer's taste (%s)", pref.Mood())},
	})
	writePNG(*out+".png", sheet)
	fmt.Printf("wrote %s.png\n", *out)
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
	return fmt.Sprintf("%s  [%.2f..%.2f]", b.String(), lo, hi)
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
