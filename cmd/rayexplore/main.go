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
//	go run ./cmd/rayexplore -branch                  # branching paths: walk left/right at a crossroads
//	go run ./cmd/rayexplore -maze -path              # a "square forest" labyrinth of blocks and roads
//	go run ./cmd/rayexplore -optic tunnel            # dream optics: tunnel vision / skew / double
//	go run ./cmd/rayexplore -mirror                  # dream symmetry: a north-south mirror world
//	go run ./cmd/rayexplore -layer lower -path       # descend to the dark lower world
//	go run ./cmd/rayexplore -funnel -path            # a swirling transit funnel (vortex portal)
//	go run ./cmd/rayexplore -materialize             # each region renders in from voxel blocks
//	go run ./cmd/rayexplore -flyer                   # a predator stalks you (drains luminosity)
//	go run ./cmd/rayexplore -sprites                 # dream characters: type "?question" to ask one
//	go run ./cmd/rayexplore -intention               # type "!a theme" and hold it: the world bends to your will
//	go run ./cmd/rayexplore -hexagram 101010         # cast a world from an I-Ching hexagram
//	go run ./cmd/rayexplore -hexagram 000000 -q6walk # grand tour of all 64 hexagram worlds (Q6 Gray code)
//	go run ./cmd/rayexplore -memory                  # the director remembers and echoes past places
//	go run ./cmd/rayexplore -path -ris 16            # ReSTIR many-lights: clean firefly/lantern worlds
//	go run ./cmd/rayexplore -sound -music            # a generative melody tuned to the world
//	go run ./cmd/rayexplore -travelogue              # collect the trip as postcards (saved on quit)
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
	"image/png"
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
	optic := flag.String("optic", "", "dream optics: skew (broken perspective) | tunnel (tunnel vision) | double (split image)")
	ris := flag.Int("ris", 0, "ReSTIR/RIS candidate lights per pixel (path mode): cleaner many-light worlds (e.g. 16)")
	story := flag.Bool("story", false, "follow a story: the world unfolds in chapters, a beacon marks each threshold")
	maze := flag.Bool("maze", false, "a 'square forest': a flat grid of jungle/building blocks split by roads — a labyrinth to walk")
	mirror := flag.Bool("mirror", false, "dream symmetry: the world is doubled by a north-south reflection (turn around to see it)")
	layer := flag.String("layer", "", "vertical layer: upper (airy) | lower (dark, a descent tunnel)")
	funnel := flag.Bool("funnel", false, "a swirling transit funnel (vortex) ahead — a portal at its mouth (best with -path)")
	materialize := flag.Bool("materialize", false, "each new region 'renders in' from coarse voxel blocks to sharp, the way a dream forms")
	flyer := flag.Bool("flyer", false, "a predator (the 'flyer') stalks you and drains your luminosity — keep ahead of it")
	robots := flag.Int("robots", 0, "spawn N patrolling robots — the logistics yard shares your world (status beacons green->red)")
	temporal := flag.Bool("temporal", false, "temporal reprojection: reuse samples across camera motion for cleaner path-traced frames while walking (path mode, static world)")
	sprites := flag.Bool("sprites", false, "dream characters you can question: type '?your question' to ask the nearest")
	intention := flag.Bool("intention", false, "the art of intention: type '!a theme' and hold it — the world grown ahead bends to your will")
	hexagram := flag.String("hexagram", "", "cast a world from an I-Ching hexagram: six lines like 101010 or yynnyn")
	q6walk := flag.Bool("q6walk", false, "with -hexagram: walk the Q6 hypercube of all 64 hexagram worlds (one line changes per region)")
	memory := flag.Bool("memory", false, "the director remembers past regions and echoes thematically similar ones ahead (RAG-style)")
	branch := flag.Bool("branch", false, "branching paths: at a crossroads, walk left (a) or right (d) to choose where the world goes")
	music := flag.Bool("music", false, "play a generative melody under the soundscape (major by day, minor at night); needs -sound")
	travel := flag.Bool("travelogue", false, "collect the journey as captioned postcards; saved to travelogue.png on quit")
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
	hexName := ""
	if *hexagram != "" { // cast a world from an I-Ching hexagram
		if h, ok := raydir.ParseHexagram(*hexagram); ok {
			prompts = []string{h.Prompt()}
			hexName = h.Name()
			if *q6walk { // a grand tour of the Q6 hypercube: each region a one-line neighbour
				walk := h.GrayWalk()
				prompts = prompts[:0]
				for _, g := range walk {
					prompts = append(prompts, g.Prompt())
				}
				hexName = "Q6 walk from " + h.Name()
			}
		} else {
			fmt.Fprintln(os.Stderr, "hexagram: need six lines like 101010 or yynnyn")
		}
	}
	pi := 0
	nextPrompt := func() string { p := prompts[pi%len(prompts)]; pi++; return p }

	world := raydir.NewWorld()
	world.SetSeasonal(*seasons) // before any region grows, so foliage is tinted as it's built
	world.SetMoodSensing(*mood) // read how the walker moves and bias what's grown
	world.SetMemory(*memory)    // remember regions so later prompts can echo them
	var tale *raydir.Story
	if *story {
		tale = raydir.DefaultStory()
	}
	var fork *raydir.Branching
	forkActive := false
	if *branch {
		fork = raydir.DefaultBranching()
		forkActive = true // present the first crossroads at once
	}
	regionsSinceFork := 0
	var forkBanner string
	seed := func() string { // the next region's prompt: a chosen branch, the story, or the rotation
		switch {
		case fork != nil:
			return fork.Prompt()
		case tale != nil:
			return tale.Prompt()
		default:
			return nextPrompt()
		}
	}
	var storyBanner string
	if tale != nil { // open on the first chapter's narration
		c := tale.Chapter()
		n, tot := tale.Progress()
		storyBanner = fmt.Sprintf("📖 %d/%d %s — %s", n, tot, c.Title, c.Line)
	}
	var tlog *raydir.Travelogue
	if *travel {
		tlog = raydir.NewTravelogue("A Walk Through Worlds")
	}
	capturePending := *travel // capture the opening view, then each new place
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
		if _, err := world.Grow(b, world.BiasPrompt(world.RecallPrompt(world.IntendPrompt(seed()))), raytrace.Vec3{X: 0, Y: 0, Z: frontZ}); err != nil {
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
	switch *layer {  // a vertical layer paints its own sky (instead of a time of day)
	case "upper":
		world.SetLayer(raydir.LayerUpper)
	case "lower":
		world.SetLayer(raydir.LayerLower)
		world.AddDecor(raydir.DescentTunnel(raytrace.Vec3{X: 0, Y: 0, Z: 6}, 10)...) // the way down
	default:
		world.SetTime(dayT)
	}
	if *mirror {
		world.SetMirror(true, 0) // reflect the world north-south across the start
	}
	world.SetClouds(*clouds)
	world.SetWeather(*weather) // rain/snow/fog that follows the walker (before the scene is built so fog bakes in)
	if *ris > 1 {              // ReSTIR many-lights: cleaner fireflies/lanterns (NEE without MIS)
		world.SetRIS(*ris)
	}
	if *maze { // a "square forest" labyrinth spanning the path ahead
		world.AddDecor(raydir.NewSquareForest(8, 10, 1).Objects(raytrace.Vec3{X: -32, Z: 8})...)
	}
	if *funnel { // a swirling transit funnel ahead, linking deeper into the world
		world.AddFunnel(raytrace.Vec3{X: 0, Z: 16}, raytrace.Translate(raytrace.Vec3{Z: 40}))
	}
	if *portals { // a non-Euclidean window ahead, linking to the place behind you
		world.AddPortal(raytrace.NewPortal(
			raytrace.Vec3{X: 0, Y: 2.2, Z: 20}, raytrace.Vec3{X: 2.5}, raytrace.Vec3{Y: 2.2},
			raytrace.Translate(raytrace.Vec3{Z: -30})))
	}
	scene := world.Scene()
	pathOpt := raytrace.PathOptions{Samples: 4, MaxDepth: 5, Seed: 1, NEE: true, MIS: true, Sobol: true}
	if *ris > 1 { // RIS engages on NEE-only paths (mode 1): turn MIS off
		pathOpt.MIS = false
	}
	// progressive refinement: stand still (press Enter) and the path-traced view
	// keeps sharpening toward a clean render; moving or growing restarts it.
	refiner := raydir.NewRefiner(pxW, pxH, 4, 256, pathOpt)
	// optional temporal reprojection: carries samples across camera motion so a
	// walk through a STATIC world stays clean; recreated (reset) on any scene change.
	var tr *raytrace.TemporalReprojector
	resetTR := func() {}
	if *temporal {
		tr = raytrace.NewTemporalReprojector(pxW, pxH, 0.15)
		resetTR = func() { tr = raytrace.NewTemporalReprojector(pxW, pxH, 0.15) }
	}

	formStart := time.Now() // when the current region began "rendering in" (materialize)
	// grow authors a new region ahead and rebuilds the scene.
	grow := func() {
		jitter := math.Sin(float64(world.Chunks())*1.7) * 2.5
		frontZ += 14
		n, err := world.Grow(b, world.BiasPrompt(world.RecallPrompt(world.IntendPrompt(seed()))), raytrace.Vec3{X: jitter, Y: 0, Z: frontZ})
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
		refiner.Reset()        // the scene changed
		resetTR()              // history is stale after the world changes
		capturePending = true  // a new place worth a postcard
		regionsSinceFork++     // toward the next crossroads
		formStart = time.Now() // the new region renders in from blocks
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
	if *flyer { // a predator that hunts the walker
		world.SpawnFlyer(cam.Pos.Add(raytrace.Vec3{X: 6, Z: 12}))
	}
	if *robots > 0 { // a yard of patrolling machines shares the walker's world
		for i := 0; i < *robots; i++ {
			c := cam.Pos.Add(raytrace.Vec3{X: float64(i*3) - float64(*robots), Z: 10})
			loop := []raytrace.Vec3{c, c.Add(raytrace.Vec3{X: 4}), c.Add(raytrace.Vec3{X: 4, Z: 5}), c.Add(raytrace.Vec3{Z: 5})}
			status := 0.0
			if *robots > 1 {
				status = float64(i) / float64(*robots-1)
			}
			world.SpawnRobot(raydir.NewRobot(c, loop, status))
		}
	}
	if *sprites { // a few dream characters to meet and question
		world.AddSprite(raydir.NewSprite("Mira", cam.Pos.Add(raytrace.Vec3{X: -3, Z: 9}), 1))
		world.AddSprite(raydir.NewSprite("Ko", cam.Pos.Add(raytrace.Vec3{X: 4, Z: 13}), 3))
	}
	luminosity := 1.0
	var guideMsg string
	var spriteMsg string

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
		world.Reveal(cam.Pos, 14)                      // lift the fog where you've been (bird's-eye map)
		world.StepCreatures(dt, cam.Pos)               // the inhabitants live and react
		world.StepWeather(dt, cam.Pos)                 // rain/snow drifts around you
		world.ObserveWalker(raydir.PoseOf(cam), dt)    // read how you move (for mood)
		world.StepSprites(dt)                          // dream characters drift
		world.StepRobots(dt)                           // patrolling machines move on their rounds
		world.HoldIntention(dt)                        // sustain the held intention (no-op if none)
		if *flyer {                                    // the predator hunts; it drains you when close
			if world.StepFlyer(dt, cam.Pos) {
				luminosity = math.Max(0, luminosity-dt*0.3)
			} else {
				luminosity = math.Min(1, luminosity+dt*0.08)
			}
		}
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
		case *pathT && *temporal && !world.HasAnimated() && comp == nil:
			// temporal reprojection reuses samples across motion (clean while walking);
			// skipped for animated worlds, which would ghost (history is static-only).
			im = tr.Frame(fscene, c, pathOpt)
			spp = pathOpt.Samples
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
		switch *optic { // a dream optical oddity over the frame
		case "skew":
			im = raytrace.PerspectiveSkew(im, 0.35)
		case "tunnel":
			im = raytrace.TunnelVision(im, 0.85)
		case "double":
			im = raytrace.DoubleVision(im, pxW/24, 0)
		}
		if *materialize { // the region renders in from coarse voxel blocks to sharp
			if ft := time.Since(formStart).Seconds() / 1.4; ft < 1 {
				im = raytrace.Materialize(im, ft)
			}
		}
		if tlog != nil && capturePending { // a postcard of this place
			place := "the world"
			if marks := world.Landmarks(); len(marks) > 0 {
				place = marks[len(marks)-1].Name
			}
			tlog.Capture(place, dayT, raydir.Thumbnail(im, 200, 130))
			capturePending = false
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		if showMap {
			fmt.Print("\n" + raydir.Minimap(world.Landmarks(), nil, cam.Pos, *cols, *rows/2))
		}
		if guideMsg != "" {
			fmt.Printf("\n  🧭 %s", guideMsg)
		}
		if spriteMsg != "" {
			fmt.Printf("\n  💬 %s", spriteMsg)
		}
		if storyBanner != "" {
			fmt.Printf("\n  %s", storyBanner)
		}
		if fork != nil { // a crossroads to choose, or the branch last taken
			if forkActive {
				if f, ok := fork.PendingFork(); ok {
					fmt.Printf("\n  ⑂ %s   [a] %s   [d] %s", f.Question, f.Left.Label, f.Right.Label)
				}
			} else if forkBanner != "" {
				fmt.Printf("\n  %s", forkBanner)
			}
		}
		mins := int((dayT - math.Floor(dayT)) * 24 * 60)
		season := ""
		if world.Seasonal() {
			season = " | 🍂" + raydir.SeasonAt(cam.Pos.Z).Name
		}
		if m := world.MoodName(); m != "" {
			season += " | 🎭" + m
		}
		if *flyer {
			season += fmt.Sprintf(" | ✨%.0f%%", luminosity*100)
			if luminosity < 0.45 {
				fmt.Printf("\n  ⚠ the flyer is draining your luminosity — run!")
			}
		}
		if th, st := world.Intention(); th != "" {
			season += fmt.Sprintf(" | 🔮%s %.0f%%", th, st*100)
		}
		if hexName != "" {
			season += " | ䷂" + hexName
		}
		fmt.Printf("\n[director: %s | chunks:%d props:%d | 🕓%02d:%02d spp:%d%s | pos (%.1f,%.1f,%.1f) | w/s a/d q/e r/f g=grow t=time p=path m=map Enter=refine x=quit] ",
			who, world.Chunks(), world.Props(), mins/60, mins%60, spp, season, cam.Pos.X, cam.Pos.Y, cam.Pos.Z)
	}

	// chooseFork takes a branch when a crossroads is active (you walk left/right).
	chooseFork := func(goRight bool) {
		if fork == nil || !forkActive {
			return
		}
		if ch, ok := fork.Choose(goRight); ok {
			forkBanner = "⑂ you took the " + ch.Label + " (" + strings.Join(fork.Path(), " → ") + ")"
			forkActive = false
			regionsSinceFork = 0
		}
	}

	fmt.Printf("rayexplore — walking a world authored by %s. Each region is shipped as a tiny scene description and ray-traced locally.\n", who)
	render()

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if *sprites && strings.HasPrefix(line, "?") { // question the nearest dream character
			if sp := world.NearestSprite(cam.Pos, 10); sp != nil {
				spriteMsg = sp.Name + ": " + sp.Answer(strings.TrimPrefix(line, "?"))
			} else {
				spriteMsg = "(no one near to ask)"
			}
			render()
			continue
		}
		if *intention && strings.HasPrefix(line, "!") { // set or clear the held intention
			if theme := strings.TrimSpace(strings.TrimPrefix(line, "!")); theme == "" {
				world.ClearIntention()
			} else {
				world.SetIntention(theme)
			}
			render()
			continue
		}
		for _, ch := range strings.ToLower(line) {
			switch ch {
			case 'w':
				cam.Walk(1.2, 0)
			case 's':
				cam.Walk(-1.2, 0)
			case 'a':
				cam.Walk(0, -1.2)
				chooseFork(false) // at a crossroads, stepping left chooses the left branch
			case 'd':
				cam.Walk(0, 1.2)
				chooseFork(true) // stepping right chooses the right branch
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
				resetTR()
			case 'p':
				*pathT = !*pathT
				refiner.Reset()
				resetTR()
			case 'm':
				showMap = !showMap
			case 'k': // save the bird's-eye fog-of-war map
				if cg := world.Cartograph(); cg != nil {
					if f, err := os.Create("map.png"); err == nil {
						_ = png.Encode(f, cg.Render(world.Landmarks(), cam.Pos, 600, 600))
						_ = f.Close()
						fmt.Fprintln(os.Stderr, "\nsaved map.png")
					}
				}
			case 'x':
				if tlog != nil && tlog.Len() > 0 { // save the journey's postcards
					if f, err := os.Create("travelogue.png"); err == nil {
						_ = png.Encode(f, tlog.Render(3))
						_ = f.Close()
						fmt.Fprintf(os.Stderr, "\nsaved travelogue.png (%d moments)\n", tlog.Len())
					}
				}
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
			resetTR()
		}
		// after walking a stretch since the last choice, the next crossroads appears.
		if fork != nil && !forkActive && !fork.Done() && regionsSinceFork >= 3 {
			forkActive = true
		}
		render()
	}
}
