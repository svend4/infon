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

func TestGatesTolerateModestNoise(t *testing.T) {
	sim := DemoScenario()
	// modest zero-mean noise (no drift): the baseline absorbs it and the severity
	// distribution is preserved, so every gate — including the KS distribution test —
	// should still pass. A systematic bias is drift and is meant to be caught (see
	// TestKSGateDiscriminates and the bias case in cmd/raygates).
	rep := Compare(sim, ApplyGap(sim, GapModel{Noise: 0.04, Seed: 1}))
	if !rep.Pass {
		t.Errorf("a modest zero-mean gap should still pass: %+v", rep.Gates)
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

func gateByName(rep GateReport, name string) *Gate {
	for i := range rep.Gates {
		if rep.Gates[i].Name == name {
			return &rep.Gates[i]
		}
	}
	return nil
}

func TestKS2Statistic(t *testing.T) {
	if d := ks2([]float64{1, 2, 3}, []float64{1, 2, 3}); d != 0 {
		t.Errorf("identical samples should give D=0, got %.3f", d)
	}
	if d := ks2([]float64{0, 0, 0}, []float64{1, 1, 1}); d != 1 {
		t.Errorf("disjoint samples should give D=1, got %.3f", d)
	}
	if d := ks2([]float64{0, 0, 1, 1}, []float64{0, 1, 1, 1}); d <= 0 || d >= 1 {
		t.Errorf("partly-overlapping samples should give 0<D<1, got %.3f", d)
	}
}

func TestKSGateDiscriminates(t *testing.T) {
	sim := DemoScenario()
	zero := gateByName(Compare(sim, ApplyGap(sim, GapModel{})), "severity-KS")
	if zero == nil {
		t.Fatal("Compare should include a severity-KS gate")
	}
	if zero.Metric != 0 || !zero.Pass {
		t.Errorf("zero gap should give KS D=0 and pass, got D=%.3f pass=%v", zero.Metric, zero.Pass)
	}
	big := gateByName(Compare(sim, ApplyGap(sim, GapModel{Noise: 0.6, Seed: 1})), "severity-KS")
	if big.Metric <= zero.Metric {
		t.Errorf("a large gap should widen the KS distance (%.3f) beyond zero-gap (%.3f)", big.Metric, zero.Metric)
	}
}
