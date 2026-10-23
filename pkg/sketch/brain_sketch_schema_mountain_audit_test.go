package sketch

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/scene"
)

// A hill/mountain's H comes verbatim from a brain's JSON. ToScene used to emit
// one rect op per row of it, so {"kind":"mountain","h":1e9} allocated a billion
// ops although every row above the top of the canvas is clipped away anyway.
func TestAuditMountainOpsBoundedByCanvas(t *testing.T) {
	const w, h = 64, 24
	for _, kind := range []string{"mountain", "hill"} {
		sc := ToScene(Sketch{Shapes: []Shape{{Kind: kind, X: w / 2, H: 1 << 20, Color: "green"}}}, w, h)
		if got := opsByKind(sc.Ops, "rect"); got > h {
			t.Errorf("%s with h=1<<20 on a %dx%d canvas emitted %d rect ops, want <= %d", kind, w, h, got, h)
		}
		f := scene.RenderScene(&sc)
		// The base row sits just above the horizon and spans the whole canvas.
		horizon := h * 2 / 3
		green := palette["green"]
		for x := 0; x < w; x++ {
			b := f.Blocks[horizon-1][x]
			if b.Fg.R != uint8(green[0]) || b.Fg.G != uint8(green[1]) || b.Fg.B != uint8(green[2]) {
				t.Fatalf("%s base row cell %d not drawn: %+v", kind, x, b)
			}
		}
	}
}

// Rows that fall above the canvas are invisible, so dropping them must not change
// the rendered picture: a mountain taller than the horizon renders exactly as
// the full row-by-row construction did.
func TestAuditTallMountainRendersUnchanged(t *testing.T) {
	const w, h = 64, 24
	sh := Shape{Kind: "mountain", X: 20, H: 30, Color: "darkgreen"}
	got := ToScene(Sketch{Sky: "navy", Shapes: []Shape{sh}}, w, h)

	want := ToScene(Sketch{Sky: "navy"}, w, h)
	c := colv(sh.Color, [3]int{})
	horizon := h * 2 / 3
	for i := 0; i < sh.H; i++ {
		half := sh.H - 1 - i
		want.Ops = append(want.Ops, scene.Op{Op: "rect", X: sh.X - half, Y: horizon - 1 - i, W: 2*half + 1, H: 1, Glyph: "█", Fg: p(c), Bg: p(c)})
	}
	if !reflect.DeepEqual(scene.RenderScene(&got).Blocks, scene.RenderScene(&want).Blocks) {
		t.Fatal("tall mountain renders differently from the full row-by-row construction")
	}
}
