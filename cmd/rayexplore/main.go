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
//	go run ./cmd/rayexplore -guide                  # an AI companion leads you on a tour
//	go run ./cmd/rayexplore -creatures              # a flock lives in the world and reacts to you
//	go run ./cmd/rayexplore -path -denoise          # clean path-traced frames while moving
//	go run ./cmd/rayexplore -image photo.png        # walk into a world derived from a picture
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
	_ "image/jpeg"
	_ "image/png"
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

// loadImage decodes a PNG/JPG file.
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

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
	imageSeed := flag.String("image", "", "seed the first region from a picture (PNG/JPG): walk into a world derived from it")
	denoise := flag.Bool("denoise", false, "path mode: render few samples and denoise (clean frames while moving)")
	sound := flag.Bool("sound", false, "play a procedural soundscape of the world (needs an audio device)")
	guide := flag.Bool("guide", false, "an AI companion that walks with you and leads a tour of the world")
	creatures := flag.Bool("creatures", false, "a flock of inhabitants that lives in the world, gathers at places, and scatters from you")
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
	if *imageSeed != "" { // walk into a world derived from a picture
		if img, err := loadImage(*imageSeed); err == nil {
			world.AddRegion(raydir.Region{Index: 0, At: raytrace.Vec3{Z: frontZ}, Spec: raydir.SceneFromImage(img)})
			who = "image: " + *imageSeed
		} else {
			fmt.Fprintln(os.Stderr, "image:", err)
		}
	}
	if world.Chunks() == 0 {
		if _, err := world.Grow(b, nextPrompt(), raytrace.Vec3{X: 0, Y: 0, Z: frontZ}); err != nil {
			fmt.Fprintln(os.Stderr, "director failed to author the first region:", err)
		}
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

	var comp *raydir.Guide
	if *guide {
		comp = raydir.NewGuide("Guide", cam.Pos.Add(raytrace.Vec3{X: 3, Z: 3}))
	}
	if *creatures { // give the world a flock of its own, ahead of the start
		world.SpawnFlock(14, cam.Pos.Add(raytrace.Vec3{Y: 4, Z: 16}), 20)
	}
	var guideMsg string

	start, lastTick := time.Now(), time.Now()
	render := func() {
		now := time.Now()
		dt := now.Sub(lastTick).Seconds() // wall-clock frame step for movers
		if dt > 0.5 {
			dt = 0.5
		}
		lastTick = now
		world.SetAnimTime(time.Since(start).Seconds()) // keep the world alive
		world.Tread(cam.Pos)                           // wear a path where you walk
		world.StepCreatures(dt, cam.Pos)               // the inhabitants live and react
		featMu.Lock()
		feat = world.Ambient()
		featMu.Unlock()
		c := cam.Camera()
		var extra []raytrace.Object
		if comp != nil { // a moving companion: step it and inject its avatar this frame
			comp.Step(dt, cam.Pos, world.Landmarks())
			if r, ok := comp.Remark(time.Since(start).Seconds(), world.Landmarks()); ok {
				guideMsg = comp.Name + ": " + r
			}
			extra = raydir.AvatarSpheres(comp.Pose(), raydir.GuideColor)
		}
		// a living world (movers, flock, or a companion) is rebuilt each frame so the
		// motion shows; a fully static world reuses the cached, BVH-built scene.
		fscene := scene
		if world.HasAnimated() || comp != nil {
			fscene = world.SceneWith(extra)
		}
		var im image.Image
		spp := 1
		switch {
		case *pathT && *denoise:
			// fast clean frame: a few samples + an edge-aware (guided) à-trous denoise,
			// so a moving path-traced walk stays clean without waiting to converge.
			o := pathOpt
			o.Samples = 5
			raw := raytrace.PathRender(fscene, c, pxW, pxH, o)
			alb, nrm := raytrace.GBuffer(fscene, c, pxW, pxH)
			im = raytrace.DenoiseGuided(raw, alb, nrm, 4, 0.5, 0.2, 0.3)
			spp = 5
		case *pathT:
			if world.HasAnimated() || comp != nil {
				refiner.Reset() // a moving world can't accumulate; render fresh
			}
			im = refiner.Frame(fscene, c)
			spp = refiner.Samples()
		default:
			im = raytrace.Render(fscene, c, pxW, pxH, raytrace.Options{Samples: 1})
		}
		if *grade {
			im = raytrace.Grade(im, raytrace.GradeOptions{BloomThresh: 1.0, BloomStrength: 0.4, Vignette: 0.35, AgX: true})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		if showMap {
			fmt.Print("\n" + raydir.Minimap(world.Landmarks(), nil, cam.Pos, *cols, *rows/2))
		}
		if guideMsg != "" {
			fmt.Printf("\n  🧭 %s", guideMsg)
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
		// ...and forgets far behind, so a long walk keeps flat memory (worn paths stay).
		if world.Prune(cam.Pos.Z-55) > 0 {
			scene = world.Scene()
			refiner.Reset()
		}
		render()
	}
}
