package semframe

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
)

func marksSpec(rows, cols [][]string) pseudo.Spec {
	return pseudo.Spec{Format: pseudo.FormatMarks, Marks: &pseudo.MarkArt{Rows: rows, Colors: cols}}
}

func TestMarksDiffApply(t *testing.T) {
	prev := marksSpec(
		[][]string{{"", "tri-ll", ""}, {"full", "full", "full"}},
		[][]string{{"", "white", ""}, {"slate", "slate", "slate"}},
	)
	next := marksSpec(
		[][]string{{"", "", "tri-ll"}, {"full", "full", "full"}},
		[][]string{{"", "", "gold"}, {"slate", "slate", "slate"}},
	)
	d, ok := DiffMarks(prev, next)
	if !ok {
		t.Fatal("DiffMarks not ok")
	}
	// two cells changed: (0,1) cleared, (0,2) set to tri-ll/gold
	if len(d.Cells) != 2 {
		t.Fatalf("cells = %d, want 2: %+v", len(d.Cells), d.Cells)
	}
	got := ApplyMarks(prev, d)
	if !reflect.DeepEqual(got.Marks.Rows, next.Marks.Rows) || !reflect.DeepEqual(got.Marks.Colors, next.Marks.Colors) {
		t.Fatalf("reconstruction mismatch:\n got=%+v\nwant=%+v", got.Marks.Rows, next.Marks.Rows)
	}
	if Bytes(d) >= Bytes(next) {
		t.Fatalf("delta %d not smaller than full %d", Bytes(d), Bytes(next))
	}
}
