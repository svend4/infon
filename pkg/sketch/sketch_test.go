package sketch

import (
	"testing"

	"github.com/svend4/infon/pkg/scene"
)

func opsByKind(ops []scene.Op, kind string) int {
	n := 0
	for _, o := range ops {
		if o.Op == kind {
			n++
		}
	}
	return n
}

func TestToSceneDefaultsCanvas(t *testing.T) {
	sc := ToScene(Sketch{Sky: "navy"}, 0, 0)
	if sc.Width != 64 || sc.Height != 24 {
		t.Errorf("default canvas = %dx%d, want 64x24", sc.Width, sc.Height)
	}
}

func TestToSceneSkyAndGround(t *testing.T) {
	// Sky always emits a vgradient; ground (when set) adds a rect.
	sc := ToScene(Sketch{Sky: "orange", Ground: "navy"}, 40, 20)
	if opsByKind(sc.Ops, "vgradient") < 1 {
		t.Error("expected a sky vgradient op")
	}
	if opsByKind(sc.Ops, "rect") < 1 {
		t.Error("expected a ground rect op")
	}

	// No ground => no leading ground rect (mountains may still add rects, so use
	// a shape-free sketch to isolate this).
	noGround := ToScene(Sketch{Sky: "orange"}, 40, 20)
	if opsByKind(noGround.Ops, "rect") != 0 {
		t.Errorf("sky-only sketch should emit no rect, got %d", opsByKind(noGround.Ops, "rect"))
	}
}

func TestToSceneShapesMap(t *testing.T) {
	sk := Sketch{
		Sky: "navy",
		Shapes: []Shape{
			{Kind: "sun", X: 10, Y: 4, W: 3, Color: "gold"},
			{Kind: "star", X: 5, Y: 2, Color: "white"},
			{Kind: "text", X: 1, Y: 1, S: "hello", Color: "white"},
			{Kind: "mountain", X: 20, H: 5, Color: "darkgreen"},
		},
	}
	sc := ToScene(sk, 64, 24)
	if opsByKind(sc.Ops, "disc") != 1 {
		t.Errorf("sun should map to exactly one disc, got %d", opsByKind(sc.Ops, "disc"))
	}
	// star + text => two text ops.
	if opsByKind(sc.Ops, "text") != 2 {
		t.Errorf("star+text should map to two text ops, got %d", opsByKind(sc.Ops, "text"))
	}
	// A mountain of height 5 emits one rect per row.
	if got := opsByKind(sc.Ops, "rect"); got < 5 {
		t.Errorf("mountain h=5 should emit >=5 rects, got %d", got)
	}
}

func TestUnknownColorAndShapeAreSafe(t *testing.T) {
	// An unknown color name falls back to a default; an unknown shape kind is
	// silently ignored. Neither should panic, and the scene must still render.
	sk := Sketch{
		Sky: "not-a-color",
		Shapes: []Shape{
			{Kind: "unicorn", X: 1, Y: 1, Color: "chartreuse"},
			{Kind: "text", X: 0, Y: 0, S: "ok", Color: "white"},
		},
	}
	sc := ToScene(sk, 32, 16)
	f := scene.RenderScene(&sc)
	if f.Width != 32 || f.Height != 16 {
		t.Fatalf("rendered size = %dx%d, want 32x16", f.Width, f.Height)
	}
	// The text shape should still have produced a glyph somewhere on row 0.
	if f.Blocks[0][0].Glyph != 'o' {
		t.Errorf("expected 'o' at (0,0), got %q", f.Blocks[0][0].Glyph)
	}
}

func TestSketchRoundTripsThroughScene(t *testing.T) {
	// End-to-end: a sketch -> scene -> frame must not panic and must fill the sky.
	sk := Sketch{Sky: "purple", Ground: "navy", Shapes: []Shape{
		{Kind: "sun", Color: "gold"},
		{Kind: "cloud", X: 4, Y: 3, W: 6, Color: "white"},
		{Kind: "building", X: 10, W: 4, H: 6, Color: "gray"},
		{Kind: "band", Y: 8, H: 2, Color: "teal"},
	}}
	sc := ToScene(sk, 48, 18)
	f := scene.RenderScene(&sc)
	// The very top row is sky: with a vgradient fill it must not be the default
	// black background.
	top := f.Blocks[0][0].Bg
	if top.R == 0 && top.G == 0 && top.B == 0 {
		t.Error("sky gradient did not paint the top row")
	}
}
