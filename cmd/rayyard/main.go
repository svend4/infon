// Command rayyard renders the shared world as a robot yard: several robots patrol
// loops with status beacons green->red, and the world is rendered over time into a
// contact sheet — proof that robots are first-class world entities. The same World
// is what rayexplore (-robots) walks and raymeet shares, so this is the multi-robot
// world of Part 1, observed.
//
//	go run ./cmd/rayyard                  # writes yard.png (a contact sheet over time)
//	go run ./cmd/rayyard -cols 3 -rows 2 -spp 64
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
		out  = flag.String("out", "yard", "output basename")
		w    = flag.Int("w", 360, "per-frame width")
		h    = flag.Int("h", 250, "per-frame height")
		cols = flag.Int("cols", 3, "contact-sheet columns")
		rows = flag.Int("rows", 2, "contact-sheet rows")
		spp  = flag.Int("spp", 48, "samples per pixel")
	)
	flag.Parse()

	world := raydir.NewWorld()
	world.SetTime(0.4) // a morning sun: shadows and warm light on the metal bodies
	// A yard of robots, each on a square patrol loop, status spread green -> red.
	centers := []raytrace.Vec3{
		{X: -4, Z: 6}, {X: 0, Z: 7}, {X: 4, Z: 6}, {X: -2, Z: 10}, {X: 3, Z: 11},
	}
	for i, c := range centers {
		loop := []raytrace.Vec3{c, c.Add(raytrace.Vec3{X: 3.5}), c.Add(raytrace.Vec3{X: 3.5, Z: 3.5}), c.Add(raytrace.Vec3{Z: 3.5})}
		status := float64(i) / float64(len(centers)-1)
		world.SpawnRobot(raydir.NewRobot(c, loop, status))
	}

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 4.2, Z: -1.5}, Pitch: -0.4, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 6, NEE: true, MIS: true, Sobol: true}

	frames := *cols * *rows
	const gap, labelH = 6, 14
	sheetW := *cols*(*w) + (*cols-1)*gap
	sheetH := *rows*(*h+labelH) + (*rows-1)*gap
	sheet := image.NewRGBA(image.Rect(0, 0, sheetW, sheetH))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 14, G: 14, B: 18, A: 255}}, image.Point{}, draw.Src)

	tsec := 0.0
	for f := 0; f < frames; f++ {
		for k := 0; k < 8; k++ { // ~0.8 s of motion between frames
			world.StepRobots(0.1)
			tsec += 0.1
		}
		img := raytrace.PathRender(world.Scene(), cam, *w, *h, opt)
		cx := (f % *cols) * (*w + gap)
		cy := (f / *cols) * (*h + labelH + gap)
		microfont.Draw(sheet, cx+3, cy+2, 1, fmt.Sprintf("t=%.1fs", tsec), color.RGBA{R: 220, G: 220, B: 230, A: 255})
		draw.Draw(sheet, image.Rect(cx, cy+labelH, cx+*w, cy+labelH+*h), img, img.Bounds().Min, draw.Src)
	}

	writePNG(*out+".png", sheet)
	fmt.Printf("rendered %d robots over %d frames -> %s.png\n", len(centers), frames, *out)
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
