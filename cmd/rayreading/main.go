// Command rayreading casts an I-Ching reading and renders it as a world
// transition: the primary hexagram's world dissolves into voxel blocks and the
// relating hexagram's world forms from them (raydir.Reading + raytrace.Materialize)
// — the oracle's narrative arc A -> B made into a moving image, and a deeper bridge
// to pro2 than the static Gray walk. A stable reading (no changing lines) just
// pulses in place.
//
//	go run ./cmd/rayreading -seed 7
//	BRAIN_URL=http://127.0.0.1:8095/v1/decide go run ./cmd/rayreading -seed 7
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
		out  = flag.String("out", "reading", "output basename")
		seed = flag.Int64("seed", 7, "reading seed (the cast)")
		w    = flag.Int("w", 300, "per-frame width")
		h    = flag.Int("h", 200, "per-frame height")
		cols = flag.Int("cols", 3, "frames across")
		rows = flag.Int("rows", 2, "frames down")
		spp  = flag.Int("spp", 48, "samples per pixel")
	)
	flag.Parse()

	reading := raydir.CastReading(*seed)
	fmt.Printf("reading (seed %d): %s\n", *seed, reading.String())

	var b brain.Brain = brain.Local{}
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b = brain.HTTPBrain{URL: url}
	}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}

	render := func(hx raydir.Hexagram) image.Image {
		scene, _, err := raydir.AuthorScene(b, hx.Prompt())
		if err != nil {
			fmt.Fprintln(os.Stderr, "author:", err)
			os.Exit(1)
		}
		return raytrace.PathRender(scene, cam, *w, *h, opt)
	}
	imgA := render(reading.Primary)
	imgB := imgA
	if !reading.Stable() {
		imgB = render(reading.Relating())
	}

	// the transition: A dissolves to blocks (t:0->0.5), B forms from blocks (0.5->1).
	frames := *cols * *rows
	const gap, labelH = 6, 14
	sheetW := *cols*(*w) + (*cols-1)*gap
	sheetH := *rows*(*h+labelH) + (*rows-1)*gap
	sheet := image.NewRGBA(image.Rect(0, 0, sheetW, sheetH))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 14, G: 14, B: 18, A: 255}}, image.Point{}, draw.Src)
	for f := 0; f < frames; f++ {
		t := 0.0
		if frames > 1 {
			t = float64(f) / float64(frames-1)
		}
		var fr image.Image
		if t < 0.5 {
			fr = raytrace.Materialize(imgA, 1-2*t) // sharp A -> blocks
		} else {
			fr = raytrace.Materialize(imgB, 2*t-1) // blocks -> sharp B
		}
		cx := (f % *cols) * (*w + gap)
		cy := (f / *cols) * (*h + labelH + gap)
		microfont.Draw(sheet, cx+3, cy+2, 1, fmt.Sprintf("t=%.2f", t), color.RGBA{R: 220, G: 220, B: 230, A: 255})
		draw.Draw(sheet, image.Rect(cx, cy+labelH, cx+*w, cy+labelH+*h), fr, fr.Bounds().Min, draw.Src)
	}
	writePNG(*out+".png", sheet)
	fmt.Printf("wrote %s.png (%s)\n", *out, map[bool]string{true: "stable pulse", false: "A -> B transition"}[reading.Stable()])
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
