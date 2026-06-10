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
//	go run ./cmd/rayexplore -weather rain           # rain (or snow/fog) that follows you
//	go run ./cmd/rayexplore -seasons                 # walk forward through spring→summer→autumn→winter
//	go run ./cmd/rayexplore -mood                    # the world's tone follows how you move
//	go run ./cmd/rayexplore -path -style oil         # a painted (oil/ink/poster) look
//	go run ./cmd/rayexplore -path -portals           # an Escher portal: a non-Euclidean window
//	go run ./cmd/rayexplore -path -dream             # a lens-and-film dream pass
//	go run ./cmd/rayexplore -stereo                  # red-cyan anaglyph (3-D with glasses)
//	go run ./cmd/rayexplore -story                   # the world unfolds as a story, in chapters
//	go run ./cmd/rayexplore -sound -music            # a generative melody tuned to the world
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
// If melody is non-nil it is mixed in (looped) under the ambient — a generative
// score for the world.
func startSoundscape(mu *sync.Mutex, feat *raydir.AmbientFeatures, melody []int16) {
	pl, err := audio.NewDefaultPlayback()
	if err != nil || pl.Open() != nil {
		fmt.Fprintln(os.Stderr, "sound: no audio device available; continuing silent")
		return
	}
	rate := audio.DefaultFormat().SampleRate
	frame := rate / 50 // 20ms
	go func() {
		t := 0.0
		pos := 0 // playhead into the looping melody
		tk := time.NewTicker(20 * time.Millisecond)
		defer tk.Stop()
		for range tk.C {
			mu.Lock()
			f := *feat
			mu.Unlock()
			buf := raydir.AmbientFrame(f, rate, t, frame)
			if len(melody) > 0 { // mix the score in under the ambient
				for i := range buf {
					m := int(melody[pos%len(melody)]) / 2 // half gain
					pos++
					s := int(buf[i]) + m
					if s > 32767 {
						s = 32767
					} else if s < -32768 {
						s = -32768
					}
					buf[i] = int16(s)
				}
			}
			_, _ = pl.Write(buf)
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
	weather := flag.String("weather", "", "weather that follows you: rain | snow | fog")
	seasons := flag.Bool("seasons", false, "walk through the year: foliage, ground and sky shift spring→summer→autumn→winter")
	mood := flag.Bool("mood", false, "the director reads how you move (linger/press on/wander) and shifts the tone of what it builds")
	style := flag.String("style", "", "non-photoreal look: oil | ink | poster")
	portals := flag.Bool("portals", false, "drop an Escher portal ahead — a non-Euclidean window (best with -path)")
	dream := flag.Bool("dream", false, "lens & film post: chromatic aberration, barrel warp, grain, vignette")
	stereo := flag.Bool("stereo", false, "render in depth: a red-cyan anaglyph (view with red/cyan glasses)")
	story := flag.Bool("story", false, "follow a story: the world unfolds in chapters, a beacon marks each threshold")
	music := flag.Bool("music", false, "play a generative melody under the soundscape (major by day, minor at night); needs -sound")
	cols := flag.Int("w", 80, "width in terminal cells")
	rows := flag.Int("h", 38, "height in terminal cells")
	flag.Parse()
	pxW, pxH := *cols*2, *rows*4
	painterStyle, _ := raytrace.ParsePainterly(*style) // StyleNone when unset/unknown

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
	world.SetSeasonal(*seasons) // before any region grows, so foliage is tinted as it's built
	world.SetMoodSensing(*mood) // read how the walker moves and bias what's grown
	var tale *raydir.Story
	if *story {
		tale = raydir.DefaultStory()
	}
	seed := func() string { // the next region's prompt: the story's, or the rotation
		if tale != nil {
			return tale.Prompt()
		}
		return nextPrompt()
	}
	var storyBanner string
	if tale != nil { // open on the first chapter's narration
		c := tale.Chapter()
		n, tot := tale.Progress()
		storyBanner = fmt.Sprintf("📖 %d/%d %s — %s", n, tot, c.Title, c.Line)
	}
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
		if _, err := world.Grow(b, world.BiasPrompt(seed()), raytrace.Vec3{X: 0, Y: 0, Z: frontZ}); err != nil {
			fmt.Fprintln(os.Stderr, "director failed to author the first region:", err)
		}
		if tale != nil {
			tale.Advance() // account the opening region against the first chapter
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
	world.SetWeather(*weather) // rain/snow/fog that follows the walker (before the scene is built so fog bakes in)
	if *portals {              // a non-Euclidean window ahead, linking to the place behind you
		world.AddPortal(raytrace.NewPortal(
			raytrace.Vec3{X: 0, Y: 2.2, Z: 20}, raytrace.Vec3{X: 2.5}, raytrace.Vec3{Y: 2.2},
			raytrace.Translate(raytrace.Vec3{Z: -30})))
	}
	scene := world.Scene()
	pathOpt := raytrace.PathOptions{Samples: 4, MaxDepth: 5, Seed: 1, NEE: true, MIS: true, Sobol: true}
	// progressive refinement: stand still (press Enter) and the path-traced view
	// keeps sharpening toward a clean render; moving or growing restarts it.
	refiner := raydir.NewRefiner(pxW, pxH, 4, 256, pathOpt)

	// grow authors a new region ahead and rebuilds the scene.
	grow := func() {
		jitter := math.Sin(float64(world.Chunks())*1.7) * 2.5
		frontZ += 14
		n, err := world.Grow(b, world.BiasPrompt(seed()), raytrace.Vec3{X: jitter, Y: 0, Z: frontZ})
		if err != nil {
			fmt.Fprintln(os.Stderr, "grow failed:", err)
			return
		}
		_ = n
		if tale != nil { // turn the page when the chapter has had its regions
			if entered, ch := tale.Advance(); entered {
				idx, tot := tale.Progress()
				world.AddDecor(raydir.BeaconObjects(raytrace.Vec3{X: jitter, Z: frontZ}, raydir.ChapterColor(idx-1))...)
				storyBanner = fmt.Sprintf("📖 %d/%d %s — %s", idx, tot, ch.Title, ch.Line)
			}
		}
		scene = world.Scene()
		refiner.Reset() // the scene changed
	}

	// optional procedural soundscape: synthesise the world's ambient locally and
	// play it (graceful fallback when there's no audio device).
	var featMu sync.Mutex
	feat := world.Ambient()
	if *sound {
		var melody []int16
		if *music { // a generative score tuned to the world (day/night, mood, liveliness)
			melody = raydir.MelodyPCM(world.Score(), audio.DefaultFormat().SampleRate, 24, 1)
		}
		startSoundscape(&featMu, &feat, melody)
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
	dreamFrame := 0
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
		world.StepWeather(dt, cam.Pos)                 // rain/snow drifts around you
		world.ObserveWalker(raydir.PoseOf(cam), dt)    // read how you move (for mood)
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
		eye := func(ec raytrace.Camera) image.Image { // one eye, for stereo
			if *pathT {
				o := pathOpt
				o.Samples = 16
				return raytrace.PathRender(fscene, ec, pxW, pxH, o)
			}
			return raytrace.Render(fscene, ec, pxW, pxH, raytrace.Options{Samples: 1})
		}
		switch {
		case *stereo:
			// a red-cyan anaglyph: render both eyes and fuse them (bypasses the
			// progressive refiner, which holds a single accumulator).
			l, r := raytrace.StereoCameras(c, 0.32)
			im = raytrace.Anaglyph(eye(l), eye(r))
			if *pathT {
				spp = 16
			}
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
		if painterStyle != raytrace.StyleNone { // an oil/ink/poster look over the frame
			im = raytrace.Painterly(im, painterStyle)
		}
		if *dream { // a lens-and-film pass; grain shimmers from the frame counter
			dreamFrame++
			im = raytrace.Dream(im, raytrace.DreamOptions{Chroma: 0.012, Grain: 0.04, Vignette: 0.25, Distort: 0.12, Seed: uint32(dreamFrame)})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		if showMap {
			fmt.Print("\n" + raydir.Minimap(world.Landmarks(), nil, cam.Pos, *cols, *rows/2))
		}
		if guideMsg != "" {
			fmt.Printf("\n  🧭 %s", guideMsg)
		}
		if storyBanner != "" {
			fmt.Printf("\n  %s", storyBanner)
		}
		mins := int((dayT - math.Floor(dayT)) * 24 * 60)
		season := ""
		if world.Seasonal() {
			season = " | 🍂" + raydir.SeasonAt(cam.Pos.Z).Name
		}
		if m := world.MoodName(); m != "" {
			season += " | 🎭" + m
		}
		fmt.Printf("\n[director: %s | chunks:%d props:%d | 🕓%02d:%02d spp:%d%s | pos (%.1f,%.1f,%.1f) | w/s a/d q/e r/f g=grow t=time p=path m=map Enter=refine x=quit] ",
			who, world.Chunks(), world.Props(), mins/60, mins%60, spp, season, cam.Pos.X, cam.Pos.Y, cam.Pos.Z)
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
