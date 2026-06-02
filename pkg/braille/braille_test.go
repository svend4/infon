package braille

import (
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
)

func TestCellFullAndBlank(t *testing.T) {
	if got := Cell(""); got != 0x2800 {
		t.Fatalf("blank cell = %U, want U+2800", got)
	}
	if got := Cell("full"); got != 0x28FF {
		t.Fatalf("full cell = %U, want U+28FF (all 8 dots)", got)
	}
}

func TestFromMarksShape(t *testing.T) {
	s := pseudo.Spec{Format: pseudo.FormatMarks, Marks: &pseudo.MarkArt{
		Rows: [][]string{{"full", ""}, {"", "full"}},
	}}
	rows := FromMarks(s)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r0 := []rune(rows[0])
	if len(r0) != 2 || r0[0] != 0x28FF || r0[1] != 0x2800 {
		t.Fatalf("row 0 = %q (%U %U)", rows[0], r0[0], r0[1])
	}
}

func TestCellPartialIsBetweenBlankAndFull(t *testing.T) {
	// a triangle covers some but not all sub-cells.
	got := Cell("tri-ul")
	if got == 0x2800 || got == 0x28FF {
		t.Fatalf("partial glyph mapped to %U, expected a mix of dots", got)
	}
}
