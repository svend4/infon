// Command raytour renders the Gray-code grand tour of all 64 hexagrams as cinema
// (Block I: tour of every world). The tour visits every hexagram once, changing a
// single line at each step, so neighbours differ by one trait. By default it lays the
// 64 worlds out as an 8×8 atlas in Gray-snake order (a periodic table of every
// possible world, each labelled and tinted by its mood). With -morph it instead
// renders a strip that morphs smoothly from world to world, one trait at a time.
//
//	go run ./cmd/raytour                 # the 8x8 atlas of all 64 worlds
//	go run ./cmd/raytour -morph 6 -steps 7 # a smooth morph along the first 7 steps
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

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func moodColor(m string) color.RGBA {
	switch m {
	case "joyful":
		return color.RGBA{R: 240, G: 200, B: 80, A: 255}
	case "serene":
		return color.RGBA{R: 120, G: 210, B: 140, A: 255}
	case "ominous":
		return color.RGBA{R: 220, G: 90, B: 80, A: 255}
	case "somber":
		return color.RGBA{R: 110, G: 150, B: 220, A: 255}
	default:
		return color.RGBA{R: 200, G: 200, B: 205, A: 255}
	}
}

func main() {
	var (
		out   = flag.String("out", "tour", "output basename")
		start = flag.String("start", "000000", "hexagram the tour starts from")
		morph = flag.Int("morph", 0, "if >0, render a morph strip with this many frames per Gray step")
		steps = flag.Int("steps", 7, "Gray steps to morph through (with -morph)")
		tw    = flag.Int("tw", 110, "thumbnail width")
		th    = flag.Int("th", 84, "thumbnail height")
		spp   = flag.Int("spp", 28, "samples per pixel")
	)
	flag.Parse()

	h0, ok := raydir.ParseHexagram(*start)
	if !ok {
		h0 = raydir.Hexagram{}
	}
	walk := h0.GrayWalk()

	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 4, NEE: true, MIS: true, Sobol: true}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	render := func(v raydir.SceneVector) image.Image {
		return raytrace.PostProcess(raytrace.PathRender(raydir.BuildScene(v.SceneSpec()), cam, *tw, *th, opt), 1.0, 0.9, 0.4)
	}

	if *morph > 0 {
		if *steps < 1 {
			*steps = 1
		}
		if *steps > len(walk)-1 {
			*steps = len(walk) - 1
		}
		path := raydir.TourMorph(walk[:*steps+1], *morph)
		strip := image.NewRGBA(image.Rect(0, 0, len(path)*(*tw+2), *th+16))
		draw.Draw(strip, strip.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
		for i, v := range path {
			img := render(v)
			x := i * (*tw + 2)
			draw.Draw(strip, image.Rect(x, 16, x+*tw, 16+*th), img, img.Bounds().Min, draw.Src)
			if i%*morph == 0 { // label each Gray corner
				microfont.Draw(strip, x+2, 3, 1, fmt.Sprintf("%06b", v.Hexagram().Number()), moodColor(v.Mood()))
			}
		}
		writePNG(*out+"_morph.png", strip)
		fmt.Printf("morphed %d Gray steps in %d frames; wrote %s_morph.png\n", *steps, len(path), *out)
		return
	}

	// the 8×8 atlas of all 64 worlds, in Gray-snake order so neighbours differ by one
	// trait both across and down.
	const labelH = 12
	cw, chh := *tw, *th+labelH
	atlas := image.NewRGBA(image.Rect(0, 0, 8*cw, 8*chh))
	draw.Draw(atlas, atlas.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	moods := map[string]int{}
	for k, hx := range walk {
		v := raydir.VectorFromHexagram(hx)
		moods[v.Mood()]++
		row := k / 8
		col := k % 8
		if row%2 == 1 { // boustrophedon: the tour snakes, so cells stay one-trait apart
			col = 7 - col
		}
		x, y := col*cw, row*chh
		img := render(v)
		draw.Draw(atlas, image.Rect(x, y+labelH, x+*tw, y+labelH+*th), img, img.Bounds().Min, draw.Src)
		microfont.Draw(atlas, x+2, y+2, 1, fmt.Sprintf("%06b", hx.Number()), moodColor(v.Mood()))
	}
	writePNG(*out+".png", atlas)
	fmt.Printf("atlas of all 64 worlds (Gray tour from %06b)\n", h0.Number())
	fmt.Printf("  moods across the space: joyful=%d serene=%d neutral=%d somber=%d ominous=%d\n",
		moods["joyful"], moods["serene"], moods["neutral"], moods["somber"], moods["ominous"])
	fmt.Printf("wrote %s.png\n", *out)
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
