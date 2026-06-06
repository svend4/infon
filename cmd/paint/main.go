// Command paint shows the on-device painter (pkg/painter): with no model and no
// network it reads a prompt into a pseudo-image grid, which the deterministic
// pseudo-diffusion renderer turns into a painterly picture. It cycles several
// prompts into a GIF.
//
//	go run ./cmd/paint
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"

	"github.com/svend4/infon/pkg/painter"
)

func main() {
	out := flag.String("out", "_pseudo", "output directory")
	flag.Parse()
	_ = os.MkdirAll(*out, 0o755)

	prompts := []string{
		"a calm harbor at sunset",
		"a forest under a starry night",
		"snowy mountains at dawn",
		"a desert at noon",
		"a stormy sea",
		"a green meadow valley",
	}
	const pxW, pxH = 480, 270

	g := &gif.GIF{}
	fmt.Println("on-device painter (no model, no network) — prompt -> pseudo-image:")
	for _, p := range prompts {
		spec := painter.Paint(p, 64, 32)
		img, err := spec.Image(pxW, pxH)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q: %v\n", p, err)
			continue
		}
		fmt.Printf("  painted %q\n", p)
		pi := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, img.Bounds(), img, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 140)
	}
	path := fmt.Sprintf("%s/paint.gif", *out)
	fp, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer fp.Close()
	if err := gif.EncodeAll(fp, g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d scenes)\n", path, len(g.Image))
}
