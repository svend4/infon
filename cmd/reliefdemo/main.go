// Command reliefdemo renders tangram grid figures as 2.5D isometric reliefs
// (path 2): each glyph cell is extruded into a block, the diagonal glyphs
// becoming lit facets. It writes a sheet of animals, a hero still and a "rise"
// GIF, and prints the per-figure height-spec size (one byte per cell).
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/relief"
	"github.com/svend4/infon/pkg/tangram"
)

func savePNG(name string, img image.Image) {
	fp, _ := os.Create(name)
	defer fp.Close()
	_ = png.Encode(fp, img)
}

func main() {
	out := "_relief"
	_ = os.MkdirAll(out, 0o755)

	names := []string{"cat", "fox", "owl", "turtle"}
	const cell = 300
	sheet := image.NewRGBA(image.Rect(0, 0, 2*cell, 2*cell))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)
	for i, n := range names {
		f, _ := tangram.Named(n)
		img := relief.Render(f, relief.Gradient(f, 70, 150), cell, cell)
		cx, cy := (i%2)*cell, (i/2)*cell
		draw.Draw(sheet, image.Rect(cx, cy, cx+cell, cy+cell), img, image.Point{}, draw.Src)
		fmt.Printf("%-8s  cells=%d  height-spec=%d bytes\n", n, len(f.Pieces), relief.SpecBytes(f))
	}
	savePNG(out+"/animals_relief.png", sheet)

	cat := tangram.Cat()
	savePNG(out+"/cat_relief.png", relief.Render(cat, relief.Uniform(cat, 120), 460, 460))

	// rise GIF: heights ramp from nearly flat to full
	g := &gif.GIF{}
	const N = 14
	for k := 0; k <= N; k++ {
		t := (1 - math.Cos(math.Pi*float64(k)/float64(N))) / 2
		b := uint8(8 + t*120)
		frame := relief.Render(cat, relief.Uniform(cat, b), 380, 380)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 7)
	}
	for k := N; k >= 0; k-- {
		t := (1 - math.Cos(math.Pi*float64(k)/float64(N))) / 2
		b := uint8(8 + t*120)
		frame := relief.Render(cat, relief.Uniform(cat, b), 380, 380)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 7)
	}
	fp, _ := os.Create(out + "/cat_rise.gif")
	_ = gif.EncodeAll(fp, g)
	fp.Close()
	fmt.Printf("wrote sheet, hero still and rise GIF to %s/\n", out)
}
