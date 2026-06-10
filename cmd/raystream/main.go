// Command raystream is a call where the wire carries scene DELTAS, not pixels: a
// sphere orbits a small world over many frames, each frame sent as a rayscene
// P-frame (raydir.SceneStream), and the receiver reconstructs and ray-traces it
// locally. It prints what each transport would put on the wire — pixels, a full
// scene per frame, and deltas — and renders the receiver's reconstruction of the first and a
// mid-orbit frame to show fidelity. "Meaning, not pixels", taken to its codec.
//
//	go run ./cmd/raystream -frames 60
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out    = flag.String("out", "stream", "output basename")
		frames = flag.Int("frames", 60, "frames in the call")
		w      = flag.Int("w", 360, "render width")
		h      = flag.Int("h", 240, "render height")
		spp    = flag.Int("spp", 56, "samples per pixel")
	)
	flag.Parse()

	// the world: static forms plus one orbiting sphere (the only thing that changes).
	base := brain.SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: [3]float64{0.36, 0.46, 0.78}, SkyBot: [3]float64{0.82, 0.84, 0.92},
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: [3]float64{0.5, 0.52, 0.5}},
			{Kind: "sphere", X: -1.6, Y: 1, Z: 4, R: 1, Color: [3]float64{0.85, 0.3, 0.3}, Rough: 0.4},
			{Kind: "sphere", X: 1.6, Y: 0.8, Z: 4, R: 0.8, Glass: 1.5},
			{X: 0, Y: 6, Z: -1, R: 0.7, Emit: [3]float64{16, 16, 15}},
			{Kind: "sphere", X: 0, Y: 1, Z: 4, R: 0.5, Color: [3]float64{0.95, 0.85, 0.3}, Emit: [3]float64{2, 1.7, 0.5}}, // the orbiter
		},
		Name: "stream",
	}
	orbiter := len(base.Objects) - 1

	var s raydir.SceneStream
	fullEach := 0
	var first, mid brain.SceneSpec
	for f := 0; f < *frames; f++ {
		a := float64(f) / float64(*frames) * 2 * math.Pi
		base.Objects[orbiter].X = math.Cos(a) * 2.2
		base.Objects[orbiter].Z = 4 + math.Sin(a)*1.6
		s.Push(base)
		fullEach += raydir.SceneBytes(base)
		if f == 0 {
			first = s.Scene()
		}
		if f == *frames/2 {
			mid = s.Scene() // the orbiter on the far side — a clearly different frame
		}
	}

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: 0}, Pitch: -0.12, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	imgA := raytrace.PathRender(raydir.BuildScene(first), cam, *w, *h, opt)
	imgB := raytrace.PathRender(raydir.BuildScene(mid), cam, *w, *h, opt)

	// measure one rendered frame's pixel size (PNG) for the "vs pixels" comparison.
	var buf bytes.Buffer
	_ = png.Encode(&buf, imgA)
	pixelEach := buf.Len()

	fmt.Printf("call of %d frames (%dx%d), %d-object world, one orbiting:\n", *frames, *w, *h, len(base.Objects))
	fmt.Printf("  pixels (PNG/frame ~%d B):   %d B total\n", pixelEach, pixelEach**frames)
	fmt.Printf("  full scene per frame:        %d B total\n", fullEach)
	fmt.Printf("  scene deltas (this stream):  %d B total  (%d keyframe, %d deltas)\n", s.Bytes, s.Keys, s.Frames-s.Keys)
	if s.Bytes > 0 {
		fmt.Printf("  -> %.0fx smaller than full scenes, %.0fx smaller than pixels\n",
			float64(fullEach)/float64(s.Bytes), float64(pixelEach**frames)/float64(s.Bytes))
	}

	writePNG(*out+".png", montage([]panel{{imgA, "frame 0 (keyframe)"}, {imgB, fmt.Sprintf("frame %d (rebuilt from deltas)", *frames/2)}}))
	fmt.Printf("wrote %s.png\n", *out)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	pw := panels[0].img.Bounds().Dx()
	ph := panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	W := len(panels)*pw + (len(panels)-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, ph+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		x := i * (pw + gap)
		draw.Draw(out, image.Rect(x, labelH, x+pw, labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, 3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
	return out
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
