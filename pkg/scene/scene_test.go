package scene

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestRenderClampsCanvasSize(t *testing.T) {
	// Defaults applied for non-positive dimensions.
	f := RenderScene(&Scene{Width: 0, Height: -5})
	if f.Width != 64 || f.Height != 28 {
		t.Errorf("default size = %dx%d, want 64x28", f.Width, f.Height)
	}
	// Hard upper clamp so a misbehaving model cannot allocate a huge frame.
	big := RenderScene(&Scene{Width: 100000, Height: 100000})
	if big.Width != 400 || big.Height != 200 {
		t.Errorf("clamped size = %dx%d, want 400x200", big.Width, big.Height)
	}
}

func TestFillOp(t *testing.T) {
	red := [3]int{255, 0, 0}
	f := RenderScene(&Scene{Width: 4, Height: 3, Ops: []Op{
		{Op: "fill", Glyph: "#", Fg: &red, Bg: &red},
	}})
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			b := f.Blocks[y][x]
			if b.Glyph != '#' {
				t.Fatalf("cell (%d,%d) glyph = %q, want #", x, y, b.Glyph)
			}
			if b.Fg != (color.RGB{R: 255}) {
				t.Fatalf("cell (%d,%d) fg = %+v, want red", x, y, b.Fg)
			}
		}
	}
}

func TestRectOpBounds(t *testing.T) {
	green := [3]int{0, 255, 0}
	f := RenderScene(&Scene{Width: 6, Height: 6, Ops: []Op{
		{Op: "rect", X: 1, Y: 1, W: 2, H: 2, Glyph: "X", Fg: &green, Bg: &green},
	}})
	// Inside the rect.
	if f.Blocks[1][1].Glyph != 'X' || f.Blocks[2][2].Glyph != 'X' {
		t.Error("rect interior not drawn")
	}
	// Outside the rect must be untouched (space).
	if f.Blocks[0][0].Glyph != ' ' || f.Blocks[3][3].Glyph != ' ' {
		t.Error("rect drew outside its bounds")
	}
}

func TestVGradientEndpoints(t *testing.T) {
	from := [3]int{0, 0, 0}
	to := [3]int{255, 255, 255}
	f := RenderScene(&Scene{Width: 2, Height: 4, Ops: []Op{
		{Op: "vgradient", X: 0, Y: 0, W: 2, H: 4, Glyph: " ", From: &from, To: &to},
	}})
	top := f.Blocks[0][0].Bg
	bot := f.Blocks[3][0].Bg
	if top != (color.RGB{R: 0, G: 0, B: 0}) {
		t.Errorf("gradient top = %+v, want black", top)
	}
	if bot != (color.RGB{R: 255, G: 255, B: 255}) {
		t.Errorf("gradient bottom = %+v, want white", bot)
	}
	// Middle rows must interpolate (strictly between the endpoints).
	mid := f.Blocks[1][0].Bg
	if mid.R == 0 || mid.R == 255 {
		t.Errorf("gradient middle did not interpolate: %+v", mid)
	}
}

func TestTextOp(t *testing.T) {
	white := [3]int{255, 255, 255}
	f := RenderScene(&Scene{Width: 10, Height: 2, Ops: []Op{
		{Op: "text", X: 1, Y: 0, S: "Hi", Fg: &white},
	}})
	if f.Blocks[0][1].Glyph != 'H' || f.Blocks[0][2].Glyph != 'i' {
		t.Errorf("text not rendered: got %q%q", f.Blocks[0][1].Glyph, f.Blocks[0][2].Glyph)
	}
}

func TestDiscIsRoughlyCircular(t *testing.T) {
	white := [3]int{255, 255, 255}
	f := RenderScene(&Scene{Width: 21, Height: 21, Ops: []Op{
		{Op: "disc", CX: 10, CY: 10, R: 4, Glyph: "O", Fg: &white},
	}})
	// Center is filled.
	if f.Blocks[10][10].Glyph != 'O' {
		t.Error("disc center not filled")
	}
	// A far corner is not.
	if f.Blocks[0][0].Glyph == 'O' {
		t.Error("disc spilled into the corner")
	}
}

func TestRenderParsesJSON(t *testing.T) {
	data := []byte(`{"width":5,"height":2,"ops":[{"op":"text","x":0,"y":0,"s":"ok","fg":[255,255,255]}]}`)
	f, err := Render(data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if f.Blocks[0][0].Glyph != 'o' || f.Blocks[0][1].Glyph != 'k' {
		t.Error("parsed scene did not render text")
	}
}

func TestRenderRejectsBadJSON(t *testing.T) {
	if _, err := Render([]byte("{not json")); err == nil {
		t.Error("expected an error for malformed JSON")
	}
}

func TestOpsOutOfBoundsDoNotPanic(t *testing.T) {
	white := [3]int{255, 255, 255}
	// Negative and overflowing coordinates must be clipped by SetBlock, not panic.
	RenderScene(&Scene{Width: 4, Height: 4, Ops: []Op{
		{Op: "rect", X: -10, Y: -10, W: 100, H: 100, Glyph: "#", Fg: &white},
		{Op: "text", X: 100, Y: 100, S: "off", Fg: &white},
		{Op: "disc", CX: -5, CY: -5, R: 50, Glyph: "O", Fg: &white},
	}})
}
