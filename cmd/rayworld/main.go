// Command rayworld flies a ray-traced camera with a tvcp-ai brain and renders the
// result — neural directives -> camera -> real 3-D. By default the built-in
// reference brain drives it via game:"world"; -brain URL points at a live model
// (Ollama / OpenAI / Anthropic adapter); -ref uses the scripted director.
//
//	go run ./cmd/rayworld                       # reference brain, final frame
//	go run ./cmd/rayworld -gif fly.gif          # the whole flight as a GIF
//	go run ./cmd/rayworld -brain http://127.0.0.1:8092/v1/decide -gif fly.gif
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/raysource"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func who(ref bool, url string) string {
	switch {
	case ref:
		return "scripted director"
	case url != "":
		return "brain " + url
	default:
		return "built-in reference brain (game:world)"
	}
}

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai brain URL (default: built-in reference)")
	ref := flag.Bool("ref", false, "use the scripted director instead of a brain")
	ticks := flag.Int("ticks", 48, "number of camera ticks to fly")
	cols := flag.Int("w", 90, "width in terminal cells")
	rows := flag.Int("h", 45, "height in terminal cells")
	spp := flag.Int("spp", 1, "anti-aliasing samples per axis")
	mode := flag.String("mode", "auto", "terminal mode (auto|halfblock|sextant|octant|braille|perceptual|optimal)")
	gifPath := flag.String("gif", "", "render the whole flight to a GIF")
	flag.Parse()

	var dir raydir.Director
	switch {
	case *ref:
		dir = raydir.RefDirector{}
	case *brainURL != "":
		dir = &raydir.BrainDirector{B: brain.HTTPBrain{URL: *brainURL}}
	default:
		dir = &raydir.BrainDirector{B: brain.Local{}}
	}

	scene := raysource.DemoScene()
	init := raydir.CamState{Angle: 0, Radius: 6, Height: 2.4, Target: raytrace.Vec3{X: 0, Y: 1, Z: 0}}
	cams := raydir.Fly(dir, init, *ticks)

	if *gifPath != "" {
		g := &gif.GIF{}
		for _, cam := range cams {
			img := raytrace.Render(scene, cam, *cols*2, *rows*4, raytrace.Options{Samples: *spp})
			pal := image.NewPaletted(img.Bounds(), palette.Plan9)
			draw.FloydSteinberg.Draw(pal, img.Bounds(), img, image.Point{})
			g.Image = append(g.Image, pal)
			g.Delay = append(g.Delay, 6)
		}
		f, err := os.Create(*gifPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gif:", err)
			return
		}
		defer func() { _ = f.Close() }()
		_ = gif.EncodeAll(f, g)
		fmt.Printf("wrote %s (%d frames flown by %s)\n", *gifPath, len(cams), who(*ref, *brainURL))
		return
	}

	name := *mode
	if name == "" || name == "auto" {
		name = terminal.DetectCapability().BestBlitMode()
	}
	rm, _ := babe.ParseRenderMode(name)
	img := raytrace.Render(scene, cams[len(cams)-1], *cols*2, *rows*4, raytrace.Options{Samples: *spp})
	fmt.Print(babe.ImageToFrameMode(img, *cols, *rows, rm).Render())
	fmt.Printf("flew %d ticks via %s; final frame, mode=%s\n", len(cams), who(*ref, *brainURL), name)
}
