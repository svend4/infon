package tangram_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/tangram"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// The reference solver must reproduce every catalog figure exactly: the palette
// contains each cell's own glyph, so best-match picks it and IoU is 1.
func TestSolveReproducesEveryFigure(t *testing.T) {
	for name, f := range tangram.Catalog() {
		p := tangram.PuzzleFromFigure(f)
		sc := tangram.ScorePuzzle(p, tangram.Solve(p))
		if !sc.WellFormed {
			t.Errorf("%s: solve not well-formed", name)
		}
		if !almost(sc.IoU, 1.0) {
			t.Errorf("%s: solve IoU=%.4f, want 1.0", name, sc.IoU)
		}
		if !almost(sc.Recall, 1.0) {
			t.Errorf("%s: solve recall=%.4f, want 1.0", name, sc.Recall)
		}
		if sc.Overflow > 1e-9 {
			t.Errorf("%s: solve overflow=%.4f, want 0", name, sc.Overflow)
		}
	}
}

// The naive solid-fill baseline covers the target (recall 1) but spills over the
// diagonal boundaries every catalog figure has, so IoU<1 and overflow>0. This is
// what makes game:tangram a discriminating benchmark.
func TestNaiveBaselineIsWeaker(t *testing.T) {
	for name, f := range tangram.Catalog() {
		p := tangram.PuzzleFromFigure(f)
		sc := tangram.ScorePuzzle(p, tangram.NaiveSolve(p))
		if !almost(sc.Recall, 1.0) {
			t.Errorf("%s: naive recall=%.4f, want 1 (covers target)", name, sc.Recall)
		}
		if sc.IoU >= 1.0 {
			t.Errorf("%s: naive IoU=%.4f, should be <1", name, sc.IoU)
		}
		if sc.Overflow <= 0 {
			t.Errorf("%s: naive overflow=%.4f, should be >0", name, sc.Overflow)
		}
		if !sc.WellFormed {
			t.Errorf("%s: naive should still be well-formed", name)
		}
	}
}

func TestEmptySolutionScoresZero(t *testing.T) {
	p := tangram.PuzzleFromFigure(tangram.Cat())
	sc := tangram.ScorePuzzle(p, tangram.Figure{H: p.H, W: p.W})
	if sc.IoU != 0 || sc.Recall != 0 {
		t.Fatalf("empty solution IoU=%.3f recall=%.3f, want 0/0", sc.IoU, sc.Recall)
	}
}

// End-to-end through the protocol: a tangram puzzle in Request.State, a solved
// figure in Response.Tangram, scored back to a high IoU.
func TestProtocolTangramRoundTrip(t *testing.T) {
	p := tangram.PuzzleFromFigure(tangram.Swan())
	state, _ := json.Marshal(p)
	resp := brain.Reference(brain.Request{Kind: "move", Game: "tangram", State: state})
	if resp.Error != "" {
		t.Fatalf("reference error: %s", resp.Error)
	}
	if len(resp.Tangram) == 0 {
		t.Fatal("no tangram solution in response")
	}
	var sol tangram.Figure
	if err := json.Unmarshal(resp.Tangram, &sol); err != nil {
		t.Fatal(err)
	}
	sc := tangram.ScorePuzzle(p, sol)
	if !sc.WellFormed || sc.IoU < 0.99 {
		t.Fatalf("protocol solve weak: well-formed=%v IoU=%.3f", sc.WellFormed, sc.IoU)
	}
}
