package fold

import (
	"testing"

	t7 "github.com/svend4/infon/pkg/tangram7"
)

func maxZ(faces []Face) float64 {
	m := 0.0
	for _, f := range faces {
		for _, p := range f.Pts {
			if p.Z > m {
				m = p.Z
			}
		}
	}
	return m
}

// At t=0 the figure is flat (every vertex at z=0); at t=1 it has risen.
func TestFrameRises(t *testing.T) {
	sq := t7.Square()
	if z := maxZ(Frame(sq, 0)); z != 0 {
		t.Errorf("t=0 should be flat, max z=%.3f", z)
	}
	if z := maxZ(Frame(sq, 1)); z < 1.0 {
		t.Errorf("t=1 should rise, max z=%.3f", z)
	}
	// every piece contributes one top face plus one wall per edge
	faces := Frame(sq, 1)
	want := 0
	for _, p := range sq {
		want += 1 + len(p.Shape)
	}
	if len(faces) != want {
		t.Errorf("faces=%d, want %d", len(faces), want)
	}
}

// iso is z-up: raising z moves a point up the screen (smaller screen-y).
func TestIsoZUp(t *testing.T) {
	_, y0 := iso(Pt3{0, 0, 0})
	_, y1 := iso(Pt3{0, 0, 1})
	if !(y1 < y0) {
		t.Errorf("z up should decrease screen-y: y0=%.3f y1=%.3f", y0, y1)
	}
}

func TestSpecAndRender(t *testing.T) {
	sq := t7.Square()
	fs := SpecFor(sq)
	if len(fs.Heights) != len(sq) || len(fs.Names) != len(sq) {
		t.Fatalf("spec has %d heights for %d pieces", len(fs.Heights), len(sq))
	}
	if Render(Frame(sq, 1), 200, 200) == nil {
		t.Fatal("nil image")
	}
}
