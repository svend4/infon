package scene

import "testing"

func isControlGlyph(r rune) bool {
	return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r > 0x10ffff || (r >= 0xd800 && r <= 0xdfff)
}

// A model response must not be able to put terminal control characters into
// the frame through "text" strings or op glyphs.
func TestDSLControlRunesDoNotReachFrame(t *testing.T) {
	data := []byte(`{"width":8,"height":3,"ops":[
		{"op":"fill","glyph":"\u001b"},
		{"op":"text","x":0,"y":0,"s":"a\u001b[2J\u009bb"},
		{"op":"rect","x":0,"y":1,"w":2,"h":1,"glyph":"\u0007"},
		{"op":"hgradient","x":2,"y":1,"w":2,"h":1,"glyph":"\r","from":[1,1,1],"to":[2,2,2]},
		{"op":"vgradient","x":4,"y":1,"w":1,"h":1,"glyph":"\u0090","from":[1,1,1],"to":[2,2,2]},
		{"op":"disc","cx":6,"cy":2,"r":0,"glyph":"\u007f"}
	]}`)
	f, err := Render(data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			if g := f.Blocks[y][x].Glyph; isControlGlyph(g) {
				t.Errorf("cell (%d,%d) holds control glyph %U", x, y, g)
			}
		}
	}
	// Text keeps its layout: each control rune occupies one blank cell.
	var row []rune
	for x := 0; x < 7; x++ {
		row = append(row, f.Blocks[0][x].Glyph)
	}
	if got, want := string(row), "a [2J b"; got != want {
		t.Errorf("text row = %q, want %q", got, want)
	}
}
