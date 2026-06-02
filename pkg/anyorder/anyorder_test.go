package anyorder

import "testing"

func TestAnyOrderNoWorseThanL2R(t *testing.T) {
	hyps, W, H := Hypotheses()
	n := W * H
	if len(hyps) < 4 {
		t.Fatalf("only %d same-size hypotheses; need a real catalog set", len(hyps))
	}
	sumA, sumL, strict := 0, 0, 0
	for _, tg := range hyps {
		a := Reveals(tg, hyps, n, "anyorder")
		l := Reveals(tg, hyps, n, "l2r")
		sumA += a
		sumL += l
		if a < l {
			strict++
		}
	}
	if sumA > sumL {
		t.Fatalf("any-order total anchors %d worse than left-to-right %d", sumA, sumL)
	}
	if strict == 0 {
		t.Fatalf("any-order never beat left-to-right (sumA=%d sumL=%d)", sumA, sumL)
	}
}

func TestIdentifyRecoversTarget(t *testing.T) {
	hyps, W, H := Hypotheses()
	n := W * H
	for _, tg := range hyps {
		got, anchors := Identify(tg, hyps, n)
		if got.Name != tg.Name {
			t.Fatalf("identify %s -> %q", tg.Name, got.Name)
		}
		if len(anchors) > n {
			t.Fatalf("%s used %d anchors on an %d-cell grid", tg.Name, len(anchors), n)
		}
	}
}
