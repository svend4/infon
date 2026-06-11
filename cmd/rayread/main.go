// Command rayread points the inverse reader at the real world (Block G: reader ×
// vision). Block C's reader recovers meaning from a scene the director authored;
// rayread recovers it from raw PIXELS — a photo or camera frame: warmth from the
// red/blue balance, sun from luminance, fog from washed-out colour and flat contrast,
// glow from highlights, and (with a vision model at VISION_API_URL) density and scale
// from detected objects. From the pixels it names the world's mood and the hexagram
// it implies, and re-authors that hexagram's Q6 world beside the original — meaning
// read back out of an image.
//
//	go run ./cmd/rayread                 # read three rendered "photos" (self-check)
//	go run ./cmd/rayread -img photo.png  # read a real image (VISION_API_URL optional)
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // decode JPEG photos passed via -img
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/internal/vision"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

type shot struct {
	name string
	img  image.Image
	dets []vision.Detection
	tru  *raydir.SceneVector // ground truth, when we rendered it ourselves
}

func main() {
	var (
		out  = flag.String("out", "read", "output basename")
		imgF = flag.String("img", "", "read this image file instead of the built-in photos")
		w    = flag.Int("w", 300, "panel width")
		h    = flag.Int("h", 220, "panel height")
		spp  = flag.Int("spp", 56, "samples per pixel")
	)
	flag.Parse()

	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	renderVec := func(v raydir.SceneVector) image.Image {
		return raytrace.PostProcess(raytrace.PathRender(raydir.BuildScene(v.SceneSpec()), cam, *w, *h, opt), 1.0, 0.9, 0.4)
	}

	var shots []shot
	if *imgF != "" {
		f, err := os.Open(*imgF)
		if err != nil {
			fmt.Fprintln(os.Stderr, "open:", err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		im, _, err := image.Decode(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "decode:", err)
			os.Exit(1)
		}
		var dets []vision.Detection
		if an, ok := vision.NewHTTPAnalyzerFromEnv(); ok && an != nil {
			if d, e := an.Analyze(context.Background(), im); e == nil {
				dets = d
			}
		}
		shots = append(shots, shot{name: *imgF, img: im, dets: dets})
	} else {
		// three rendered "photos" with distinct character: the reader gets only pixels.
		photos := []struct {
			name string
			v    raydir.SceneVector
		}{
			{"warm dusk", raydir.SceneVector{0.10, 0.88, 0.5, 0.55, 0.6, 0.45}},
			{"cold fog", raydir.SceneVector{0.90, 0.18, 0.4, 0.30, 0.4, 0.05}},
			{"vivid noon", raydir.SceneVector{0.02, 0.62, 0.8, 0.95, 0.6, 0.72}},
		}
		for _, p := range photos {
			v := p.v
			shots = append(shots, shot{name: p.name, img: renderVec(v), tru: &v})
		}
	}

	axes := []string{"fog", "warm", "dens", "sun", "scale", "glow"}
	var panels []panel
	for _, s := range shots {
		read := raydir.ReadImage(s.img, s.dets)
		fmt.Printf("%-12s -> mood %-8s hexagram %06b (%s)\n", s.name, read.Mood(), read.Hexagram().Number(), read.Hexagram().Name())
		if s.tru != nil {
			fmt.Printf("   axis     %s\n", joinAxes(axes))
			fmt.Printf("   true     %s\n", joinVec(*s.tru))
			fmt.Printf("   read     %s\n", joinVec(read))
		}
		// the photo, labelled with the reader's verdict, beside the re-authored Q6 world.
		reauthored := renderVec(read)
		panels = append(panels,
			panel{s.img, s.name},
			panel{reauthored, fmt.Sprintf("read: %s %06b", read.Mood(), read.Hexagram().Number())},
		)
	}
	writePNG(*out+".png", montage(panels))
	fmt.Printf("wrote %s.png\n", *out)
}

func joinAxes(a []string) string {
	s := ""
	for _, x := range a {
		s += fmt.Sprintf("%-6s", x)
	}
	return s
}

func joinVec(v raydir.SceneVector) string {
	s := ""
	for _, x := range v {
		s += fmt.Sprintf("%-6.2f", x)
	}
	return s
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
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
