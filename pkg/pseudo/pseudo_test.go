package pseudo

import (
	"encoding/json"
	"image"
	"testing"
)

func sameImage(a, b image.Image) bool {
	ra, ok1 := a.(*image.RGBA)
	rb, ok2 := b.(*image.RGBA)
	if !ok1 || !ok2 || ra.Bounds() != rb.Bounds() || len(ra.Pix) != len(rb.Pix) {
		return false
	}
	for i := range ra.Pix {
		if ra.Pix[i] != rb.Pix[i] {
			return false
		}
	}
	return true
}

func TestGridFrameAndImage(t *testing.T) {
	doc := []byte(`{"format":"grid","cols":40,"rows":20,"grid":{"rows":[["navy","navy","blue"],["blue","teal","green"],["green","gold","orange"]]}}`)
	f, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	if f.Width != 40 || f.Height != 20 {
		t.Fatalf("frame dims = %dx%d, want 40x20", f.Width, f.Height)
	}

	var s Spec
	if err := json.Unmarshal(doc, &s); err != nil {
		t.Fatal(err)
	}
	img, err := s.Image(80, 40)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 80 || b.Dy() != 40 {
		t.Fatalf("image bounds = %v, want 80x40", b)
	}
	img2, _ := s.Image(80, 40)
	if !sameImage(img, img2) {
		t.Fatal("pseudo-diffusion image is not deterministic")
	}
}

func TestPixelsMosaic(t *testing.T) {
	doc := []byte(`{"format":"pixels","cols":24,"rows":12,"pixels":{"rows":[["red","green"],["blue","gold"]]}}`)
	var s Spec
	if err := json.Unmarshal(doc, &s); err != nil {
		t.Fatal(err)
	}
	f, err := s.Frame()
	if err != nil {
		t.Fatal(err)
	}
	// top-left cell must be exactly "red" (crisp, no blur)
	red := Color("red", defGridColor)
	if got := f.Blocks[0][0].Bg; got != red {
		t.Fatalf("top-left = %v, want red %v", got, red)
	}
}

func TestGlyphs(t *testing.T) {
	doc := []byte(`{"format":"glyphs","cols":20,"rows":8,"glyphs":{"bg":"navy","palette":{"^":"green","~":"blue","*":"gold"},"rows":["   *    ","  ^^^   ","~~~~~~~~"]}}`)
	f, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	green := Color("green", defGridColor)
	found := false
	for _, row := range f.Blocks {
		for _, b := range row {
			if b.Glyph == '^' {
				found = true
				if b.Fg != green {
					t.Fatalf("'^' fg = %v, want green %v", b.Fg, green)
				}
			}
		}
	}
	if !found {
		t.Fatal("glyph '^' was not placed")
	}
}

func TestSigils(t *testing.T) {
	doc := []byte(`{"format":"sigils","cols":30,"rows":12,"sigils":{"sky":"navy","ground":"darkgreen","items":[{"name":"sun","x":0.8,"y":0.2,"color":"gold"},{"name":"mountain","x":0.4,"y":0.6,"color":"slate"}]}}`)
	f, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range f.Blocks {
		for _, b := range row {
			if b.Glyph == '☀' {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("sun sigil was not placed")
	}
}

func TestVectorAndSketch(t *testing.T) {
	v := []byte(`{"format":"vector","cols":24,"rows":10,"vector":{"width":24,"height":10,"bg":[10,10,20],"ops":[{"op":"disc","cx":6,"cy":5,"r":3,"glyph":"█","fg":[250,200,90]}]}}`)
	if _, err := Render(v); err != nil {
		t.Fatalf("vector: %v", err)
	}
	sk := []byte(`{"format":"sketch","cols":40,"rows":16,"sketch":{"sky":"navy","ground":"darkgreen","shapes":[{"kind":"sun","color":"gold"},{"kind":"mountain","color":"slate"}]}}`)
	if _, err := Render(sk); err != nil {
		t.Fatalf("sketch: %v", err)
	}
}

func TestColorTokens(t *testing.T) {
	if c := Color("#ff0000", defGridColor); c.R != 255 || c.G != 0 || c.B != 0 {
		t.Fatalf("#ff0000 -> %v", c)
	}
	if c := Color("#0f0", defGridColor); c.R != 0 || c.G != 255 || c.B != 0 {
		t.Fatalf("#0f0 -> %v", c)
	}
	if c := Color("10,20,30", defGridColor); c.R != 10 || c.G != 20 || c.B != 30 {
		t.Fatalf("triple -> %v", c)
	}
	if c := Color("navy", defGridColor); c == defGridColor {
		t.Fatal("named color 'navy' was not resolved")
	}
	if c := Color("definitely-not-a-color", defGridColor); c != defGridColor {
		t.Fatal("unknown token should fall back to default")
	}
}

func TestDiffuseFrames(t *testing.T) {
	g := &Grid{Rows: [][]string{{"navy", "blue"}, {"teal", "gold"}}}
	frames := g.DiffuseFrames(32, 16, 6)
	if len(frames) != 6 {
		t.Fatalf("frames = %d, want 6", len(frames))
	}
	for i, fr := range frames {
		if b := fr.Bounds(); b.Dx() != 32 || b.Dy() != 16 {
			t.Fatalf("frame %d bounds = %v", i, b)
		}
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := Render([]byte(`{"format":"bogus"}`)); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
