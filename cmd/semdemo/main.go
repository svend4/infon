// Command semdemo shows semantic P-frames: a tiny "video" (a sun drifting across
// a sky grid) sent as one keyframe + per-frame deltas, and the byte savings vs
// sending every full frame. It also verifies the deltas reconstruct exactly.
//
//	go run ./cmd/semdemo
package main

import (
	"fmt"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semframe"
)

const (
	cols = 14
	rows = 8
)

// frame builds a sky grid with a 3-wide sun centered at column sunX.
func frame(sunX int) pseudo.Spec {
	g := make([][]string, rows)
	for y := 0; y < rows; y++ {
		row := make([]string, cols)
		for x := 0; x < cols; x++ {
			switch {
			case y >= rows-2:
				row[x] = "teal"
			case x >= sunX-1 && x <= sunX+1 && y >= 1 && y <= 3:
				row[x] = "gold"
			default:
				row[x] = "navy"
			}
		}
		g[y] = row
	}
	return pseudo.Spec{Format: pseudo.FormatGrid, Cols: 64, Rows: 28, Grid: &pseudo.Grid{Rows: g}}
}

func main() {
	prev := frame(2)
	key := semframe.Bytes(prev)
	fmt.Printf("keyframe: %d bytes (full spec)\n\n", key)

	totalDelta, totalFull, ok := 0, 0, true
	for x := 3; x <= 11; x++ {
		next := frame(x)
		d, isDelta := semframe.Diff(prev, next)
		full := semframe.Bytes(next)
		db := semframe.Bytes(d)
		totalFull += full
		totalDelta += db
		// verify exact reconstruction
		if recon := semframe.Apply(prev, d); !sameGrid(recon, next) {
			ok = false
		}
		fmt.Printf("frame sun@%-2d  P-frame %3d B   (full would be %3d B)   %d cells changed   %s\n",
			x, db, full, len(d.Cells), check(isDelta))
		prev = next
	}
	fmt.Printf("\ntotal: %d B as P-frames vs %d B as full frames  (%.1fx smaller)\n",
		totalDelta, totalFull, float64(totalFull)/float64(max(totalDelta, 1)))
	fmt.Printf("reconstruction exact: %v\n", ok)

	// --- marks P-frames: a bird (◣) flies across a sky ---
	fmt.Println("\nmarks P-frames (a bird flying across the sky):")
	bird := func(bx int) pseudo.Spec {
		const w, h = 16, 5
		rows := make([][]string, h)
		col := make([][]string, h)
		for y := 0; y < h; y++ {
			rows[y] = make([]string, w)
			col[y] = make([]string, w)
		}
		if bx >= 0 && bx+1 < w {
			rows[1][bx], rows[1][bx+1] = "tri-ll", "tri-lr"
			col[1][bx], col[1][bx+1] = "white", "white"
		}
		return pseudo.Spec{Format: pseudo.FormatMarks, Marks: &pseudo.MarkArt{Bg: "navy", Rows: rows, Colors: col}}
	}
	mp := bird(0)
	td, tf := 0, 0
	exact := true
	for x := 1; x <= 12; x++ {
		nx := bird(x)
		d, _ := semframe.DiffMarks(mp, nx)
		td += semframe.Bytes(d)
		tf += semframe.Bytes(nx)
		if r := semframe.ApplyMarks(mp, d); !sameMarks(r, nx) {
			exact = false
		}
		fmt.Printf("  bird@%-2d  P-frame %3d B  (full %3d B)  %d cells\n", x, semframe.Bytes(d), semframe.Bytes(nx), len(d.Cells))
		mp = nx
	}
	fmt.Printf("  total: %d B vs %d B  (%.1fx smaller)  exact=%v\n", td, tf, float64(tf)/float64(max(td, 1)), exact)
}

func sameMarks(a, b pseudo.Spec) bool {
	for y := range a.Marks.Rows {
		for x := range a.Marks.Rows[y] {
			if a.Marks.Rows[y][x] != b.Marks.Rows[y][x] {
				return false
			}
		}
	}
	return true
}

func sameGrid(a, b pseudo.Spec) bool {
	ar, br := a.Grid.Rows, b.Grid.Rows
	if len(ar) != len(br) {
		return false
	}
	for y := range ar {
		for x := range ar[y] {
			if ar[y][x] != br[y][x] {
				return false
			}
		}
	}
	return true
}

func check(ok bool) string {
	if ok {
		return ""
	}
	return "(keyframe needed)"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
