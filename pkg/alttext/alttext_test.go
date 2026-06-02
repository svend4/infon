package alttext

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
)

func TestDescribeGrid(t *testing.T) {
	rows := make([][]string, 4)
	for y := range rows {
		rows[y] = []string{"navy", "navy", "navy", "teal"}
	}
	s := pseudo.Spec{Format: pseudo.FormatGrid, Cols: 12, Rows: 8, Grid: &pseudo.Grid{Rows: rows}}
	d := Describe(s)
	if !strings.Contains(d, "color-grid") || !strings.Contains(d, "navy") {
		t.Fatalf("grid description = %q", d)
	}
}

func TestDescribeSigils(t *testing.T) {
	s := pseudo.Spec{Format: pseudo.FormatSigils, Title: "harbor", Sigils: &pseudo.SigilScene{
		Items: []pseudo.Sigil{{Name: "sun"}, {Name: "boat"}},
	}}
	d := Describe(s)
	if !strings.Contains(d, "sun") || !strings.Contains(d, "harbor") {
		t.Fatalf("sigil description = %q", d)
	}
}

func TestDescribeMarks(t *testing.T) {
	s := pseudo.Spec{Format: pseudo.FormatMarks, Marks: &pseudo.MarkArt{
		Rows: [][]string{{"full", ""}, {"", "tri-ul"}},
	}}
	d := Describe(s)
	if !strings.Contains(d, "marks") || !strings.Contains(d, "2 sub-cell") {
		t.Fatalf("marks description = %q", d)
	}
}
