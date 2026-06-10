// Command rayvoice shows voice prosody steering the world's mood (the
// audio/reactive -> mood combination): three synthetic voice profiles — quiet,
// loud-and-steady, loud-and-swinging — are fed to a world's mood sense, and the
// world authored under the induced mood is rendered. A calm voice makes a quiet,
// intimate world; a loud steady one a vast, open world; an expressive swinging one
// a strange, varied world. It connects three existing pipes: audio loudness, the
// mood reader, and the director.
//
//	go run ./cmd/rayvoice
package main

import (
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
		out = flag.String("out", "voice", "output basename")
		w   = flag.Int("w", 380, "panel width")
		h   = flag.Int("h", 250, "panel height")
		spp = flag.Int("spp", 48, "samples per pixel")
	)
	flag.Parse()

	profiles := []struct {
		name string
		loud func(i int) float64
	}{
		{"quiet", func(int) float64 { return 0.2 }},
		{"loud, steady", func(int) float64 { return 0.9 }},
		{"loud, swinging", func(i int) float64 {
			if i%2 == 0 {
				return 0.3
			}
			return 0.85
		}},
	}

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	var panels []panel
	for _, p := range profiles {
		world := raydir.NewWorld()
		world.SetMoodSensing(true)
		for i := 0; i < 140; i++ {
			world.ObserveVoice(frame(p.loud(i)), 0.05)
		}
		mood := world.MoodName()
		prompt := world.BiasPrompt("a place of forms")
		scene, _, err := raydir.AuthorScene(brain.Local{}, prompt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "author:", err)
			os.Exit(1)
		}
		img := raytrace.PathRender(scene, cam, *w, *h, opt)
		fmt.Printf("%-16s voice -> mood %-9s -> %q\n", p.name, mood, prompt)
		panels = append(panels, panel{img, p.name + " -> " + mood})
	}
	writePNG(*out+".png", montage(panels))
	fmt.Printf("wrote %s.png\n", *out)
}

// frame makes a PCM frame whose RMS matches loud (0..1) — alternating +/- so the
// RMS is the amplitude.
func frame(loud float64) []int16 {
	v := int16(loud * 6000)
	s := make([]int16, 512)
	for i := range s {
		if i%2 == 0 {
			s[i] = v
		} else {
			s[i] = -v
		}
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
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
