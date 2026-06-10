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

	"github.com/svend4/infon/pkg/fleet"
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
		live = flag.Bool("live", false, "drive each beacon from the monitoring engine — one robot overheats and reddens on its own")
	)
	flag.Parse()

	world := raydir.NewWorld()
	world.SetTime(0.4) // a morning sun: shadows and warm light on the metal bodies
	// A yard of robots, each on a square patrol loop.
	centers := []raytrace.Vec3{
		{X: -4, Z: 6}, {X: 0, Z: 7}, {X: 4, Z: 6}, {X: -2, Z: 10}, {X: 3, Z: 11},
	}
	var mon *fleet.RobotMonitor
	if *live {
		mon = fleet.NewRobotMonitor()
	}
	for i, c := range centers {
		loop := []raytrace.Vec3{c, c.Add(raytrace.Vec3{X: 3.5}), c.Add(raytrace.Vec3{X: 3.5, Z: 3.5}), c.Add(raytrace.Vec3{Z: 3.5})}
		// Fixed status spread green->red when not live; the monitor overwrites it live.
		status := float64(i) / float64(len(centers)-1)
		rb := raydir.NewRobot(c, loop, status)
		world.SpawnRobot(rb)
		if *live {
			idx := i
			mon.Add(rb, fmt.Sprintf("amr-%d", i+1), func(t float64) []fleet.Signal {
				thermal := 0.12 + 0.03*float64(idx) // a steady, slightly-warm baseline
				if idx == 3 {                       // one robot's motor overheats over time
					thermal = 0.13 + 0.16*t
				}
				return []fleet.Signal{
					{Name: "thermal", Value: thermal, Weight: 1},
					{Name: "vibration", Value: 0.15, Weight: 1},
					{Name: "battery", Value: 0.3, Weight: 0.6},
				}
			})
		}
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
			if *live {
				mon.Step(0.1) // steps robots AND drives beacons from the engine
			} else {
				world.StepRobots(0.1)
			}
			tsec += 0.1
		}
		img := raytrace.PathRender(world.Scene(), cam, *w, *h, opt)
		img = raytrace.PostProcess(img, 1.0, 0.75, 0.6) // bloom the status beacons into a glow
		cx := (f % *cols) * (*w + gap)
		cy := (f / *cols) * (*h + labelH + gap)
		microfont.Draw(sheet, cx+3, cy+2, 1, fmt.Sprintf("t=%.1fs", tsec), color.RGBA{R: 220, G: 220, B: 230, A: 255})
		draw.Draw(sheet, image.Rect(cx, cy+labelH, cx+*w, cy+labelH+*h), img, img.Bounds().Min, draw.Src)
		if *live {
			for _, a := range mon.Assessments() {
				if a.Level != fleet.LevelOK {
					fmt.Printf("  t=%.1fs %s\n", tsec, a.Cue)
				}
			}
		}
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
