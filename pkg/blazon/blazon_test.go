package blazon

import "testing"

func TestParseFieldAndCharge(t *testing.T) {
	sp := Parse("Azure, a sun Or")
	if sp.Format != "mixed" || sp.Mixed == nil || sp.Mixed.Grid == nil {
		t.Fatalf("bad spec: %+v", sp)
	}
	if got := sp.Mixed.Grid.Rows[0][0]; got != "blue" {
		t.Fatalf("field = %q, want blue", got)
	}
	found := false
	for _, s := range sp.Mixed.Sigils {
		if s.Name == "sun" && s.Color == "gold" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing gold sun charge")
	}
	if _, err := sp.Frame(); err != nil {
		t.Fatal(err)
	}
}

func TestParseNumberAndOrdinary(t *testing.T) {
	sp := Parse("Gules, on a chief Argent three mullets Sable")
	if sp.Mixed.Grid.Rows[0][0] != "white" { // chief argent paints the top band
		t.Fatalf("chief not painted: %q", sp.Mixed.Grid.Rows[0][0])
	}
	n := 0
	for _, s := range sp.Mixed.Sigils {
		if s.Name == "star" {
			n++
			if s.Color != "black" {
				t.Fatalf("mullet color %q", s.Color)
			}
		}
	}
	if n != 3 {
		t.Fatalf("got %d mullets, want 3", n)
	}
}

func TestParseCrossOrdinary(t *testing.T) {
	sp := Parse("Argent, a cross Gules")
	// center cell should be the cross tincture (red)
	if c := sp.Mixed.Grid.Rows[8][8]; c != "red" {
		t.Fatalf("cross center = %q, want red", c)
	}
}
