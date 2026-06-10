package fleet

import "testing"

func TestGatesZeroGapIsPerfect(t *testing.T) {
	sim := DemoScenario()
	real := ApplyGap(sim, GapModel{}) // scale 1, no bias, no noise => identical
	rep := Compare(sim, real)
	if !rep.Pass {
		t.Fatalf("an identical real run must pass every gate: %+v", rep.Gates)
	}
	for _, g := range rep.Gates {
		switch g.Name {
		case "level-agreement", "cause-agreement", "bridge-agreement":
			if g.Metric != 1 {
				t.Errorf("%s should be 1.0 with no gap, got %.3f", g.Name, g.Metric)
			}
		case "severity-MAE":
			if g.Metric != 0 {
				t.Errorf("severity MAE should be 0 with no gap, got %.3f", g.Metric)
			}
		}
	}
}

func TestGatesTolerateModestGap(t *testing.T) {
	sim := DemoScenario()
	// a small steady bias: the per-unit baseline absorbs most of it, so the
	// engine's conclusions should still hold.
	rep := Compare(sim, ApplyGap(sim, GapModel{Bias: 0.06, Noise: 0.03, Seed: 1}))
	if !rep.Pass {
		t.Errorf("a modest sim-to-real gap should still pass: %+v", rep.Gates)
	}
}

func TestGatesFailOnLargeGap(t *testing.T) {
	sim := DemoScenario()
	rep := Compare(sim, ApplyGap(sim, GapModel{Noise: 0.5, Seed: 1}))
	if rep.Pass {
		t.Errorf("a large gap should break at least one gate, but all passed: %+v", rep.Gates)
	}
}

func TestApplyGapDeterministicAndClamped(t *testing.T) {
	sim := DemoScenario()
	g := GapModel{Bias: 0.1, Noise: 0.4, Seed: 7}
	a := ApplyGap(sim, g)
	b := ApplyGap(sim, g)
	for i := range a {
		for j := range a[i].Signals {
			if a[i].Signals[j].Value != b[i].Signals[j].Value {
				t.Fatal("ApplyGap should be deterministic for a fixed seed")
			}
			v := a[i].Signals[j].Value
			if v < 0 || v > 1 {
				t.Fatalf("perturbed value out of range: %.3f", v)
			}
		}
	}
}

func TestGateDirection(t *testing.T) {
	hi := gate("agree", 0.8, 0.75, true)  // higher-is-better, above threshold
	lo := gate("err", 0.2, 0.15, false)   // lower-is-better, above threshold
	good := gate("err", 0.1, 0.15, false) // lower-is-better, below threshold
	if !hi.Pass {
		t.Error("0.8 >= 0.75 should pass a higher-is-better gate")
	}
	if lo.Pass {
		t.Error("0.2 > 0.15 should fail a lower-is-better gate")
	}
	if !good.Pass {
		t.Error("0.1 <= 0.15 should pass a lower-is-better gate")
	}
}
