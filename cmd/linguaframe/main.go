// Command linguaframe builds a self-describing "Lingua Cosmica" frame for a
// tangram figure and then decodes it back using nothing but the bytes — the
// receiver learns to count, then learns the radix, the grid, the cells and the
// checksum, all from the stream itself (pkg/lincos). It prints the frame, the
// step-by-step self-decoding, and renders a GIF where the bit-ribbon lights up
// layer by layer and the reconstructed figure appears once the checksum agrees.
//
//	go run ./cmd/linguaframe                 # cat -> _lincos/cat.gif
//	go run ./cmd/linguaframe -fig swan
//	go run ./cmd/linguaframe -flip 60        # corrupt bit 60 to show self-check
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math"
	"os"

	"github.com/svend4/infon/pkg/lincos"
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	figName := flag.String("fig", "cat", "figure name (tangram catalog)")
	flipBit := flag.Int("flip", -1, "flip this bit before decoding (demo the self-check); -1 = none")
	out := flag.String("out", "_lincos", "output directory")
	tile := flag.Int("tile", 220, "reconstructed figure pixel size in the GIF")
	flag.Parse()

	f, ok := tangram.Named(*figName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown figure %q\n", *figName)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	frame := lincos.Encode(f)
	fmt.Printf("figure %q  %dx%d  №16=%s\n", *figName, f.H, f.W, short(f.Number().Text(16)))
	fmt.Printf("self-describing frame: %d bytes\n  %s\n", len(frame), hexd(frame, 48))
	fmt.Print(lincos.Explain(frame))

	got, L, err := lincos.Decode(frame)
	switch {
	case err != nil:
		fmt.Printf("decode error: %v\n", err)
	case got.Number().Cmp(f.Number()) == 0:
		fmt.Printf("round-trip: reconstructed figure is identical (no external key used)\n")
	default:
		fmt.Printf("round-trip: MISMATCH\n")
	}

	if *flipBit >= 0 && *flipBit < len(frame)*8 {
		bad := append([]byte(nil), frame...)
		bad[*flipBit/8] ^= 1 << uint(7-*flipBit%8)
		_, L2, err2 := lincos.Decode(bad)
		fmt.Printf("\nflip bit %d -> checksum %s (%v)\n", *flipBit, okword(L2.OK), err2)
	}

	gifPath := fmt.Sprintf("%s/%s.gif", *out, *figName)
	if err := writeGIF(gifPath, frame, L, got, *tile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\nwrote %s\n", gifPath)
}

func okword(ok bool) string {
	if ok {
		return "OK"
	}
	return "CAUGHT the corruption"
}
func short(s string) string {
	if len(s) > 16 {
		return s[:16] + "…"
	}
	return s
}
func hexd(b []byte, n int) string {
	if len(b) < n {
		n = len(b)
	}
	s := fmt.Sprintf("%x", b[:n])
	if len(b) > n {
		s += " …"
	}
	return s
}

// ---- GIF: a frame that teaches itself -------------------------------------

const cW, cH = 760, 430

func writeGIF(path string, frame []byte, L lincos.Lesson, fig tangram.Figure, tile int) error {
	// layer bit-boundaries, mirroring lincos.Encode's layout.
	b0 := 9                       // preamble 1,2,3
	b1 := b0 + L.Radix + 1        // radix tally
	b2 := b1 + L.H + 1            // height tally
	b3 := b2 + L.W + 1            // width tally
	b4 := b3 + len(L.Cells)*L.Bits // payload
	b5 := L.BitLen                 // + checksum tally
	payStart := b3

	layerColor := func(i int) color.RGBA {
		switch {
		case i < b0:
			return color.RGBA{60, 200, 200, 255} // teal: learn to count
		case i < b1:
			return color.RGBA{240, 200, 60, 255} // gold: radix
		case i < b2:
			return color.RGBA{245, 150, 50, 255} // orange: height
		case i < b3:
			return color.RGBA{245, 110, 40, 255} // deep orange: width
		case i < b4:
			cell := (i - payStart) / L.Bits
			d := 0
			if cell >= 0 && cell < len(L.Cells) {
				d = L.Cells[cell]
			}
			if d == 0 {
				return color.RGBA{90, 90, 110, 255}
			}
			return hsv(float64(d)/16.0*300.0, 0.6, 1.0) // payload: per-digit hue
		default:
			return color.RGBA{80, 220, 110, 255} // green: checksum
		}
	}

	// reveal stages: count, +radix, +dims, +payload, +checksum, then show figure.
	reveals := []int{b0, b1, b3, b4, b5, b5, b5, b5}
	showFig := []bool{false, false, false, false, false, true, true, true}
	figImg := fig.Image(tile, tile)

	g := &gif.GIF{}
	for k := range reveals {
		fr := drawFrame(frame, L, reveals[k], layerColor, showFig[k], figImg)
		pi := image.NewPaletted(fr.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, fr.Bounds(), fr, image.Point{})
		g.Image = append(g.Image, pi)
		d := 80
		if k >= 5 {
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

func drawFrame(frame []byte, L lincos.Lesson, reveal int, layerColor func(int) color.RGBA, showFig bool, figImg image.Image) image.Image {
	c := image.NewRGBA(image.Rect(0, 0, cW, cH))
	fillRect(c, 0, 0, cW, cH, color.RGBA{14, 14, 22, 255})

	// bit ribbon on the left: each bit a square, colored by layer, bright if 1.
	const cols, cell = 26, 16
	rx, ry := 24, 40
	for i := 0; i < L.BitLen; i++ {
		col := rx + (i%cols)*cell
		row := ry + (i/cols)*cell
		base := layerColor(i)
		var c2 color.RGBA
		if i >= reveal {
			c2 = color.RGBA{26, 26, 36, 255} // not taught yet
		} else if bitAt(frame, i) == 1 {
			c2 = base
		} else {
			c2 = dim(base, 0.30)
		}
		fillRect(c, col, row, cell-2, cell-2, c2)
	}

	// legend (always visible): the five lessons, in order.
	leg := []color.RGBA{{60, 200, 200, 255}, {240, 200, 60, 255}, {245, 130, 45, 255}, {120, 120, 200, 255}, {80, 220, 110, 255}}
	lx, ly := 24, cH-34
	for i, lc := range leg {
		fillRect(c, lx+i*54, ly, 46, 14, lc)
	}

	// right: the reconstructed figure, once the checksum has agreed.
	if showFig {
		fw := figImg.Bounds().Dx()
		fx, fy := cW-fw-40, 64
		border := color.RGBA{80, 220, 110, 255}
		if !L.OK {
			border = color.RGBA{220, 80, 80, 255}
		}
		fillRect(c, fx-8, fy-8, fw+16, fw+16, border)
		fillRect(c, fx-3, fy-3, fw+6, fw+6, color.RGBA{20, 20, 30, 255})
		draw.Draw(c, image.Rect(fx, fy, fx+fw, fy+figImg.Bounds().Dy()), figImg, figImg.Bounds().Min, draw.Over)
	}
	return c
}

func bitAt(b []byte, i int) int {
	if i/8 >= len(b) {
		return 0
	}
	return int(b[i/8]>>uint(7-i%8)) & 1
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func dim(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), 255}
}

func hsv(h, s, v float64) color.RGBA {
	cc := v * s
	x := cc * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	mm := v - cc
	var r, g, b float64
	switch int(h/60.0) % 6 {
	case 0:
		r, g, b = cc, x, 0
	case 1:
		r, g, b = x, cc, 0
	case 2:
		r, g, b = 0, cc, x
	case 3:
		r, g, b = 0, x, cc
	case 4:
		r, g, b = x, 0, cc
	default:
		r, g, b = cc, 0, x
	}
	return color.RGBA{uint8((r + mm) * 255), uint8((g + mm) * 255), uint8((b + mm) * 255), 255}
}
