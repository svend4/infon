// Command rayclimate visualises the hexagram-cellular-automaton climate (block:
// hexca -> weather). The Q6 automaton is a regional weather map — each cell's
// yang-count picks snow / fog / rain — so weather varies by place, deterministically
// (every peer in a shared world derives the same map from the seed, nothing to sync;
// the future is a forecast you compute by stepping the automaton). It writes a
// side-by-side of the yang-count grid and the weather zones, and prints a walk's
// weather, a forecast, and a determinism check.
//
//	go run ./cmd/rayclimate -seed 42
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
)

func main() {
	var (
		out  = flag.String("out", "climate", "output basename")
		seed = flag.Int64("seed", 42, "climate automaton seed")
		px   = flag.Int("px", 40, "pixels per cell")
	)
	flag.Parse()

	c := raydir.NewClimate(*seed)
	const scale = 12.0

	// the weather-zone map: each cell coloured by the kind it yields.
	wcol := map[string]color.RGBA{
		"snow": {R: 235, G: 238, B: 245, A: 255},
		"fog":  {R: 150, G: 155, B: 165, A: 255},
		"rain": {R: 70, G: 110, B: 180, A: 255},
	}
	zoneMap := image.NewRGBA(image.Rect(0, 0, 8*(*px), 8*(*px)))
	counts := map[string]int{}
	for cz := 0; cz < 8; cz++ {
		for cx := 0; cx < 8; cx++ {
			k := c.KindAt(float64(cx)*scale+1, float64(cz)*scale+1, scale)
			counts[k]++
			fill(zoneMap, cx*(*px), cz*(*px), *px, *px, wcol[k])
		}
	}
	yang := c.Render(*px)

	sheet := montage([]panel{{yang, "yang-count (dark->gold)"}, {zoneMap, "weather zones (snow/fog/rain)"}})
	writePNG(*out+".png", sheet)

	// text: zone histogram, a southbound walk, a forecast, and a determinism check.
	fmt.Printf("climate seed %d — zones: snow=%d fog=%d rain=%d\n", *seed, counts["snow"], counts["fog"], counts["rain"])
	var walk []string
	prev := ""
	for z := 0; z < 8; z++ {
		k := c.KindAt(0, float64(z)*scale+1, scale)
		if k != prev {
			walk = append(walk, fmt.Sprintf("z%d:%s", z*int(scale), k))
			prev = k
		}
	}
	fmt.Printf("walk south at x=0: %s\n", strings.Join(walk, " -> "))
	fmt.Printf("global forecast (mean-yang season), next 6 steps: %s\n", strings.Join(c.Forecast(6), " "))
	same := raydir.NewClimate(*seed).KindAt(0, 0, scale) == c.KindAt(0, 0, scale)
	fmt.Printf("deterministic (same seed reproduces): %v\n", same)
	fmt.Printf("wrote %s.png\n", *out)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	pw := panels[0].img.Bounds().Dx()
	ph := panels[0].img.Bounds().Dy()
	const gap, labelH = 10, 16
	W := len(panels)*pw + (len(panels)-1)*gap
	H := ph + labelH
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		x := i * (pw + gap)
		draw.Draw(out, image.Rect(x, labelH, x+pw, labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, 3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
	return out
}

func fill(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			img.SetRGBA(i, j, c)
		}
	}
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
