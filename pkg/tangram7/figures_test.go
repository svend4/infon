package tangram7

import (
	"math"
	"sort"
	"testing"
)

func polyEdges(poly Poly) []float64 {
	n := len(poly)
	e := make([]float64, n)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		e[i] = math.Round(math.Hypot(poly[j].X-poly[i].X, poly[j].Y-poly[i].Y)*1000) / 1000
	}
	sort.Float64s(e)
	return e
}

// pieceKind infers the canonical piece type from area + sorted edge lengths.
func pieceKind(poly Poly) string {
	a := math.Round(polyArea(poly)*100) / 100
	e := polyEdges(poly)
	switch {
	case a == 4 && len(e) == 3:
		return "large"
	case a == 2 && len(e) == 3:
		return "medium"
	case a == 1 && len(e) == 3:
		return "small"
	case a == 2 && len(e) == 4 && e[0] == e[3]:
		return "square"
	case a == 2 && len(e) == 4:
		return "parallelogram"
	}
	return "?"
}

// Every figure must use exactly the seven pieces, each congruent to its base,
// tiling area 16 with no overlap.
func TestFiguresValid(t *testing.T) {
	want := map[string]int{"large": 2, "medium": 1, "small": 2, "square": 1, "parallelogram": 1}
	for _, name := range FigureNames() {
		f := Figures()[name]
		rep := Verify(f, 300)
		if rep.Pieces != 7 {
			t.Errorf("%s: pieces=%d, want 7", name, rep.Pieces)
		}
		if math.Abs(rep.Area-16) > 1e-6 {
			t.Errorf("%s: area=%.4f, want 16", name, rep.Area)
		}
		if rep.Overlap > 0.02 {
			t.Errorf("%s: overlap=%.4f, want <=0.02", name, rep.Overlap)
		}
		got := map[string]int{}
		for _, pc := range f {
			got[pieceKind(pc.Shape)]++
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s: piece %q count=%d, want %d (got %v)", name, k, got[k], v, got)
			}
		}
		if got["?"] != 0 {
			t.Errorf("%s: has %d unrecognised pieces (%v)", name, got["?"], got)
		}
	}
}

func TestFiguresRender(t *testing.T) {
	for _, name := range FigureNames() {
		if img := Render(Figures()[name], 120, 120); img == nil {
			t.Fatalf("%s: nil image", name)
		}
	}
}
