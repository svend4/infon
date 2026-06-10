// Command rayhard shows the integrator "renderer brain" (block L): on a caustic
// dream scene where the unidirectional path tracer is noisy, RenderBest probes the
// difficulty and switches to the bidirectional path tracer — BDPT, shipped and
// tested but previously reachable only from the ray3d demo. It renders the path
// tracer and the auto-selected integrator side by side at equal samples.
//
//	go run ./cmd/rayhard              # writes hard.png (path tracer | auto: bdpt)
//	go run ./cmd/rayhard -spp 96
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
		out = flag.String("out", "hard", "output basename")
		w   = flag.Int("w", 380, "panel width")
		h   = flag.Int("h", 280, "panel height")
		spp = flag.Int("spp", 48, "samples per pixel (both integrators)")
	)
	flag.Parse()

	// A caustic dream scene: a glass sphere and a diffuse one under a small, bright
	// light over a pale floor (dark sky), so the light-through-glass caustic is the
	// high-variance path that trips the unidirectional tracer.
	// Lit purely by the area light (black sky): both integrators see the same energy
	// so they converge, and BDPT — which connects light subpaths to the camera —
	// resolves the light-through-glass caustic with far less noise than the
	// unidirectional tracer. (This renderer's BDPT does not gather an environment
	// sky, so RenderBest only switches to it for area-lit scenes like this one.)
	scene := raydir.BuildScene(brain.SceneSpec{
		Light:  [3]float64{0, 10, 2},
		SkyTop: [3]float64{0, 0, 0}, SkyBot: [3]float64{0, 0, 0},
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: [3]float64{0.72, 0.72, 0.74}},
			{Y: 1.0, Z: 3, R: 1.0, Glass: 1.5},
			{X: -1.9, Y: 0.8, Z: 3.6, R: 0.8, Color: [3]float64{0.85, 0.3, 0.3}, Rough: 0.3},
			{Y: 7, Z: 3, R: 0.6, Emit: [3]float64{120, 120, 110}},
		},
	})
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.8, Z: -2.2}, Pitch: -0.4, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 3, NEE: true, MIS: true, Sobol: true}

	pt := raytrace.PathRender(scene, cam, *w, *h, opt)
	best, name := raytrace.RenderBest(scene, cam, *w, *h, opt)
	choice := raytrace.PickIntegrator(scene, cam, *w, *h, 0.065)
	fmt.Printf("probe noise=%.4f -> auto-selected %q (vs plain path tracer), %d spp\n", choice.Noise, name, *spp)

	sheet := montage([]panel{{pt, fmt.Sprintf("path tracer, %d spp", *spp)}, {best, fmt.Sprintf("auto: %s, %d spp", name, *spp)}})
	writePNG(*out+".png", sheet)
	fmt.Printf("wrote %s.png\n", *out)
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
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
