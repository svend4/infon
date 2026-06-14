// Command framecast measures the adaptive frame codec (pkg/framecodec) on REAL encoded
// terminal frames — the exact bytes internal/network.EncodeFrame puts on the wire. It
// runs three motion regimes (a still scene with a small moving shape, a smooth
// full-frame plasma, and full random noise) through the codec, verifies every frame
// round-trips losslessly, and reports how little crosses the wire vs raw — DELTA wins
// the still scene, ZLIB the smooth and even the noisy one (real frames keep their
// x/y/glyph positional structure, so ZLIB never loses to RAW here; RAW is the
// guaranteed floor, exercised by the codec's own tests). It draws a bytes-per-frame
// chart per regime (bars coloured by mode, against the raw baseline).
//
//	go run ./cmd/framecast -cols 100 -rows 40 -frames 40
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"

	"github.com/svend4/infon/internal/network"
	pcolor "github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/framecodec"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	var (
		out    = flag.String("out", "framecast", "output basename")
		cols   = flag.Int("cols", 100, "frame width in cells")
		rows   = flag.Int("rows", 40, "frame height in cells")
		frames = flag.Int("frames", 40, "frames per regime")
	)
	flag.Parse()

	regimes := []struct {
		name  string
		build func(f int, rng *rand.Rand) *terminal.Frame
	}{
		{"still (moving shape)", func(f int, _ *rand.Rand) *terminal.Frame { return stillFrame(*cols, *rows, f) }},
		{"smooth (plasma)", func(f int, _ *rand.Rand) *terminal.Frame { return plasmaFrame(*cols, *rows, f) }},
		{"noise (random)", func(f int, rng *rand.Rand) *terminal.Frame { return noiseFrame(*cols, *rows, rng) }},
	}

	var panels []panel
	fmt.Printf("framecast: adaptive RAW/ZLIB/DELTA on real EncodeFrame bytes (%d×%d, %d frames)\n", *cols, *rows, *frames)
	fmt.Printf("  %-22s %8s %8s %7s   modes (raw/zlib/delta)\n", "regime", "raw", "coded", "saved")
	for _, rg := range regimes {
		rng := rand.New(rand.NewSource(1))
		var st framecodec.Stream
		var dec framecodec.Decoder
		perFrame := make([]int, 0, *frames)
		modes := make([]byte, 0, *frames)
		rawSize := 0
		for f := 0; f < *frames; f++ {
			raw, err := network.EncodeFrame(rg.build(f, rng))
			if err != nil {
				fmt.Fprintln(os.Stderr, "encode frame:", err)
				os.Exit(1)
			}
			rawSize = len(raw)
			enc := st.Push(raw)
			got, derr := dec.Push(enc)
			if derr != nil || !bytes.Equal(got, raw) {
				fmt.Fprintf(os.Stderr, "round trip broke in %q at frame %d\n", rg.name, f)
				os.Exit(1)
			}
			perFrame = append(perFrame, len(enc))
			modes = append(modes, enc[0])
		}
		fmt.Printf("  %-22s %7dB %7dB %6.0f%%   %d/%d/%d\n",
			rg.name, st.RawBytes, st.Bytes, st.Savings()*100, st.Raw, st.Zlib, st.Delta)
		panels = append(panels, panel{drawBars(perFrame, modes, rawSize), fmt.Sprintf("%s — saved %.0f%%", rg.name, st.Savings()*100)})
	}

	writePNG(*out+".png", vstack(panels))
	fmt.Printf("  legend: blue=raw  amber=zlib  green=delta  (grey line = raw frame size)\n")
	fmt.Printf("wrote %s.png\n", *out)
}

// stillFrame: a static gradient background with a small bright square that moves —
// most cells are identical frame-to-frame, so DELTA wins.
func stillFrame(cols, rows, f int) *terminal.Frame {
	fr := terminal.NewFrame(cols, rows)
	sx := (f * 2) % maxi(cols-6, 1)
	sy := f % maxi(rows-6, 1)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			r, g, b := uint8(x*255/cols), uint8(y*255/rows), uint8(40)
			if x >= sx && x < sx+6 && y >= sy && y < sy+6 {
				r, g, b = 255, 240, 120
			}
			fr.Blocks[y][x] = terminal.Block{Glyph: '█', Fg: pcolor.RGB{R: r, G: g, B: b}, Bg: pcolor.RGB{R: r / 3, G: g / 3, B: b / 3}}
		}
	}
	return fr
}

// plasmaFrame: a smooth full-frame field that moves everywhere — no delta, but very
// compressible, so ZLIB wins.
func plasmaFrame(cols, rows, f int) *terminal.Frame {
	fr := terminal.NewFrame(cols, rows)
	t := float64(f)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			v := math.Sin(float64(x)/4+t/3) + math.Sin(float64(y)/3-t/4) + math.Sin(float64(x+y)/5+t/2)
			n := (v + 3) / 6 // 0..1
			r := uint8(80 + 175*n)
			g := uint8(60 + 120*math.Abs(math.Sin(t/5)))
			b := uint8(255 - 175*n)
			fr.Blocks[y][x] = terminal.Block{Glyph: '█', Fg: pcolor.RGB{R: r, G: g, B: b}, Bg: pcolor.RGB{R: r / 3, G: g / 3, B: b / 3}}
		}
	}
	return fr
}

// noiseFrame: full random every cell every frame — the worst case; even so the frame
// keeps its x/y/glyph structure, so ZLIB still trims a little (RAW is the floor).
func noiseFrame(cols, rows int, rng *rand.Rand) *terminal.Frame {
	fr := terminal.NewFrame(cols, rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			fr.Blocks[y][x] = terminal.Block{
				Glyph: rune(0x2580 + rng.Intn(0x20)),
				Fg:    pcolor.RGB{R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: uint8(rng.Intn(256))},
				Bg:    pcolor.RGB{R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: uint8(rng.Intn(256))},
			}
		}
	}
	return fr
}

func modeColor(m byte) color.RGBA {
	switch m {
	case framecodec.ModeZlib:
		return color.RGBA{R: 230, G: 180, B: 70, A: 255} // amber
	case framecodec.ModeDelta:
		return color.RGBA{R: 90, G: 210, B: 130, A: 255} // green
	default:
		return color.RGBA{R: 90, G: 140, B: 220, A: 255} // blue (raw)
	}
}

// drawBars draws bytes-per-frame as bars coloured by mode, with the raw frame size as
// a baseline line at the top.
func drawBars(perFrame []int, modes []byte, rawSize int) image.Image {
	const h, pad = 120, 10
	bw := 8
	if len(perFrame)*bw > 480 {
		bw = maxi(480/len(perFrame), 2)
	}
	w := len(perFrame)*bw + 2*pad
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	scale := float64(h-2*pad) / float64(maxi(rawSize, 1))
	base := h - pad
	// raw baseline (what every frame would cost uncompressed)
	ry := base - int(float64(rawSize)*scale)
	for x := pad; x < w-pad; x++ {
		img.SetRGBA(x, ry, color.RGBA{R: 120, G: 120, B: 130, A: 255})
	}
	for i, n := range perFrame {
		bh := int(float64(n) * scale)
		x0 := pad + i*bw
		fillRect(img, x0, base-bh, bw-1, bh, modeColor(modes[i]))
	}
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if image.Pt(i, j).In(img.Bounds()) {
				img.SetRGBA(i, j, c)
			}
		}
	}
}

type panel struct {
	img   image.Image
	label string
}

func vstack(panels []panel) image.Image {
	const gap, labelH = 6, 14
	w := 0
	for _, p := range panels {
		if d := p.img.Bounds().Dx(); d > w {
			w = d
		}
	}
	h := 0
	for _, p := range panels {
		h += labelH + p.img.Bounds().Dy() + gap
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	white := color.RGBA{R: 230, G: 230, B: 235, A: 255}
	y := 0
	for _, p := range panels {
		microfont.Draw(out, 4, y+3, 1, p.label, white)
		draw.Draw(out, image.Rect(0, y+labelH, p.img.Bounds().Dx(), y+labelH+p.img.Bounds().Dy()), p.img, p.img.Bounds().Min, draw.Src)
		y += labelH + p.img.Bounds().Dy() + gap
	}
	return out
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
