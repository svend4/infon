// Command glyphqrdemo shows the glyph-QR with Reed-Solomon error correction:
// encode a message into a glyph grid, damage several cells, and watch it decode
// back to the exact text — the self-correcting "crossword x QR" idea, applied to
// our glyph alphabet.
//
//	go run ./cmd/glyphqrdemo [outdir]
package main

import (
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/svend4/infon/pkg/glyphqr"
)

func main() {
	out := "./_glyphqr"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	_ = os.MkdirAll(out, 0o755)

	msg := "TVCP-AI gate address: CAT"
	ecc := 14 // corrects up to 7 byte errors
	grid, err := glyphqr.EncodeText(msg, ecc)
	if err != nil {
		fail(err)
	}
	fmt.Printf("message: %q   ecc=%d (corrects up to %d cell-errors)\n\n", msg, ecc, ecc/2)
	fmt.Println("glyph-QR:")
	printGrid(grid)
	savePNG(filepath.Join(out, "code.png"), glyphqr.Render(grid, 22))

	// Damage cells WITHIN the correction budget.
	dmg := corrupt(grid, ecc/2, 1)
	fmt.Printf("\ndamaged %d cells (within budget):\n", ecc/2)
	printGrid(dmg)
	savePNG(filepath.Join(out, "damaged.png"), glyphqr.Render(dmg, 22))
	got, err := glyphqr.DecodeText(dmg)
	fmt.Printf("decode -> %q   err=%v   %s\n", got, err, mark(got == msg && err == nil))

	// Damage BEYOND the budget to show the bound.
	over := corrupt(grid, ecc/2+3, 2)
	got2, err2 := glyphqr.DecodeText(over)
	fmt.Printf("\ndamaged %d cells (beyond budget): decode -> %q err=%v   %s\n",
		ecc/2+3, got2, err2, mark(got2 != msg || err2 != nil))

	fmt.Printf("\nrenders in %s (code.png, damaged.png)\n", out)
}

// corrupt flips n distinct codeword bytes (one hi-nibble cell each).
func corrupt(grid [][]string, n, seed int) [][]string {
	g := make([][]string, len(grid))
	for i := range grid {
		g[i] = append([]string(nil), grid[i]...)
	}
	w := len(g[0])
	r := rand.New(rand.NewSource(int64(seed)))
	alts := []string{"█", "◤", "◥", "◣", "◢", "▌", "▐", "□"}
	for b := 0; b < n; b++ {
		idx := 12 + 2*b
		y, x := idx/w, idx%w
		if y >= len(g) {
			break
		}
		cur := g[y][x]
		alt := cur
		for alt == cur {
			alt = alts[r.Intn(len(alts))]
		}
		g[y][x] = alt
	}
	return g
}

func printGrid(grid [][]string) {
	for _, r := range glyphqr.Rows(grid) {
		fmt.Println("   " + r)
	}
}
func mark(ok bool) string {
	if ok {
		return "[OK]"
	}
	return "[as expected: not recovered]"
}
func savePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fail(err)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, "glyphqrdemo:", err); os.Exit(1) }

var _ = strings.TrimSpace
