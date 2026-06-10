// Command rayface renders a meeting INSIDE the world: participant avatars whose
// faces are drawn from keypoints (raydir.AvatarFace — the landmarks internal/avatar
// sends ~35 kbps over the wire) stand together in a shared world and look at you.
// It is the missing entity that puts the call into the dream rather than beside it.
// Here the faces are synthetic; live keypoints from the avatar sidecar drop in
// unchanged.
//
//	go run ./cmd/rayface
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out = flag.String("out", "face", "output basename")
		w   = flag.Int("w", 720, "render width")
		h   = flag.Int("h", 460, "render height")
		spp = flag.Int("spp", 64, "samples per pixel")
		n   = flag.Int("n", 3, "participants")
	)
	flag.Parse()

	world := raydir.NewWorld()
	world.SetTime(0.42) // a soft daylight

	var extra []raytrace.Object
	for i := 0; i < *n; i++ {
		x := float64(i)*2.0 - float64(*n-1)
		pose := raydir.Pose{Pos: raytrace.Vec3{X: x, Y: 1.6, Z: 3.5 + 0.4*float64(i%2)}, Yaw: math.Pi} // facing the camera (-Z)
		extra = append(extra, raydir.AvatarFace(pose, syntheticFace(i), raydir.AvatarColor(uint32(i)))...)
	}
	scene := world.SceneWith(extra)

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 1.6, Z: -0.5}, Pitch: -0.02, FOV: math.Pi / 3}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 6, NEE: true, MIS: true, Sobol: true})
	img = raytrace.PostProcess(img, 1.0, 0.8, 0.5) // bloom the glowing heads
	writePNG(*out+".png", img)
	fmt.Printf("a meeting of %d inside the world -> %s.png\n", *n, *out)
}

// syntheticFace builds recognizable face keypoints (oval, two eyes, nose, mouth),
// slightly varied per participant.
func syntheticFace(seed int) [][2]float32 {
	var p [][2]float32
	add := func(x, y float64) { p = append(p, [2]float32{float32(x), float32(y)}) }
	for i := 0; i < 18; i++ { // oval outline
		a := float64(i) / 18 * 2 * math.Pi
		add(0.5+0.42*math.Cos(a), 0.5+0.47*math.Sin(a))
	}
	eyeY := 0.40 + 0.02*float64(seed%2)
	for _, ex := range []float64{0.35, 0.65} { // two eyes
		for i := 0; i < 6; i++ {
			a := float64(i) / 6 * 2 * math.Pi
			add(ex+0.06*math.Cos(a), eyeY+0.04*math.Sin(a))
		}
	}
	add(0.5, 0.50) // nose
	add(0.5, 0.56)
	smile := 0.04 + 0.03*float64(seed%3) // varied mouth curve
	for i := 0; i <= 8; i++ {
		t := float64(i) / 8
		add(0.36+0.28*t, 0.66+smile*math.Sin(t*math.Pi))
	}
	return p
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
