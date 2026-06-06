package tangram_test

import (
	"testing"

	"github.com/svend4/infon/pkg/tangram"
)

// The address book is the sixteen named figures: every entry tiles cleanly,
// resolves to its own name under the classifier, and has a distinct address.
func TestAddressBookComplete(t *testing.T) {
	cat := tangram.Catalog()
	if len(cat) != 16 {
		t.Fatalf("catalog has %d figures, want 16", len(cat))
	}
	addrs := map[string]string{}
	for name, f := range cat {
		if err := f.Check(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if m := tangram.Classify(f); m.Name != name || m.Score < 0.999 {
			t.Errorf("classify(%s) = %s (score %.3f), want self", name, m.Name, m.Score)
		}
		a := f.Address()
		if other, dup := addrs[a]; dup {
			t.Errorf("%s and %s share address %s", name, other, a)
		}
		addrs[a] = name
	}
}

// The eleven new animals use only the full block and the four corner triangles,
// so each cell is an exact I Ching trigram (digits 0..7).
func TestNewAnimalsTrigramClean(t *testing.T) {
	for _, n := range []string{"dog", "fox", "rabbit", "fish", "bird", "duck",
		"owl", "butterfly", "crab", "turtle", "snake"} {
		f, ok := tangram.Named(n)
		if !ok {
			t.Fatalf("%s missing from catalog", n)
		}
		if !f.TrigramClean() {
			t.Errorf("%s should be trigram-clean", n)
		}
	}
}
