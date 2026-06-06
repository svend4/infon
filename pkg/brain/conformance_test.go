package brain

import "testing"

// The reference brain must pass its own conformance battery — any third-party
// brain is conformant iff it passes the same suite (see cmd/conform).
func TestConformanceReference(t *testing.T) {
	results, ok := RunConformance(func(r Request) (Response, error) { return Reference(r), nil })
	for _, r := range results {
		if !r.Pass {
			t.Errorf("case %s: %s", r.Name, r.Detail)
		}
	}
	if !ok {
		t.Fatal("reference brain is not conformant")
	}
}
