// Command tangramgallery renders our figures as a framed "gallery wall": the
// catalog plus many procedural Creature(seed) figures, each in a wooden frame on
// a warm wall — the kind of geometric gallery the project naturally generates,
// where every framed piece is a figure with its own number/address.
//
//	go run ./cmd/tangramgallery [out.png]
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	out := "./_gallery/wall.png"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	_ = os.MkdirAll(dir(out), 0o755)

	figs := []tangram.Figure{tangram.Cat(), tangram.Swan(), tangram.Boat(), tangram.Tree(), tangram.House()}
	for s := int64(1); s <= 35; s++ {
		figs = append(figs, tangram.Creature(s))
	}

	const cols, tile, pad, mat, frame = 8, 140, 22, 9, 6
	rows := (len(figs) + cols - 1) / cols
	cellW := tile + 2*(mat+frame)
	cellH := cellW
	W := cols*cellW + (cols+1)*pad
	H := rows*cellH + (rows+1)*pad

	wall := image.NewRGBA(image.Rect(0, 0, W, H))
	fillRect(wall, 0, 0, W, H, color.RGBA{224, 212, 190, 255}) // warm wall
	wood := color.RGBA{150, 112, 70, 255}
	matc := color.RGBA{246, 244, 239, 255}

	for i, f := range figs {
		cx := pad + (i%cols)*(cellW+pad)
		cy := pad + (i/cols)*(cellH+pad)
		fillRect(wall, cx, cy, cellW, cellH, wood)                             // frame
		fillRect(wall, cx+frame, cy+frame, cellW-2*frame, cellH-2*frame, matc) // mat
		fi := f.Image(tile, tile)                                              // artwork
		draw.Draw(wall, image.Rect(cx+frame+mat, cy+frame+mat, cx+frame+mat+tile, cy+frame+mat+tile), fi, image.Point{}, draw.Over)
	}

	save(out, wall)
	fmt.Printf("gallery: %d framed figures (%d catalog + 35 creatures) -> %s\n", len(figs), 5, out)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}
func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func dir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}
