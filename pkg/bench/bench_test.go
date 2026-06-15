package bench_test

import (
	"testing"

	"github.com/svend4/infon/pkg/bench"
)

// The bench is self-verifying: a green run means every probed capability works.
func TestRunAllGreen(t *testing.T) {
	r := bench.Run()
	if r.Total < 8 {
		t.Fatalf("expected at least 8 checks, got %d", r.Total)
	}
	if r.Passed != r.Total {
		for _, c := range r.Checks {
			if !c.Pass {
				t.Errorf("FAIL %s/%s: %s %s", c.Group, c.Name, c.Metric, c.Detail)
			}
		}
		t.Fatalf("%d/%d checks passed", r.Passed, r.Total)
	}
	for _, c := range r.Checks {
		if c.Name == "conformance" && c.Metric != "14/14" {
			t.Errorf("conformance metric = %q, want 14/14", c.Metric)
		}
	}
}
