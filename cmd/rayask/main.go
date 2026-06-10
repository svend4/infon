// Command rayask asks a brain to author a world from a prompt and path-traces it.
// With BRAIN_URL set it routes to that tvcp-ai/1 brain (e.g. yijing_brain.py — the
// pro2 director); otherwise the built-in reference brain authors offline. It is the
// non-interactive way to see what a director makes of a prompt — and the showcase
// for yijing v2, whose continuous Q6 vector bends fog/palette/density/day-night.
//
//	go run ./cmd/rayask "a forest at night"
//	BRAIN_URL=http://127.0.0.1:8095/v1/decide go run ./cmd/rayask "a bright meadow"
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out = flag.String("out", "ask", "output basename")
		w   = flag.Int("w", 720, "render width")
		h   = flag.Int("h", 480, "render height")
		spp = flag.Int("spp", 80, "samples per pixel")
	)
	flag.Parse()
	prompt := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if prompt == "" {
		prompt = "a calm world of spheres"
	}

	var b brain.Brain = brain.Local{}
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b = brain.HTTPBrain{URL: url}
		fmt.Println("director:", url)
	} else {
		fmt.Println("director: built-in reference brain (set BRAIN_URL for a model)")
	}

	scene, spec, err := raydir.AuthorScene(b, prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "author:", err)
		os.Exit(1)
	}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{
		Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true,
	})
	writePNG(*out+".png", img)
	fmt.Printf("prompt %q -> %q (%d objects) -> %s.png\n", prompt, spec.Name, len(spec.Objects), *out)
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
