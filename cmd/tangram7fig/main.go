// Command tangram7fig renders the authored tangram7 figures (square, cat, swan)
// to PNGs and a combined menagerie sheet, printing each figure's verification.
package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	t "github.com/svend4/infon/pkg/tangram7"
)

func save(name string, img image.Image) {
	fp, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	if err := png.Encode(fp, img); err != nil {
		panic(err)
	}
}

func main() {
	out := "_t7fig"
	_ = os.MkdirAll(out, 0o755)
	const cell = 360
	names := t.FigureNames()
	sheet := image.NewRGBA(image.Rect(0, 0, cell*len(names), cell))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)

	fmt.Println("figure    pieces  area   overlap")
	for i, name := range names {
		f := t.Figures()[name]
		rep := t.Verify(f, 300)
		fmt.Printf("%-8s  %4d   %5.2f   %.4f\n", name, rep.Pieces, rep.Area, rep.Overlap)
		img := t.Render(f, cell, cell)
		save(fmt.Sprintf("%s/%s.png", out, name), img)
		draw.Draw(sheet, image.Rect(i*cell, 0, (i+1)*cell, cell), img, image.Point{}, draw.Src)
	}
	save(out+"/menagerie.png", sheet)
	fmt.Printf("wrote %d figures + menagerie.png to %s/\n", len(names), out)
}
