// Command glyphqrscan demonstrates the optical glyph-QR: a framed code with
// finder marks in three corners decodes from ANY rotation. It prints the code,
// presents it at 0/90/180/270 degrees (each auto-oriented and decoded), shows a
// rotated+corrupted code still recovered (finders orient, Reed-Solomon repairs),
// and renders a GIF of the four orientations all decoding to the same payload.
//
//	go run ./cmd/glyphqrscan
//	go run ./cmd/glyphqrscan -msg "gate://swan#7" -ecc 10
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"

	"github.com/svend4/infon/pkg/glyphqr"
)

func main() {
	msg := flag.String("msg", "gate://cat#A5", "payload to encode")
	ecc := flag.Int("ecc", 8, "Reed-Solomon parity bytes")
	cell := flag.Int("cell", 18, "pixels per glyph cell in the GIF")
	out := flag.String("out", "_glyphqr", "output directory")
	flag.Parse()
	_ = os.MkdirAll(*out, 0o755)

	framed, err := glyphqr.EncodeTextFramed(*msg, *ecc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("optical glyph-QR for %q (ecc=%d) — finders in 3 corners, blank in the 4th:\n", *msg, *ecc)
	for _, r := range glyphqr.Rows(framed) {
		fmt.Println("  " + r)
	}

	fmt.Println("\npresented at each rotation (no prior key — orientation from finders):")
	for k := 0; k < 4; k++ {
		g := glyphqr.RotateCW(framed, k)
		rot, ok := glyphqr.Orient(g)
		txt, derr := glyphqr.DecodeTextFramed(g)
		status := "ok"
		if derr != nil {
			status = derr.Error()
		}
		fmt.Printf("  %3d deg  -> +%d turns to upright, found=%v  -> %q (%s)\n", k*90, rot, ok, txt, status)
	}

	// rotated AND corrupted: finders orient, RS repairs the interior.
	bad := glyphqr.RotateCW(framed, 1)
	hits := 0
	for y := 2; y < len(bad)-2 && hits < 2; y++ {
		bad[y][2] = framed[0][0] // overwrite an interior cell with the finder glyph
		hits++
	}
	txt, derr := glyphqr.DecodeTextFramed(bad)
	fmt.Printf("\nrotated 90 deg + %d interior errors -> %q (orient+RS recovered: %v)\n", hits, txt, derr == nil)

	gifPath := fmt.Sprintf("%s/glyphqrscan.gif", *out)
	if err := writeGIF(gifPath, framed, *cell); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\nwrote %s (one code, four orientations, all decoded)\n", gifPath)
}

func writeGIF(path string, framed [][]string, cell int) error {
	const cv = 400
	g := &gif.GIF{}
	for rep := 0; rep < 4+2; rep++ {
		k := rep
		if k >= 4 {
			k = 3
		}
		grid := glyphqr.RotateCW(framed, k)
		_, derr := glyphqr.DecodeTextFramed(grid)
		img := glyphqr.Render(grid, cell)

		c := image.NewRGBA(image.Rect(0, 0, cv, cv))
		fillRect(c, 0, 0, cv, cv, color.RGBA{14, 14, 22, 255})
		iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
		ox, oy := (cv-iw)/2, (cv-ih)/2
		draw.Draw(c, image.Rect(ox, oy, ox+iw, oy+ih), img, img.Bounds().Min, draw.Over)
		border := color.RGBA{90, 220, 120, 255} // green: decoded
		if derr != nil {
			border = color.RGBA{220, 80, 80, 255}
		}
		outline(c, ox-7, oy-7, iw+14, ih+14, border)

		pi := image.NewPaletted(c.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, c.Bounds(), c, image.Point{})
		g.Image = append(g.Image, pi)
		d := 90
		if rep >= 4 {
			d = 140
		}
		g.Delay = append(g.Delay, d)
	}
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	return gif.EncodeAll(fp, g)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func outline(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	fillRect(img, x, y, w, 3, c)
	fillRect(img, x, y+h-3, w, 3, c)
	fillRect(img, x, y, 3, h, c)
	fillRect(img, x+w-3, y, 3, h, c)
}
