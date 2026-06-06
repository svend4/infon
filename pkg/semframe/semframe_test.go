package semframe

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
)

func gridSpec(rows [][]string, palette string) pseudo.Spec {
	return pseudo.Spec{Format: pseudo.FormatGrid, Palette: palette, Grid: &pseudo.Grid{Rows: rows}}
}

func TestDiffApplyRoundTrip(t *testing.T) {
	prev := gridSpec([][]string{{"navy", "navy", "navy"}, {"teal", "teal", "teal"}}, "")
	next := gridSpec([][]string{{"navy", "gold", "navy"}, {"teal", "teal", "teal"}}, "neon")

	d, ok := Diff(prev, next)
	if !ok {
		t.Fatal("Diff not ok")
	}
	if len(d.Cells) != 1 || d.Cells[0] != (CellChange{Y: 0, X: 1, C: "gold"}) {
		t.Fatalf("cells = %+v", d.Cells)
	}
	if d.Palette == nil || *d.Palette != "neon" {
		t.Fatalf("palette delta = %v", d.Palette)
	}
	got := Apply(prev, d)
	if !reflect.DeepEqual(gridOf(got).Rows, gridOf(next).Rows) || got.Palette != "neon" {
		t.Fatalf("reconstruction mismatch: %+v", gridOf(got).Rows)
	}
	if Bytes(d) >= Bytes(next) {
		t.Fatalf("delta (%d) not smaller than full frame (%d)", Bytes(d), Bytes(next))
	}
}

func TestDimMismatchNeedsKeyframe(t *testing.T) {
	a := gridSpec([][]string{{"navy"}}, "")
	b := gridSpec([][]string{{"navy"}, {"teal"}}, "")
	if _, ok := Diff(a, b); ok {
		t.Fatal("dim mismatch should require a keyframe (ok=false)")
	}
}
