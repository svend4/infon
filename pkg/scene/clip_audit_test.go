package scene

import (
	"math"
	"testing"
	"time"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// renderWithin renders sc, failing the test if it takes longer than two
// seconds: an op's cost must be bounded by the canvas, not by its own size.
func renderWithin(t *testing.T, sc *Scene) *terminal.Frame {
	t.Helper()
	done := make(chan *terminal.Frame, 1)
	go func() { done <- RenderScene(sc) }()
	select {
	case f := <-done:
		return f
	case <-time.After(2 * time.Second):
		t.Fatalf("RenderScene(%+v) did not finish within 2s", sc.Ops)
		return nil
	}
}

func countGlyph(f *terminal.Frame, g rune) int {
	n := 0
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			if f.Blocks[y][x].Glyph == g {
				n++
			}
		}
	}
	return n
}

func TestHugeOpsAreClippedToCanvas(t *testing.T) {
	c := [3]int{9, 9, 9}
	cells := 8 * 4
	cases := []struct {
		name  string
		op    Op
		glyph rune
		want  int // cells carrying glyph
	}{
		{"rect covering", Op{Op: "rect", X: -1 << 40, Y: -1 << 40, W: 1 << 41, H: 1 << 41, Glyph: "#"}, '#', cells},
		{"rect negative width overflow", Op{Op: "rect", X: math.MinInt + 5, Y: 0, W: -10, H: 1, Glyph: "#"}, '#', 0},
		{"rect far away", Op{Op: "rect", X: 1 << 62, Y: 1 << 62, W: math.MaxInt, H: math.MaxInt, Glyph: "#"}, '#', 0},
		{"vgradient covering", Op{Op: "vgradient", X: -1 << 40, Y: -1 << 40, W: 1 << 41, H: 1 << 41, Glyph: "v", From: &c, To: &c}, 'v', cells},
		{"hgradient covering", Op{Op: "hgradient", X: -1 << 40, Y: -1 << 40, W: 1 << 41, H: 1 << 41, Glyph: "h", From: &c, To: &c}, 'h', cells},
		{"disc covering", Op{Op: "disc", CX: 3, CY: 2, R: 1 << 40, Glyph: "O"}, 'O', cells},
		{"disc far away", Op{Op: "disc", CX: 1 << 62, CY: 0, R: 1 << 30, Glyph: "O"}, 'O', 0},
		{"disc max radius", Op{Op: "disc", CX: math.MinInt, CY: math.MaxInt, R: math.MaxInt, Glyph: "O"}, 'O', -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := renderWithin(t, &Scene{Width: 8, Height: 4, Ops: []Op{tc.op}})
			if tc.want < 0 {
				return // only termination is pinned
			}
			if got := countGlyph(f, tc.glyph); got != tc.want {
				t.Errorf("%d cells drawn, want %d", got, tc.want)
			}
		})
	}
}

// Clipping must not change what lands on the canvas for partially visible ops.
func TestPartiallyVisibleOpsMatchReference(t *testing.T) {
	// Disc: a cell is filled iff ((x-cx)/2)^2 + (y-cy)^2 <= r^2.
	cx, cy, r := -1, 2, 3
	f := renderWithin(t, &Scene{Width: 8, Height: 6, Ops: []Op{{Op: "disc", CX: cx, CY: cy, R: r, Glyph: "O"}}})
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			fx, fy := float64(x-cx)*0.5, float64(y-cy)
			want := fx*fx+fy*fy <= float64(r*r)
			if got := f.Blocks[y][x].Glyph == 'O'; got != want {
				t.Errorf("disc cell (%d,%d) filled=%v, want %v", x, y, got, want)
			}
		}
	}

	// Vertical gradient starting two rows above the canvas: row y has
	// t = (y+2)/4, so the blue channel is round(200*t).
	from, to := [3]int{0, 0, 0}, [3]int{0, 0, 200}
	f = renderWithin(t, &Scene{Width: 3, Height: 4, Ops: []Op{{Op: "vgradient", X: -1, Y: -2, W: 3, H: 5, From: &from, To: &to}}})
	for y, b := range []uint8{100, 150, 200} {
		for x := 0; x < 2; x++ {
			if got := f.Blocks[y][x].Bg; got != color.NewRGB(0, 0, b) {
				t.Errorf("vgradient cell (%d,%d) = %+v, want blue %d", x, y, got, b)
			}
		}
		if got := f.Blocks[y][2].Bg; got != color.NewRGB(0, 0, 0) {
			t.Errorf("vgradient drew outside its columns at (2,%d): %+v", y, got)
		}
	}
	if got := f.Blocks[3][0].Bg; got != color.NewRGB(0, 0, 0) {
		t.Errorf("vgradient drew below its rows: %+v", got)
	}

	// Horizontal gradient starting one column left of the canvas: column x has
	// t = (x+1)/3, so the red channel is round(90*t).
	hf, ht := [3]int{0, 0, 0}, [3]int{90, 0, 0}
	f = renderWithin(t, &Scene{Width: 4, Height: 2, Ops: []Op{{Op: "hgradient", X: -1, Y: 1, W: 4, H: 5, From: &hf, To: &ht, Glyph: "h"}}})
	for x, rr := range []uint8{30, 60, 90} {
		if b := f.Blocks[1][x]; b.Glyph != 'h' || b.Fg != color.NewRGB(rr, 0, 0) {
			t.Errorf("hgradient cell (%d,1) = %q %+v, want 'h' red %d", x, b.Glyph, b.Fg, rr)
		}
	}
	if f.Blocks[0][0].Glyph == 'h' || f.Blocks[1][3].Glyph == 'h' {
		t.Error("hgradient drew outside its bounds")
	}

	// Rect hanging off the top-left corner covers exactly cells (0..1, 0..1).
	f = renderWithin(t, &Scene{Width: 4, Height: 4, Ops: []Op{{Op: "rect", X: -2, Y: -1, W: 4, H: 3, Glyph: "#"}}})
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			want := x < 2 && y < 2
			if got := f.Blocks[y][x].Glyph == '#'; got != want {
				t.Errorf("rect cell (%d,%d) filled=%v, want %v", x, y, got, want)
			}
		}
	}
}
