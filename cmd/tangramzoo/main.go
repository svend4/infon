// Command tangramzoo renders the sixteen-figure tangram address book to a sheet
// and prints each figure's address and canonical id.
package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	order := []string{"cat", "swan", "boat", "tree", "house", "dog", "fox", "rabbit",
		"fish", "bird", "duck", "owl", "butterfly", "crab", "turtle", "snake"}
	cat := tangram.Catalog()
	const cell = 180
	cols := 4
	rows := (len(order) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cell, rows*cell))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)

	fmt.Printf("%-10s  %-12s  %s\n", "figure", "address", "canonical")
	for i, name := range order {
		f := cat[name]
		cn := f.CanonicalNumber().String()
		short := cn
		if len(short) > 16 {
			short = short[:16] + "..."
		}
		fmt.Printf("%-10s  %-12s  %s\n", name, f.Address(), short)
		img := f.Image(cell-12, cell-12)
		cx, cy := (i%cols)*cell+6, (i/cols)*cell+6
		draw.Draw(sheet, image.Rect(cx, cy, cx+cell-12, cy+cell-12), img, image.Point{}, draw.Src)
	}
	os.MkdirAll("_zoo", 0o755)
	fp, _ := os.Create("_zoo/addressbook.png")
	png.Encode(fp, sheet)
	fp.Close()
	fmt.Printf("wrote %d figures to _zoo/addressbook.png\n", len(order))
}
