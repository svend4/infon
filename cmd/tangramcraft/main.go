// Command tangramcraft is a terminal "gate" prototype: you build a figure from
// the glyph ring while the AI (a pose-invariant classifier) names what you are
// becoming and reads your address; on completion the gate "locks" onto a known
// creature. It also shows that dialing an unknown address usually does NOT lock
// — the meaningful figures are sparse, exactly like Stargate addresses.
//
//	go run ./cmd/tangramcraft [-build cat] [-dial N]
package main

import (
	"flag"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"strings"

	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	build := flag.String("build", "cat", "catalog figure to assemble at the gate")
	dial := flag.Int64("dial", -1, "dial a raw address number instead of building")
	flag.Parse()

	fmt.Println("=== TANGRAM-CRAFT : the glyph gate ===")
	fmt.Println("ring (the address alphabet):", strings.Join(ring(), " "))
	fmt.Println()

	if *dial >= 0 {
		dialAddress(big.NewInt(*dial), 8, 9)
		return
	}

	target, ok := tangram.Named(*build)
	if !ok {
		fmt.Printf("unknown figure %q\n", *build)
		return
	}
	gateBuild(target)

	fmt.Println("\n--- dialing known vs uncharted addresses ---")
	fmt.Println("\n[dial the cat's own number] ->")
	dialAddress(target.Number(), target.H, target.W)
	fmt.Println("\n[dial a random address] ->")
	dialAddress(randomAddress(target.H, target.W), target.H, target.W)
}

func ring() []string {
	var r []string
	for d := 1; d < tangram.AlphabetSize; d++ {
		r = append(r, glyphOf(d))
	}
	return r
}

func glyphOf(d int) string {
	f := tangram.FromNumber(big.NewInt(int64(d)), 1, 1)
	rows := f.GlyphRows()
	if len(rows) == 1 && rows[0] != "" {
		return rows[0]
	}
	return "?"
}

func sortedPieces(ps []tangram.Piece) []tangram.Piece {
	out := append([]tangram.Piece(nil), ps...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].Glyph < out[j].Glyph
	})
	return out
}

func prefix(f tangram.Figure, ps []tangram.Piece, k int) tangram.Figure {
	return tangram.Figure{Name: "build", H: f.H, W: f.W, Bg: f.Bg,
		Pieces: append([]tangram.Piece(nil), ps[:k]...)}
}

func board(f tangram.Figure) {
	bar := "   +" + strings.Repeat("-", f.W) + "+"
	fmt.Println(bar)
	for _, r := range f.GlyphRows() {
		fmt.Println("   |" + r + "|")
	}
	fmt.Println(bar)
}

func gateBuild(target tangram.Figure) {
	ps := sortedPieces(target.Pieces)
	n := len(ps)
	puzzle := tangram.PuzzleFromFigure(target)
	steps := map[int]bool{1: true, n / 4: true, n / 2: true, 3 * n / 4: true, n: true}
	fmt.Printf("assembling \"%s\" (%d pieces) — the AI narrates each step:\n", target.Name, n)
	for k := 1; k <= n; k++ {
		if !steps[k] {
			continue
		}
		p := prefix(target, ps, k)
		m := tangram.Classify(p)
		sc := tangram.ScorePuzzle(puzzle, p)
		fmt.Printf("\nstep %2d/%d:\n", k, n)
		board(p)
		fmt.Printf("   AI: reads as \"%s\" (%.0f%%) | toward %s IoU=%.0f%% | address %d glyphs\n",
			m.Name, m.Score*100, target.Name, sc.IoU*100, len([]rune(p.Address())))
	}
	final := target
	fmt.Printf("\n>>> GATE LOCKED — you arrived at: %s\n", strings.ToUpper(final.Name))
	fmt.Println("    address :", final.Address())
	fmt.Println("    gatecode:", final.GateCode(7), "(short quick-dial)")
	num := final.Number().String()
	fmt.Println("    number  :", short(num))
}

func dialAddress(n *big.Int, h, w int) {
	f := tangram.FromNumber(n, h, w)
	board(f)
	m := tangram.Classify(f)
	if m.Score >= 0.85 { // an address "locks" only onto a recognizable destination
		fmt.Printf("   >>> GATE LOCKED -> arrived at %s (%.0f%%)\n", strings.ToUpper(m.Name), m.Score*100)
	} else {
		fmt.Printf("   uncharted space - nearest is %s but only %.0f%%; the gate stays dark\n", m.Name, m.Score*100)
	}
}

func randomAddress(h, w int) *big.Int {
	rng := rand.New(rand.NewSource(42))
	n := new(big.Int)
	base := big.NewInt(int64(tangram.AlphabetSize))
	for i := 0; i < h*w; i++ {
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(rng.Intn(tangram.AlphabetSize))))
	}
	return n
}

func short(s string) string {
	if len(s) <= 24 {
		return s
	}
	return s[:10] + "..." + s[len(s)-10:]
}
