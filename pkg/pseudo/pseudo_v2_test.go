package pseudo

import "testing"

func TestMixedFormat(t *testing.T) {
	doc := []byte(`{"format":"mixed","cols":40,"rows":20,"palette":"neon","mixed":{"grid":{"rows":[["navy","blue"],["teal","gold"]]},"sigils":[{"name":"sun","x":0.7,"y":0.2,"color":"gold"}],"labels":[{"text":"hi","x":0.1,"y":0.1,"color":"white"}]}}`)
	f, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	if f.Width != 40 || f.Height != 20 {
		t.Fatalf("dims %dx%d", f.Width, f.Height)
	}
	sun, label := false, false
	for _, row := range f.Blocks {
		for _, b := range row {
			if b.Glyph == '☀' {
				sun = true
			}
			if b.Glyph == 'h' {
				label = true
			}
		}
	}
	if !sun {
		t.Fatal("sun sigil not placed in mixed")
	}
	if !label {
		t.Fatal("label not placed in mixed")
	}
}

func TestPaletteChangesColor(t *testing.T) {
	base, _ := Render([]byte(`{"format":"grid","cols":10,"rows":6,"grid":{"rows":[["navy"]]}}`))
	neon, _ := Render([]byte(`{"format":"grid","cols":10,"rows":6,"palette":"neon","grid":{"rows":[["navy"]]}}`))
	if base.Blocks[0][0].Bg == neon.Blocks[0][0].Bg {
		t.Fatal("neon palette did not change navy")
	}
}

func TestUnknownPaletteFallsBack(t *testing.T) {
	f, err := Render([]byte(`{"format":"grid","cols":8,"rows":4,"palette":"nope","grid":{"rows":[["gold"]]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Blocks[0][0].Bg != Color("gold", defGridColor) {
		t.Fatal("unknown palette should fall back to base")
	}
}
