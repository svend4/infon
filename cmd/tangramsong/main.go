// Command tangramsong is the third face of the project's "see, hear, summon"
// triad: it reads a tangram figure cell by cell (the same row-major reading that
// gives the figure its NUMBER), turns each cell into a note via pkg/sona, and
// writes BOTH a .wav you can hear and a .gif you can watch — the playhead sweeps
// the very cells you hear, while a piano-roll of the melody grows beneath. The
// caption names the creature this figure would SUMMON, closing the triad.
//
//	go run ./cmd/tangramsong                 # cat -> _song/cat.wav + _song/cat.gif
//	go run ./cmd/tangramsong -fig swan
//	go run ./cmd/tangramsong -all            # the whole summon roster
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
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/sona"
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	fig := flag.String("fig", "cat", "figure name (see the tangram catalog)")
	all := flag.Bool("all", false, "render the whole summon roster")
	out := flag.String("out", "_song", "output directory")
	stepMs := flag.Int("step", 160, "milliseconds per cell (note)")
	rate := flag.Int("rate", sona.DefaultRate, "audio sample rate (Hz)")
	tile := flag.Int("tile", 300, "figure pixel size in the GIF")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	names := []string{*fig}
	if *all {
		names = []string{"cat", "fox", "turtle", "owl", "swan", "crab"}
	}
	step := time.Duration(*stepMs) * time.Millisecond
	for _, name := range names {
		f, ok := tangram.Named(name)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown figure %q (skipped)\n", name)
			continue
		}
		if err := sonify(f, name, *out, step, *rate, *tile); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		}
	}
}

func sonify(f tangram.Figure, name, out string, step time.Duration, rate, tile int) error {
	notes := sona.MelodyWithStep(f, step)
	pcm := sona.Synth(notes, rate)
	wav := sona.WAV(pcm, rate)
	wavPath := fmt.Sprintf("%s/%s.wav", out, name)
	if err := os.WriteFile(wavPath, wav, 0o644); err != nil {
		return err
	}
	gifPath := fmt.Sprintf("%s/%s.gif", out, name)
	if err := writeGIF(gifPath, f, notes, tile); err != nil {
		return err
	}

	m := tangram.Classify(f)
	_, summoned, ok := arena.Summon(f, 0, 0, 0)
	verdict := "object — no creature"
	if ok {
		verdict = "summons " + summoned
	}
	num := f.Number().Text(16)
	if len(num) > 14 {
		num = num[:14] + "…"
	}
	voiced := 0
	for _, n := range notes {
		if !n.Rest {
			voiced++
		}
	}
	dur := time.Duration(len(notes)) * step
	fmt.Printf("%-7s %dx%d  №16=%s  gate=%s  class=%s(%.2f)  %d notes (%d voiced, %s)  -> %s\n",
		name, f.H, f.W, num, f.GateCode(7), m.Name, m.Score, len(notes), voiced, dur, verdict)
	fmt.Printf("        see+hear: %s  |  %s\n", gifPath, wavPath)
	return nil
}

// ---- GIF: see what you hear ----------------------------------------------

const (
	gW, gH = 760, 380
)

func writeGIF(path string, f tangram.Figure, notes []sona.Note, tile int) error {
	figImg := f.Image(tile, tile)
	g := &gif.GIF{}
	steps := len(notes)
	// a couple of trailing hold frames so the last note lingers
	for idx := 0; idx < steps+3; idx++ {
		cur := idx
		if cur >= steps {
			cur = steps - 1
		}
		frame := drawFrame(figImg, f, notes, cur)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		delay := int(notes[cur].Dur.Milliseconds() / 10)
		if delay < 4 {
			delay = 4
		}
		if idx >= steps {
			delay = 60
		}
		g.Delay = append(g.Delay, delay)
	}
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	return gif.EncodeAll(fp, g)
}

func drawFrame(figImg image.Image, f tangram.Figure, notes []sona.Note, cur int) image.Image {
	c := image.NewRGBA(image.Rect(0, 0, gW, gH))
	fillRect(c, 0, 0, gW, gH, color.RGBA{14, 14, 22, 255})

	// left: the figure itself ("see")
	fx, fy := 24, 40
	tw := figImg.Bounds().Dx()
	fillRect(c, fx-6, fy-6, tw+12, tw+12, color.RGBA{30, 30, 44, 255})
	draw.Draw(c, image.Rect(fx, fy, fx+tw, fy+figImg.Bounds().Dy()), figImg, figImg.Bounds().Min, draw.Over)

	// right top: the cell grid, playhead on the cell now sounding ("hear = see")
	gx, gy, gw2, gh2 := 372, 40, 364, 196
	cell := gw2 / f.W
	if ch := gh2 / f.H; ch < cell {
		cell = ch
	}
	if cell < 1 {
		cell = 1
	}
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			idx := y*f.W + x
			d := notes[idx].Digit
			col := color.RGBA{38, 38, 52, 255}
			if d != 0 {
				col = pitchColor(d)
			}
			px, py := gx+x*cell, gy+y*cell
			fillRect(c, px, py, cell-2, cell-2, col)
			if idx == cur {
				outline(c, px-2, py-2, cell+2, cell+2, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	// right bottom: the melody as a growing piano-roll ("hear")
	rx, ry, rw, rh := 372, 256, 364, 100
	fillRect(c, rx, ry, rw, rh, color.RGBA{22, 22, 32, 255})
	bw := rw / len(notes)
	if bw < 2 {
		bw = 2
	}
	const loMidi, hiMidi = 57, 90
	for j := 0; j <= cur && j < len(notes); j++ {
		n := notes[j]
		bx := rx + j*bw
		if n.Rest {
			fillRect(c, bx, ry+rh-3, bw-1, 2, color.RGBA{70, 70, 86, 255})
			continue
		}
		frac := float64(n.Midi-loMidi) / float64(hiMidi-loMidi)
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		bh := 6 + int(frac*float64(rh-10))
		col := pitchColor(n.Digit)
		if j == cur {
			col = color.RGBA{255, 255, 255, 255}
		}
		fillRect(c, bx, ry+rh-bh, bw-1, bh, col)
	}
	return c
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func outline(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	fillRect(img, x, y, w, 2, c)
	fillRect(img, x, y+h-2, w, 2, c)
	fillRect(img, x, y, 2, h, c)
	fillRect(img, x+w-2, y, 2, h, c)
}

// pitchColor maps a cell digit (1..15) to a hue, so each glyph has its own color.
func pitchColor(d int) color.RGBA {
	return hsv(float64(d)/16.0*300.0, 0.65, 1.0)
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
