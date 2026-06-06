// Command areciboframe renders the full Lingua-Cosmica plaque (pkg/arecibo): one
// frame that teaches its own alphabet. It decodes the frame and draws (a) the
// LEGEND — every symbol's shape as learned from the bytes — and (b) the figure
// reconstructed using ONLY that learned legend (no appeal to the glyph library),
// proving the message is decodable with no shared culture.
//
//	go run ./cmd/areciboframe -fig cat
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/arecibo"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	figName := flag.String("fig", "cat", "figure name")
	out := flag.String("out", "_lincos", "output directory")
	flag.Parse()
	_ = os.MkdirAll(*out, 0o755)

	f, ok := tangram.Named(*figName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown figure %q\n", *figName)
		os.Exit(1)
	}
	frame := arecibo.Encode(f)
	m, fig, err := arecibo.Decode(frame)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	num := fig.Number().Text(16)
	if len(num) > 12 {
		num = num[:12] + "…"
	}
	fmt.Printf("arecibo frame for %q: %d bytes, radix %d, %dx%d, number %s, legend of %d symbols, checksum %v\n",
		*figName, len(frame), m.Radix, m.H, m.W, num, m.Radix, m.OK)

	img := render(m)
	path := fmt.Sprintf("%s/arecibo_%s.png", *out, *figName)
	fp, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer fp.Close()
	if err := png.Encode(fp, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (top: alphabet legend; below: figure drawn from the legend alone)\n", path)
}

func render(m arecibo.Message) image.Image {
	bg := color.RGBA{16, 16, 24, 255}
	ink := color.RGBA{230, 235, 245, 255}
	gold := color.RGBA{240, 200, 90, 255}
	teal := color.RGBA{90, 200, 200, 255}
	green := color.RGBA{90, 220, 120, 255}

	const (
		legSub  = 3 // px per sub-cell in the legend swatches
		legCols = 8
		figSub  = 3 // px per sub-cell in the figure render
	)
	swatch := m.S * legSub
	legRows := (m.Radix + legCols - 1) / legCols
	legW := legCols*(swatch+10) + 20
	legH := 24 + legRows*(swatch+16)

	cell := m.S * figSub
	figW := m.W * cell
	figH := m.H * cell

	W := legW
	if figW+40 > W {
		W = figW + 40
	}
	H := legH + 28 + figH + 40
	c := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(c, c.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	microfont.Draw(c, 16, 8, 2, "ALPHABET LEGEND", gold)
	// legend: every symbol's shape as learned from the frame, labelled by index.
	for d := 0; d < m.Radix; d++ {
		col := d % legCols
		row := d / legCols
		ox := 20 + col*(swatch+10)
		oy := 28 + row*(swatch+16)
		fillRect(c, ox-1, oy-1, swatch+2, swatch+2, color.RGBA{40, 40, 56, 255})
		drawBitmap(c, ox, oy, m, d, legSub, teal)
		microfont.Draw(c, ox, oy+swatch+2, 1, fmt.Sprintf("%d", d), ink)
	}

	fy := legH + 8
	microfont.Draw(c, 16, fy, 2, "FIGURE FROM LEGEND", gold)
	fx0, fy0 := 20, fy+20
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			d := m.Cells[y*m.W+x]
			drawCellFromLegend(c, fx0+x*cell, fy0+y*cell, m, d, figSub, ink)
		}
	}
	outline(c, fx0-3, fy0-3, figW+6, figH+6, green)
	return c
}

func drawBitmap(c *image.RGBA, ox, oy int, m arecibo.Message, d, sub int, fg color.RGBA) {
	for sy := 0; sy < m.S; sy++ {
		for sx := 0; sx < m.S; sx++ {
			if m.LegendBit(d, sx, sy) {
				fillRect(c, ox+sx*sub, oy+sy*sub, sub, sub, fg)
			}
		}
	}
}

func drawCellFromLegend(c *image.RGBA, ox, oy int, m arecibo.Message, d, sub int, fg color.RGBA) {
	for sy := 0; sy < m.S; sy++ {
		for sx := 0; sx < m.S; sx++ {
			if m.LegendBit(d, sx, sy) {
				fillRect(c, ox+sx*sub, oy+sy*sub, sub, sub, fg)
			}
		}
	}
}

func fillRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{col}, image.Point{}, draw.Src)
}

func outline(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	fillRect(img, x, y, w, 2, col)
	fillRect(img, x, y+h-2, w, 2, col)
	fillRect(img, x, y, 2, h, col)
	fillRect(img, x+w-2, y, 2, h, col)
}
