package tangram7_test

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/tangram7"
)

func TestSquareTiles(t *testing.T) {
	r := tangram7.Verify(tangram7.Square(), 240)
	if r.Pieces != 7 {
		t.Fatalf("pieces = %d, want 7", r.Pieces)
	}
	if math.Abs(r.Area-16) > 1e-6 {
		t.Fatalf("area = %.4f, want 16", r.Area)
	}
	if r.Overlap > 0.01 {
		t.Fatalf("overlap = %.4f, want ~0 (pieces must tile, not stack)", r.Overlap)
	}
}

func TestTransformIdentity(t *testing.T) {
	p := tangram7.Poly{{1, 0}, {0, 1}, {2, 3}}
	q := p.Transform(8, false, 0, 0) // 8*45 = 360 degrees
	for i := range p {
		if math.Abs(p[i].X-q[i].X) > 1e-9 || math.Abs(p[i].Y-q[i].Y) > 1e-9 {
			t.Fatalf("360-degree rotation changed the polygon")
		}
	}
}

func TestRenderNonNil(t *testing.T) {
	img := tangram7.Render(tangram7.Square(), 200, 200)
	if img == nil || img.Bounds().Dx() != 200 {
		t.Fatal("render produced no image")
	}
}
