package pseudo

import "testing"

func TestMarksFormat(t *testing.T) {
	doc := []byte(`{"format":"marks","cols":12,"rows":6,"marks":{"bg":"navy","fg":"white",
		"rows":[["tri-ll","tri-lr"],["◣","◢"]],
		"colors":[["green","green"],["slate","slate"]]}}`)
	f, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	// both the named token and the literal glyph must resolve to ◣ / ◢
	var ll, lr bool
	for _, row := range f.Blocks {
		for _, b := range row {
			if b.Glyph == '◣' {
				ll = true
			}
			if b.Glyph == '◢' {
				lr = true
			}
		}
	}
	if !ll || !lr {
		t.Fatalf("marks not placed: ◣=%v ◢=%v", ll, lr)
	}
}
