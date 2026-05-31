package tangram_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/semframe"
	"github.com/svend4/infon/pkg/tangram"
)

// Every catalog figure must obey the tangram rule: pieces meet edge to edge,
// never stacking. They must also actually contain pieces and tile some cells.
func TestCatalogWellFormed(t *testing.T) {
	for name, f := range tangram.Catalog() {
		rep := f.Validate()
		if !rep.WellFormed {
			t.Errorf("%s: not well-formed: overlaps=%v", name, rep.Overlaps)
		}
		if rep.FilledCells == 0 {
			t.Errorf("%s: no pieces", name)
		}
		if rep.CompleteCells == 0 {
			t.Errorf("%s: no fully-tiled cells", name)
		}
		if rep.Coverage <= 0 || rep.Coverage > 1 {
			t.Errorf("%s: coverage out of range: %v", name, rep.Coverage)
		}
	}
}

// A genuine area collision (two solids in one cell) must be flagged.
func TestOverlapDetected(t *testing.T) {
	f := tangram.Figure{Name: "clash", H: 1, W: 1, Pieces: []tangram.Piece{
		{Glyph: "full", Y: 0, X: 0, Color: "red"},
		{Glyph: "full", Y: 0, X: 0, Color: "blue"},
	}}
	if f.Validate().WellFormed {
		t.Fatal("two solids in one cell should not be well-formed")
	}
	if err := f.Check(); err == nil {
		t.Fatal("Check should return an error for an overlap")
	}
}

// Complementary pieces share only their hypotenuse: that is NOT an overlap, and
// together they tile the whole cell.
func TestComplementaryPairWellFormed(t *testing.T) {
	f := tangram.Figure{Name: "pair", H: 1, W: 1, Pieces: []tangram.Piece{
		{Glyph: "tri-ul", Y: 0, X: 0, Color: "amber"},
		{Glyph: "tri-lr", Y: 0, X: 0, Color: "gold"},
	}}
	rep := f.Validate()
	if !rep.WellFormed {
		t.Fatalf("complementary pair flagged as overlap: %v", rep.Overlaps)
	}
	if rep.CompleteCells != 1 {
		t.Fatalf("complementary pair should tile the cell completely, got %d", rep.CompleteCells)
	}
}

// The marks bridge must yield a renderable frame and a raster.
func TestSpecAndImage(t *testing.T) {
	f := tangram.Cat()
	fr, err := f.Marks("").Frame()
	if err != nil {
		t.Fatalf("marks frame: %v", err)
	}
	if fr == nil || fr.Width != f.W || fr.Height != f.H {
		t.Fatalf("frame size = %v×%v, want %d×%d", fr.Width, fr.Height, f.W, f.H)
	}
	img := f.Image(180, 160)
	if img == nil || img.Bounds().Dx() != 180 || img.Bounds().Dy() != 160 {
		t.Fatalf("image bounds = %v", img.Bounds())
	}
}

// Same seed -> same creature; several seeds all produce well-formed silhouettes.
func TestCreatureDeterministic(t *testing.T) {
	a, b := tangram.Creature(7), tangram.Creature(7)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("Creature(7) is not deterministic")
	}
	if reflect.DeepEqual(tangram.Creature(7), tangram.Creature(8)) {
		t.Fatal("different seeds produced the same creature")
	}
	for seed := int64(0); seed < 24; seed++ {
		if err := tangram.Creature(seed).Check(); err != nil {
			t.Errorf("creature seed %d not well-formed: %v", seed, err)
		}
	}
}

// Morphing, part 1 (correctness): the delta from one figure to another must
// reconstruct the target EXACTLY, however large the change.
func TestMorphReconstructs(t *testing.T) {
	cat := tangram.Cat().Marks("")
	swan := tangram.Swan().Marks("")
	delta, ok := semframe.DiffMarks(cat, swan)
	if !ok {
		t.Fatal("cat and swan should diff (same grid shape)")
	}
	if len(delta.Cells) == 0 {
		t.Fatal("morph produced no changes")
	}
	got := semframe.ApplyMarks(cat, delta)
	if !reflect.DeepEqual(got.Marks.Rows, swan.Marks.Rows) {
		t.Fatal("reconstructed glyphs != swan")
	}
	if !reflect.DeepEqual(got.Marks.Colors, swan.Marks.Colors) {
		t.Fatal("reconstructed colors != swan")
	}
}

// Morphing, part 2 (compression): a SMALL change — the real use case, a figure
// shifting or a limb moving — must be a P-frame far smaller than a keyframe.
func TestPFrameCompresses(t *testing.T) {
	base := tangram.Swan().Marks("")
	moved := tangram.Swan().Marks("")
	// Nudge: beak tip flips and one water cell brightens (a few cells change).
	moved.Marks.Rows[0][5] = "◥" // beak tip flips ◤ -> ◥
	moved.Marks.Colors[6][0] = "skyblue"
	moved.Marks.Colors[6][7] = "skyblue"

	delta, ok := semframe.DiffMarks(base, moved)
	if !ok {
		t.Fatal("same-shape specs should diff")
	}
	if n := len(delta.Cells); n == 0 || n > 6 {
		t.Fatalf("expected a few changed cells, got %d", n)
	}
	got := semframe.ApplyMarks(base, delta)
	if !reflect.DeepEqual(got.Marks.Rows, moved.Marks.Rows) || !reflect.DeepEqual(got.Marks.Colors, moved.Marks.Colors) {
		t.Fatal("small-change reconstruction failed")
	}
	dJSON, _ := json.Marshal(delta)
	kJSON, _ := json.Marshal(moved)
	if len(dJSON) >= len(kJSON) {
		t.Fatalf("P-frame (%d B) not smaller than keyframe (%d B)", len(dJSON), len(kJSON))
	}
	t.Logf("small move: %d changed cells, P-frame %d B vs keyframe %d B", len(delta.Cells), len(dJSON), len(kJSON))
}
