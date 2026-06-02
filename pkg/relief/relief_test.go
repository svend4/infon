package relief_test

import (
	"testing"

	"github.com/svend4/infon/pkg/relief"
	"github.com/svend4/infon/pkg/tangram"
)

func TestFacesAndRender(t *testing.T) {
	f := tangram.Cat()
	h := relief.Uniform(f, 120)
	faces := relief.Faces(f, h)
	if len(faces) < 4*len(f.Pieces) {
		t.Errorf("faces=%d, want >= %d (top + >=3 walls per cell)", len(faces), 4*len(f.Pieces))
	}
	img := relief.Render(f, h, 200, 200)
	if img == nil || img.Bounds().Dx() != 200 {
		t.Fatal("render bad")
	}
}

func TestHeightMaps(t *testing.T) {
	f := tangram.Swan()
	u := relief.Uniform(f, 100)
	if len(u) != f.H || len(u[0]) != f.W {
		t.Fatalf("uniform dims %dx%d, want %dx%d", len(u), len(u[0]), f.H, f.W)
	}
	occ := 0
	for _, row := range u {
		for _, b := range row {
			if b == 100 {
				occ++
			}
		}
	}
	if occ == 0 {
		t.Error("uniform set no cells")
	}
	g := relief.Gradient(f, 50, 150)
	if len(g) != f.H {
		t.Errorf("gradient rows=%d, want %d", len(g), f.H)
	}
	if relief.SpecBytes(f) != f.H*f.W {
		t.Errorf("specbytes=%d, want %d", relief.SpecBytes(f), f.H*f.W)
	}
}
