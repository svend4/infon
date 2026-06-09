// Command rayexplore is the experience: you walk, in the terminal, through a 3-D
// world the AI director dreams up and extends on the fly. The brain authors each
// region as a compact scene description (game:rayscene) - meaning, not pixels -
// which is ray-traced locally, so the world unfolds for a few bytes per chunk as
// you explore. As you move forward, new regions are authored ahead of you.
//
// Brain: set BRAIN_URL to a tvcp-ai/1 endpoint (Ollama/cloud) for a real director;
// otherwise the built-in reference brain composes scenes offline.
//
//	go run ./cmd/rayexplore                         # walk a reference-authored world
//	BRAIN_URL=http://localhost:11434/... go run ./cmd/rayexplore -prompt "a glass city"
//
// Controls (type, then Enter; combine letters): w/s walk, a/d strafe, q/e turn,
// r/f look, g grow a new region now, p toggle path tracer, x quit.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"strings"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	prompt := flag.String("prompt", "a calm world of spheres", "seed prompt for the AI director")
	modeFlag := flag.String("mode", "auto", "terminal render mode (auto|halfblock|sextant|braille|ascii|...)")
	pathT := flag.Bool("path", false, "use the path tracer (prettier, slower) instead of the raster preview")
	cols := flag.Int("w", 80, "width in terminal cells")
	rows := flag.Int("h", 38, "height in terminal cells")
	flag.Parse()
	pxW, pxH := *cols*2, *rows*4

	// Pick the director: a real tvcp-ai/1 brain if BRAIN_URL is set, else the
	// built-in reference author (offline, deterministic).
	var b brain.Brain = brain.Local{}
	who := "reference (offline)"
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b = brain.HTTPBrain{URL: url}
		who = url
	}

	prompts := []string{*prompt, "a gold sphere", "a scene at night", "glass and metal", "a calm world"}
	pi := 0
	nextPrompt := func() string { p := prompts[pi%len(prompts)]; pi++; return p }

	world := raydir.NewWorld()
	frontZ := 10.0
	if _, err := world.Grow(b, nextPrompt(), raytrace.Vec3{X: 0, Y: 0, Z: frontZ}); err != nil {
		fmt.Fprintln(os.Stderr, "director failed to author the first region:", err)
	}
	cam := raydir.FlyCam{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: 0}, Yaw: 0, Pitch: -0.08, FOV: math.Pi / 3}

	mode := *modeFlag
	if mode == "" || mode == "auto" {
		mode = terminal.DetectCapability().BestBlitMode()
	}
	rm, _ := babe.ParseRenderMode(mode)
	dr := terminal.NewDiffRenderer()
	scene := world.Scene()
	pathOpt := raytrace.PathOptions{Samples: 4, MaxDepth: 5, Seed: 1, NEE: true, MIS: true, Sobol: true}

	// grow authors a new region ahead and rebuilds the scene.
	grow := func() {
		jitter := math.Sin(float64(world.Chunks())*1.7) * 2.5
		frontZ += 14
		n, err := world.Grow(b, nextPrompt(), raytrace.Vec3{X: jitter, Y: 0, Z: frontZ})
		if err != nil {
			fmt.Fprintln(os.Stderr, "grow failed:", err)
			return
		}
		_ = n
		scene = world.Scene()
	}

	render := func() {
		c := cam.Camera()
		var im image.Image
		if *pathT {
			im = raytrace.PathRender(scene, c, pxW, pxH, pathOpt)
		} else {
			im = raytrace.Render(scene, c, pxW, pxH, raytrace.Options{Samples: 1})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		fmt.Printf("\n[director: %s | chunks:%d props:%d | pos (%.1f,%.1f,%.1f) | w/s a/d q/e r/f  g=grow p=path x=quit] ",
			who, world.Chunks(), world.Props(), cam.Pos.X, cam.Pos.Y, cam.Pos.Z)
	}

	fmt.Printf("rayexplore — walking a world authored by %s. Each region is shipped as a tiny scene description and ray-traced locally.\n", who)
	render()

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		for _, ch := range strings.ToLower(strings.TrimSpace(sc.Text())) {
			switch ch {
			case 'w':
				cam.Walk(1.2, 0)
			case 's':
				cam.Walk(-1.2, 0)
			case 'a':
				cam.Walk(0, -1.2)
			case 'd':
				cam.Walk(0, 1.2)
			case 'q':
				cam.Turn(-0.2, 0)
			case 'e':
				cam.Turn(0.2, 0)
			case 'r':
				cam.Turn(0, 0.08)
			case 'f':
				cam.Turn(0, -0.08)
			case 'g':
				grow()
			case 'p':
				*pathT = !*pathT
			case 'x':
				return
			}
		}
		// the world keeps unfolding ahead as you advance.
		if cam.Pos.Z > frontZ-12 {
			grow()
		}
		render()
	}
}
