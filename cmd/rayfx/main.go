// Command rayfx activates three renderers that shipped in pkg/raytrace fully
// tested but wired to no command (the "dormant code" of the audit): photon-mapped
// caustics, adaptive sampling, and temporal reprojection. Each -mode renders a
// before/after montage so the effect is visible.
//
//	go run ./cmd/rayfx -mode caustics    # focused light under glass (BuildCaustics)
//	go run ./cmd/rayfx -mode adaptive    # where the samples go (AdaptiveRender)
//	go run ./cmd/rayfx -mode temporal    # noise falls as frames accumulate (TemporalReprojector)
//
// Correctness of each renderer is covered by the existing pkg/raytrace tests
// (TestCausticsConcentrateUnderGlass, TestAdaptiveSpendsMoreOnNoise,
// TestTemporalReprojectionReducesNoiseStaticView); this command gives them a CLI.
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

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		mode    = flag.String("mode", "caustics", "caustics | adaptive | temporal")
		out     = flag.String("out", "", "output basename (default fx-<mode>)")
		w       = flag.Int("w", 420, "panel width")
		h       = flag.Int("h", 300, "panel height")
		photons = flag.Int("photons", 1200000, "caustic photons to shoot")
		frames  = flag.Int("frames", 24, "frames to accumulate (temporal)")
	)
	flag.Parse()
	base := *out
	if base == "" {
		base = "fx-" + *mode
	}

	var img image.Image
	switch *mode {
	case "caustics":
		img = caustics(*w, *h, *photons)
	case "adaptive":
		img = adaptive(*w, *h)
	case "temporal":
		img = temporal(*w, *h, *frames)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(1)
	}
	writePNG(base+".png", img)
	fmt.Printf("wrote %s.png\n", base)
}

// caustics: a glass sphere over a floor, lit from above. Without a photon map the
// floor below is a dull shadow; with BuildCaustics a bright focused caustic appears.
func caustics(w, h, photons int) image.Image {
	build := func() *raytrace.Scene {
		return raydir.BuildScene(brain.SceneSpec{
			Light:  [3]float64{0, 10, 2},
			SkyTop: [3]float64{0.04, 0.05, 0.08}, SkyBot: [3]float64{0.07, 0.08, 0.11},
			Objects: []brain.ObjSpec{
				{Kind: "plane", Color: [3]float64{0.6, 0.6, 0.62}},
				{Y: 1.0, Z: 2, R: 1.0, Glass: 1.5},
				{Y: 5, Z: 2, R: 0.5, Emit: [3]float64{70, 70, 66}},
			},
		})
	}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 3.0, Z: -2.6}, Pitch: -0.45, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: 96, MaxDepth: 6, Seed: 3, NEE: true, MIS: true, Sobol: true}

	plain := raytrace.PathRender(build(), cam, w, h, opt)

	lit := build()
	pm := raytrace.BuildCaustics(lit, photons, 0.05, 9)
	lit.Caustics = pm
	withC := raytrace.PathRender(lit, cam, w, h, opt)

	fmt.Printf("caustics: %d photons gathered\n", pm.Count())
	return montage([]panel{{plain, "no caustics"}, {withC, fmt.Sprintf("+ caustics (%d photons)", pm.Count())}})
}

// adaptive: AdaptiveRender returns per-pixel sample counts; the right panel maps
// them blue(low)->red(high), showing the budget flowing to the noisy regions.
func adaptive(w, h int) image.Image {
	scene := raydir.BuildScene(brain.SceneSpec{
		Light:  [3]float64{6, 9, -3},
		SkyTop: [3]float64{0.3, 0.4, 0.6}, SkyBot: [3]float64{0.7, 0.72, 0.78},
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: [3]float64{0.6, 0.6, 0.62}},
			{X: -1.2, Y: 1, Z: 2.5, R: 1, Color: [3]float64{0.9, 0.3, 0.3}, Rough: 0.3},
			{X: 1.2, Y: 1, Z: 2.5, R: 1, Color: [3]float64{0.95, 0.95, 0.97}, Reflect: 0.9},
			{Y: 0.7, Z: 1, R: 0.6, Glass: 1.5},
			{Y: 6, Z: -1, R: 0.7, Emit: [3]float64{20, 20, 19}},
		},
	})
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2, Z: -3}, Pitch: -0.12, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: 256, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	img, counts := raytrace.AdaptiveRender(scene, cam, w, h, opt, 0.04)

	total := 0
	maxc := 1
	for _, c := range counts {
		total += c
		if c > maxc {
			maxc = c
		}
	}
	avg := float64(total) / float64(len(counts))
	fmt.Printf("adaptive: %.0f avg samples/px (cap %d) — %.0f%% of a uniform pass\n",
		avg, opt.Samples, 100*avg/float64(opt.Samples))

	heat := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, c := range counts {
		t := float64(c) / float64(maxc)
		heat.SetRGBA(i%w, i/w, ramp(t))
	}
	return montage([]panel{{img, "adaptive render"}, {heat, fmt.Sprintf("samples/px (avg %.0f/%d)", avg, opt.Samples)}})
}

// temporal: a static view rendered at low spp is noisy; feeding successive frames
// through the TemporalReprojector accumulates them into a clean image.
func temporal(w, h, frames int) image.Image {
	build := func() *raytrace.Scene {
		return raydir.BuildScene(brain.SceneSpec{
			Light:  [3]float64{5, 9, -3},
			SkyTop: [3]float64{0.32, 0.42, 0.62}, SkyBot: [3]float64{0.72, 0.74, 0.8},
			Objects: []brain.ObjSpec{
				{Kind: "plane", Color: [3]float64{0.6, 0.6, 0.62}},
				{X: -1, Y: 1, Z: 2.5, R: 1, Color: [3]float64{0.85, 0.5, 0.3}, Rough: 0.4},
				{X: 1.2, Y: 0.8, Z: 2, R: 0.8, Glass: 1.5},
				{Y: 6, Z: -1, R: 0.6, Emit: [3]float64{22, 22, 21}},
			},
		})
	}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2, Z: -3}, Pitch: -0.12, FOV: math.Pi / 3}
	low := raytrace.PathOptions{Samples: 4, MaxDepth: 5, Seed: 1, NEE: true, MIS: true, Sobol: true}

	single := raytrace.PathRender(build(), cam, w, h, low)

	tr := raytrace.NewTemporalReprojector(w, h, 0.1)
	var acc image.Image
	for i := 0; i < frames; i++ {
		opt := low
		opt.Seed = uint64(i + 1)
		acc = tr.Frame(build(), cam, opt)
	}
	fmt.Printf("temporal: %d frames at %d spp accumulated\n", frames, low.Samples)
	return montage([]panel{
		{single, fmt.Sprintf("1 frame, %d spp", low.Samples)},
		{acc, fmt.Sprintf("%d frames reprojected", frames)},
	})
}

// ramp maps 0..1 to a blue->cyan->yellow->red heat colour.
func ramp(t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	r := math.Max(0, math.Min(1, 1.5*t-0.4))
	g := math.Max(0, math.Min(1, 1-2*math.Abs(t-0.5)))
	b := math.Max(0, math.Min(1, 1-1.6*t))
	return color.RGBA{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
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
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
