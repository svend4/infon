// Command rayarena stages a 3-D "arena": a tvcp-ai brain authors a moving world
// (orbiting, breathing spheres) AND flies the camera, the whole evolving scene is
// broadcast as independent Reed-Solomon packets (one per tick), and a spectator
// reconstructs it over a lossy channel — rendering whatever arrives and freezing
// on loss. neural -> bytes -> real 3-D, for many viewers. Default driver is the
// built-in reference brain; -brain URL points at a live model.
//
//	go run ./cmd/rayarena                 # reference brain, 15% loss
//	go run ./cmd/rayarena -loss 0.4       # heavier loss (more freezing)
//	go run ./cmd/rayarena -brain http://127.0.0.1:8092/v1/decide
package main

import (
	"flag"
	"fmt"
	"math/rand"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai brain URL (default: built-in reference)")
	ticks := flag.Int("ticks", 60, "number of arena ticks")
	cols := flag.Int("w", 90, "width in cells")
	rows := flag.Int("h", 45, "height in cells")
	spp := flag.Int("spp", 1, "anti-aliasing samples per axis")
	mode := flag.String("mode", "auto", "terminal mode")
	loss := flag.Float64("loss", 0.15, "spectator packet-loss probability (0..1)")
	ecc := flag.Int("ecc", 12, "Reed-Solomon parity bytes per block")
	flag.Parse()

	var b brain.Brain = brain.Local{}
	if *brainURL != "" {
		b = brain.HTTPBrain{URL: *brainURL}
	}
	camDir := &raydir.BrainDirector{B: b}
	worldDir := &raydir.BrainWorldDirector{B: b}

	spheres := []raytrace.Sphere{
		{Center: raytrace.Vec3{X: -1.6, Y: 1, Z: 0}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.85, Y: 0.3, Z: 0.3}, Spec: 0.4, Shine: 48}},
		{Center: raytrace.Vec3{X: 1.6, Y: 1, Z: 0}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.9, Y: 0.9, Z: 0.95}, Reflect: 0.6}},
		{Center: raytrace.Vec3{X: 0, Y: 1.2, Z: 1.8}, Radius: 0.7, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.3, Y: 0.6, Z: 0.9}}},
	}
	init := raydir.CamState{Angle: 0, Radius: 6, Height: 2.6, Target: raytrace.Vec3{X: 0, Y: 1, Z: 0}}
	light := raytrace.Vec3{X: 6, Y: 9, Z: -4}

	anim := raydir.Choreograph(camDir, worldDir, init, spheres, light, *ticks)
	packets := raytrace.EncodeBroadcast(anim, *ecc)

	rng := rand.New(rand.NewSource(1))
	spec := raytrace.NewSpectator(*ecc)
	var lastScene raytrace.SceneWire
	var bytesTotal, lost int
	for _, pkt := range packets {
		bytesTotal += len(pkt)
		if rng.Float64() < *loss {
			lastScene, _ = spec.Miss()
			lost++
			continue
		}
		lastScene, _ = spec.Recv(pkt)
	}

	scene := lastScene.Build(0.14)
	name := *mode
	if name == "" || name == "auto" {
		name = terminal.DetectCapability().BestBlitMode()
	}
	rm, _ := babe.ParseRenderMode(name)
	img := raytrace.Render(scene, lastScene.Cam, *cols*2, *rows*4, raytrace.Options{Samples: *spp})
	fmt.Print(babe.ImageToFrameMode(img, *cols, *rows, rm).Render())

	perFrame := 0.0
	if len(packets) > 0 {
		perFrame = float64(bytesTotal) / float64(len(packets))
	}
	fmt.Printf("arena: %d ticks, %d packets, %d bytes (%.0f B/frame), %d lost (%.0f%%); spectator froze on loss; mode=%s\n",
		*ticks, len(packets), bytesTotal, perFrame, lost, *loss*100, name)
}
