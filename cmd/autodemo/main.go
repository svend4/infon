// Command autodemo shows the auto triangle/quadrant chooser (pkg/automode). It
// rasterizes a diamond (45-degree edges) into a sub-cell mask and renders it two
// ways — quadrant-only (the conservative baseline) and the full alphabet (auto).
// The win shows on EDGE cells (the diagonal boundary), where a triangle matches a
// 45-degree edge that quadrants can only approximate.
//
//	go run ./cmd/autodemo
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/automode"
	"github.com/svend4/infon/pkg/glyphqr"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/microfont"
)

func boundary(mask [][]bool) bool {
	t, f := false, false
	for _, row := range mask {
		for _, v := range row {
			if v {
				t = true
			} else {
				f = true
			}
		}
	}
	return t && f
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	out := flag.String("out", "_glyph", "output directory")
	flag.Parse()
	_ = os.MkdirAll(*out, 0o755)

	const cols, rows, cellPx = 30, 18, 9
	S := glyphset.Sub
	cx, cy := float64(cols*S)/2, float64(rows*S)/2
	inShape := func(px, py float64) bool { return (py - cy) >= (px - cx) } // a 45-degree split

	auto := make([][]string, rows)
	quad := make([][]string, rows)
	var sumA, sumQ, bSumA, bSumQ float64
	var n, bN int
	for gy := 0; gy < rows; gy++ {
		auto[gy] = make([]string, cols)
		quad[gy] = make([]string, cols)
		for gx := 0; gx < cols; gx++ {
			mask := make([][]bool, S)
			for sy := 0; sy < S; sy++ {
				mask[sy] = make([]bool, S)
				for sx := 0; sx < S; sx++ {
					mask[sy][sx] = inShape(float64(gx*S+sx)+0.5, float64(gy*S+sy)+0.5)
				}
			}
			ta, va := automode.BestAmong(mask, automode.Candidates)
			tq, vq := automode.BestAmong(mask, automode.QuadCandidates)
			auto[gy][gx], quad[gy][gx] = ta, tq
			sumA += va
			sumQ += vq
			n++
			if boundary(mask) {
				bSumA += va
				bSumQ += vq
				bN++
			}
		}
	}
	meanA, meanQ := sumA/float64(n), sumQ/float64(n)
	bA, bQ := bSumA/float64(maxi(bN, 1)), bSumQ/float64(maxi(bN, 1))
	fmt.Printf("45-degree split -> %dx%d glyph grid (sub-cell %d), %d edge cells:\n", cols, rows, S, bN)
	fmt.Printf("  overall mean IoU:  quad %.3f   auto %.3f\n", meanQ, meanA)
	fmt.Printf("  EDGE cells  IoU:   quad %.3f   auto %.3f   (+%.1f%% sharper on edges)\n", bQ, bA, (bA/bQ-1)*100)

	imgQ := glyphqr.Render(runes(quad), cellPx)
	imgA := glyphqr.Render(runes(auto), cellPx)
	path := fmt.Sprintf("%s/automode.png", *out)
	if err := save(path, imgQ, imgA, bQ, bA); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (left: quadrant-only, right: auto)\n", path)
}

func runes(tokens [][]string) [][]string {
	out := make([][]string, len(tokens))
	for y, row := range tokens {
		out[y] = make([]string, len(row))
		for x, tok := range row {
			out[y][x] = string(glyphset.Glyph(tok))
		}
	}
	return out
}

func save(path string, imgQ, imgA image.Image, iouQ, iouA float64) error {
	pad, top := 16, 28
	w := imgQ.Bounds().Dx() + imgA.Bounds().Dx() + 3*pad
	h := imgQ.Bounds().Dy() + top + pad
	c := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(c, c.Bounds(), &image.Uniform{color.RGBA{16, 16, 24, 255}}, image.Point{}, draw.Src)
	white := color.RGBA{235, 238, 245, 255}
	x1 := pad
	x2 := pad + imgQ.Bounds().Dx() + pad
	draw.Draw(c, image.Rect(x1, top, x1+imgQ.Bounds().Dx(), top+imgQ.Bounds().Dy()), imgQ, imgQ.Bounds().Min, draw.Over)
	draw.Draw(c, image.Rect(x2, top, x2+imgA.Bounds().Dx(), top+imgA.Bounds().Dy()), imgA, imgA.Bounds().Min, draw.Over)
	microfont.Draw(c, x1, 8, 2, fmt.Sprintf("QUADRANT EDGE %.2f", iouQ), white)
	microfont.Draw(c, x2, 8, 2, fmt.Sprintf("AUTO EDGE %.2f", iouA), white)
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	return png.Encode(fp, c)
}
