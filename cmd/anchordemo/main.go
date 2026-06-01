// Command anchordemo is a hand-made bridge to the MDLM world-model paper
// (Patronus AI, 2026): it shows, on our symbolic grid, why ANY-ORDER decoding
// with ANCHORS beats strict LEFT-TO-RIGHT. The cat is left-right symmetric; we
// freeze its right half as anchors and ask two "decoders" to restore the left.
//
//   - left-to-right reaches the left cells FIRST, before the deciding right
//     anchors are revealed, so it commits them to background -> half a cat
//     (globally inconsistent, exactly the paper's left-to-right bias).
//
//   - any-order sees all anchors at once and fills each left cell as the mirror
//     of its known right partner -> a whole, consistent cat.
//
//     go run ./cmd/anchordemo
package main

import (
	"fmt"

	"github.com/svend4/infon/pkg/tangram"
)

var mir = map[string]string{
	"": "", "full": "full", "tri-ul": "tri-ur", "tri-ur": "tri-ul",
	"tri-ll": "tri-lr", "tri-lr": "tri-ll", "top": "top", "bottom": "bottom",
	"left": "right", "right": "left",
}

func md(d int) int { return tangram.DigitForGlyph(mir[tangram.GlyphForDigit(d)]) }

func figFromGrid(g [][]int, h, w int) tangram.Figure {
	f := tangram.Figure{Name: "r", H: h, W: w, Bg: "navy"}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if g[y][x] != 0 {
				f.Pieces = append(f.Pieces, tangram.Piece{Glyph: tangram.GlyphForDigit(g[y][x]), Y: y, X: x})
			}
		}
	}
	return f
}

func board(g [][]int, h, w int) {
	bar := "   +" + repeat(w) + "+"
	fmt.Println(bar)
	for _, r := range figFromGrid(g, h, w).GlyphRows() {
		fmt.Println("   |" + r + "|")
	}
	fmt.Println(bar)
}
func repeat(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "-"
	}
	return s
}

// violations counts cells that break the mirror rule (left != mirror(right)).
func violations(g [][]int, h, w int) int {
	v := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if g[y][x] != md(g[y][w-1-x]) {
				v++
			}
		}
	}
	return v / 2
}

func main() {
	ps := tangram.PuzzleFromFigure(tangram.Cat())
	h, w := ps.H, ps.W
	target := make([][]int, h)
	for y := 0; y < h; y++ {
		target[y] = make([]int, w)
		for x := 0; x < w; x++ {
			target[y][x] = tangram.DigitForGlyph(ps.Target[y][x])
		}
	}
	center := w / 2 // right half + center are the anchors
	// Make the target exactly grid-symmetric so the mirror rule is clean.
	for y := 0; y < h; y++ {
		for x := 0; x < center; x++ {
			target[y][x] = md(target[y][w-1-x])
		}
	}

	fmt.Println("TARGET (symmetric cat) — right half + center are frozen ANCHORS:")
	board(target, h, w)

	// left-to-right: left cells (x<center) are reached before the right anchors
	// exist as context, so they default to background.
	ar := make([][]int, h)
	for y := 0; y < h; y++ {
		ar[y] = make([]int, w)
		for x := 0; x < w; x++ {
			if x >= center {
				ar[y][x] = target[y][x]
			}
		}
	}
	// any-order: anchors known globally; fill each left cell as mirror of its
	// right partner (the deciding anchor).
	dif := make([][]int, h)
	for y := 0; y < h; y++ {
		dif[y] = make([]int, w)
		for x := 0; x < w; x++ {
			if x >= center {
				dif[y][x] = target[y][x]
			} else {
				dif[y][x] = md(target[y][w-1-x])
			}
		}
	}

	tps := tangram.PuzzleFromFigure(figFromGrid(target, h, w))
	score := func(g [][]int) (int, float64) {
		return violations(g, h, w), tangram.ScorePuzzle(tps, figFromGrid(g, h, w)).IoU
	}
	av, ai := score(ar)
	dv, di := score(dif)

	fmt.Println("\n[1] LEFT-TO-RIGHT decode (causal, can't see the right anchors in time):")
	board(ar, h, w)
	fmt.Printf("    mirror-violations=%d   IoU-to-target=%.0f%%  -> globally inconsistent (half a cat)\n", av, ai*100)

	fmt.Println("\n[2] ANY-ORDER decode with anchors (MDLM-style: see the whole, fill in any order):")
	board(dif, h, w)
	fmt.Printf("    mirror-violations=%d   IoU-to-target=%.0f%% -> globally consistent (whole cat)\n", dv, di*100)

	fmt.Println("\nbridge: the paper's 'anchors + bidirectional denoising' = our 'frozen cells +")
	fmt.Println("any-order fill under a global rule'. Same win, on a symbolic grid, by hand.")
}
