// Command rayagent puts the robots under a tvcp-ai/1 brain: whenever a robot is
// ready for a new destination the brain picks one of its stations, given the
// robot's position and status. With ai/adapters/equipment_brain.py a hot robot is
// dispatched back to base to cool down while the healthy ones keep traversing the
// yard — brain-controlled robots in the shared world (Part 1). Offline (no
// BRAIN_URL) it falls back to a round-robin patrol, so it always runs.
//
//	go run ./cmd/rayagent                                   # offline (round-robin)
//	BRAIN_URL=http://127.0.0.1:8096/v1/decide go run ./cmd/rayagent   # brain dispatch
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
	"sort"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/fleet"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out  = flag.String("out", "agent", "output basename")
		w    = flag.Int("w", 360, "per-frame width")
		h    = flag.Int("h", 240, "per-frame height")
		cols = flag.Int("cols", 3, "contact-sheet columns")
		rows = flag.Int("rows", 2, "contact-sheet rows")
		spp  = flag.Int("spp", 48, "samples per pixel")
	)
	flag.Parse()

	world := raydir.NewWorld()
	world.SetTime(0.4)

	// stations: a base at the front, three work docks spread across the yard.
	base := raytrace.Vec3{X: 0, Z: 3}
	stations := []raytrace.Vec3{base, {X: -5, Z: 9}, {X: 5, Z: 9}, {X: 0, Z: 13}}
	names := []string{"base", "dockA", "dockB", "field"}

	var b brain.Brain
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b = brain.HTTPBrain{URL: url}
		fmt.Println("dispatching via brain:", url)
	} else {
		fmt.Println("no BRAIN_URL — round-robin fallback")
	}
	driver := fleet.NewBrainDriver(b)

	statuses := []float64{0.12, 0.35, 0.5, 0.88} // the last robot is faulty (hot)
	for i, st := range statuses {
		start := base.Add(raytrace.Vec3{X: float64(i)*1.6 - 2.4})
		rb := raydir.NewRobot(start, nil, st) // goal-driven (no patrol loop)
		world.SpawnRobot(rb)
		stt := st
		driver.Add(rb, fmt.Sprintf("amr-%d", i+1), stations, names, func() float64 { return stt })
	}

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 5.5, Z: -2.5}, Pitch: -0.5, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 6, NEE: true, MIS: true, Sobol: true}

	frames := *cols * *rows
	const gap, labelH = 6, 14
	sheetW := *cols*(*w) + (*cols-1)*gap
	sheetH := *rows*(*h+labelH) + (*rows-1)*gap
	sheet := image.NewRGBA(image.Rect(0, 0, sheetW, sheetH))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 14, G: 14, B: 18, A: 255}}, image.Point{}, draw.Src)

	tsec := 0.0
	for f := 0; f < frames; f++ {
		for k := 0; k < 9; k++ {
			driver.Step(0.1)
			tsec += 0.1
		}
		img := raytrace.PostProcess(raytrace.PathRender(world.Scene(), cam, *w, *h, opt), 1.0, 0.75, 0.6)
		cx := (f % *cols) * (*w + gap)
		cy := (f / *cols) * (*h + labelH + gap)
		microfont.Draw(sheet, cx+3, cy+2, 1, fmt.Sprintf("t=%.1fs", tsec), color.RGBA{R: 220, G: 220, B: 230, A: 255})
		draw.Draw(sheet, image.Rect(cx, cy+labelH, cx+*w, cy+labelH+*h), img, img.Bounds().Min, draw.Src)
	}

	// Report the brain's final dispatch per robot.
	dec := driver.Decisions()
	units := make([]string, 0, len(dec))
	for u := range dec {
		units = append(units, u)
	}
	sort.Strings(units)
	fmt.Println("dispatch:")
	for _, u := range units {
		fmt.Printf("  %s -> %s\n", u, dec[u])
	}

	writePNG(*out+".png", sheet)
	fmt.Printf("wrote %s.png\n", *out)
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
