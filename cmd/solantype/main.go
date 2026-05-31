// Command solantype shows two ways to set the myFront4Solan4 font: glyphs printed
// ADJACENT fuse their box borders into one continuous woven ornament/line, while
// a SPACE between them separates the letters into legible text. The same string,
// two faces — decorative vs readable — chosen by spacing.
//
//	go run ./cmd/solantype [outdir]
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/terminal"
)

var (
	navy = color.RGB{R: 18, G: 22, B: 50}
	gold = color.RGB{R: 245, G: 205, B: 90}
	teal = color.RGB{R: 70, G: 200, B: 180}
)

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

func line(text string, fg, bg color.RGB) *terminal.Frame {
	r := []rune(text)
	f := terminal.NewFrame(len(r), 1)
	f.Fill(' ', bg, bg)
	f.DrawText(0, 0, text, fg, bg)
	return f
}

func block(lines []string, fg, bg color.RGB) *terminal.Frame {
	w := 0
	for _, l := range lines {
		if n := len([]rune(l)); n > w {
			w = n
		}
	}
	f := terminal.NewFrame(w, len(lines))
	f.Fill(' ', bg, bg)
	for i, l := range lines {
		f.DrawText(0, i, l, fg, bg)
	}
	return f
}

func main() {
	out := "./_solan"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		panic(err)
	}

	// (1) adjacent -> borders fuse into one woven band
	save(filepath.Join(out, "a_merged.png"), glyphset.Rasterize(line("SOLANFONT", gold, navy), 26))
	// (2) spaced -> legible letters
	save(filepath.Join(out, "b_spaced.png"), glyphset.Rasterize(line("S O L A N  F O N T", gold, navy), 26))
	// (3) a tight multi-row block -> an ornamental tapestry from text
	save(filepath.Join(out, "c_ornament.png"), glyphset.Rasterize(block([]string{
		"SVENDSVEND", "VENDSVENDS", "ENDSVENDSV", "NDSVENDSVE", "DSVENDSVEN",
	}, teal, navy), 22))
	// (4) the 16x16 pattern tiles (codes 34..42) as fills
	save(filepath.Join(out, "d_tiles.png"), glyphset.Rasterize(line("\"#$%&'()*", gold, navy), 30))
	// (5) "draw something": a framed word — a box of text around a spaced title
	save(filepath.Join(out, "e_frame.png"), glyphset.Rasterize(block([]string{
		"OOOOOOOOOO",
		"O T V C P O",
		"O  A I   O",
		"OOOOOOOOOO",
	}, gold, navy), 24))

	fmt.Printf("wrote a_merged/b_spaced/c_ornament/d_tiles/e_frame to %s\n", out)
}
