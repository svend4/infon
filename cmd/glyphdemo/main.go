// Command glyphdemo renders the digitized alphabet chart, and compares the
// quadrant codec with the new diagonal/triangle codec on the same image.
//
//	go run ./cmd/glyphdemo [outdir]
package main

import (
	"fmt"
	"image"
	stdcolor "image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/pseudo"
)

// source: a navy sky, a gold sun, and a white mountain slope — lots of diagonals.
func source(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	navy := stdcolor.RGBA{24, 30, 70, 255}
	gold := stdcolor.RGBA{245, 205, 90, 255}
	white := stdcolor.RGBA{235, 238, 245, 255}
	cx, cy, r := float64(w)*0.7, float64(h)*0.35, float64(h)*0.22
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := navy
			// mountain: a big triangle rising to an apex (diagonal edges)
			apex := float64(w) * 0.35
			base := float64(h) * 0.95
			height := float64(h) * 0.6
			if float64(y) > base-height*(1-math.Abs(float64(x)-apex)/(float64(w)*0.45)) {
				c = white
			}
			// sun disc
			if dx, dy := float64(x)-cx, float64(y)-cy; dx*dx+dy*dy <= r*r {
				c = gold
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// marksScene composes a mountain (triangle edges) and a diamond sun from the
// marks alphabet — the kind of picture a text model can now emit as format=marks.
func marksScene(cols, rows int) pseudo.Spec {
	g := make([][]string, rows)
	col := make([][]string, rows)
	cx := cols / 2
	for y := 0; y < rows; y++ {
		g[y] = make([]string, cols)
		col[y] = make([]string, cols)
		half := y
		for x := 0; x < cols; x++ {
			switch {
			case y == rows-1:
				g[y][x], col[y][x] = "full", "slate"
			case x == cx-half-1:
				g[y][x], col[y][x] = "tri-ul", "white"
			case x == cx+half:
				g[y][x], col[y][x] = "tri-ur", "white"
			case x > cx-half-1 && x < cx+half:
				g[y][x], col[y][x] = "full", "white"
			}
		}
	}
	sx := cols * 3 / 4
	if rows >= 3 && sx+1 < cols {
		g[0][sx], g[0][sx+1] = "tri-lr", "tri-ll"
		g[1][sx], g[1][sx+1] = "tri-ur", "tri-ul"
		col[0][sx], col[0][sx+1], col[1][sx], col[1][sx+1] = "gold", "gold", "gold", "gold"
	}
	return pseudo.Spec{Format: pseudo.FormatMarks, Marks: &pseudo.MarkArt{Bg: "navy", Fg: "white", Rows: g, Colors: col}}
}

func main() {
	out := "./_glyph"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		panic(err)
	}
	const cols, rows, cell = 56, 28, 12
	src := source(320, 160)

	q := babe.ImageToFrame(src, cols, rows)         // existing quadrant codec
	tr := glyphset.RenderTriangles(src, cols, rows) // new diagonal/triangle codec
	save(filepath.Join(out, "a_quadrant.png"), glyphset.Rasterize(q, cell))
	save(filepath.Join(out, "b_triangles.png"), glyphset.Rasterize(tr, cell))
	save(filepath.Join(out, "c_alphabet.png"), glyphset.Rasterize(glyphset.Chart(8), 22))
	if mf, err := marksScene(24, 9).Frame(); err == nil {
		save(filepath.Join(out, "d_marks.png"), glyphset.Rasterize(mf, 16))
	}

	fmt.Println("alphabet (digitized from the sheet):")
	for i, m := range glyphset.Marks {
		fmt.Printf("  %-8s %c", m.Name, m.Glyph)
		if i%4 == 3 {
			fmt.Println()
		}
	}
	fmt.Printf("\nwrote a_quadrant.png, b_triangles.png, c_alphabet.png to %s\n", out)
}
