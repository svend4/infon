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
//	go run ./cmd/rayexplore -path -grade            # path-traced, cinematically graded
//	go run ./cmd/rayexplore -sound                  # hear the world (procedural ambient)
//	BRAIN_URL=http://localhost:11434/... go run ./cmd/rayexplore -prompt "a glass city"
//
// Controls (type, then Enter; combine letters): w/s walk, a/d strafe, q/e turn,
// r/f look, g grow a new region now, t step time of day, p toggle path tracer,
// x quit. In path mode, press Enter (no movement) to let the view refine (it
// converges to a clean render while you hold still); -grade adds bloom/vignette/AgX.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/svend4/infon/internal/audio"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

// startSoundscape plays the world's procedural ambient (a guarded snapshot of
// features the render loop keeps fresh), or falls back silently with no device.
func startSoundscape(mu *sync.Mutex, feat *raydir.AmbientFeatures) {
	pl, err := audio.NewDefaultPlayback()
	if err != nil || pl.Open() != nil {
		fmt.Fprintln(os.Stderr, "sound: no audio device available; continuing silent")
		return
	}
	rate := audio.DefaultFormat().SampleRate
	frame := rate / 50 // 20ms
	go func() {
		t := 0.0
		tk := time.NewTicker(20 * time.Millisecond)
		defer tk.Stop()
		for range tk.C {
			mu.Lock()
			f := *feat
			mu.Unlock()
			_, _ = pl.Write(raydir.AmbientFrame(f, rate, t, frame))
			t += float64(frame) / float64(rate)
		}
	}()
}

func main() {
	prompt := flag.String("prompt", "a calm world of spheres", "seed prompt for the AI director")
	modeFlag := flag.String("mode", "auto", "terminal render mode (auto|halfblock|sextant|braille|ascii|...)")
	pathT := flag.Bool("path", false, "use the path tracer (prettier, slower) instead of the raster preview")
	grade := flag.Bool("grade", false, "post: bloom + vignette + AgX tone map for a cinematic frame")
	clouds := flag.Bool("clouds", false, "volumetric cloud bank (path tracer; costly)")
	sound := flag.Bool("sound", false, "play a procedural soundscape of the world (needs an audio device)")
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
	dayT := 0.32     // time of day; the 't' key steps it through dawn/noon/dusk/night
	showMap := false // overlay the minimap of named places (toggle with 'm')
	world.SetTime(dayT)
	world.SetClouds(*clouds)
	scene := world.Scene()
	pathOpt := raytrace.PathOptions{Samples: 4, MaxDepth: 5, Seed: 1, NEE: true, MIS: true, Sobol: true}
	// progressive refinement: stand still (press Enter) and the path-traced view
	// keeps sharpening toward a clean render; moving or growing restarts it.
	refiner := raydir.NewRefiner(pxW, pxH, 4, 256, pathOpt)

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
		refiner.Reset() // the scene changed
	}

	// optional procedural soundscape: synthesise the world's ambient locally and
	// play it (graceful fallback when there's no audio device).
	var featMu sync.Mutex
	feat := world.Ambient()
	if *sound {
		startSoundscape(&featMu, &feat)
	}

	start := time.Now()
	render := func() {
		world.SetAnimTime(time.Since(start).Seconds()) // keep the world alive
		featMu.Lock()
		feat = world.Ambient()
		featMu.Unlock()
		c := cam.Camera()
		var im image.Image
		spp := 1
		if *pathT {
			if world.HasAnimated() {
				refiner.Reset() // a moving world can't accumulate; render fresh
			}
			im = refiner.Frame(scene, c)
			spp = refiner.Samples()
		} else {
			im = raytrace.Render(scene, c, pxW, pxH, raytrace.Options{Samples: 1})
		}
		if *grade {
			im = raytrace.Grade(im, raytrace.GradeOptions{BloomThresh: 1.0, BloomStrength: 0.4, Vignette: 0.35, AgX: true})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		if showMap {
			fmt.Print("\n" + raydir.Minimap(world.Landmarks(), nil, cam.Pos, *cols, *rows/2))
		}
		mins := int((dayT - math.Floor(dayT)) * 24 * 60)
		fmt.Printf("\n[director: %s | chunks:%d props:%d | 🕓%02d:%02d spp:%d | pos (%.1f,%.1f,%.1f) | w/s a/d q/e r/f g=grow t=time p=path m=map Enter=refine x=quit] ",
			who, world.Chunks(), world.Props(), mins/60, mins%60, spp, cam.Pos.X, cam.Pos.Y, cam.Pos.Z)
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
			case 't':
				dayT += 0.06 // step the day forward
				world.SetTime(dayT)
				scene = world.Scene()
				refiner.Reset()
			case 'p':
				*pathT = !*pathT
				refiner.Reset()
			case 'm':
				showMap = !showMap
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
