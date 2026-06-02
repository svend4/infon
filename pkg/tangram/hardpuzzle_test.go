package tangram_test

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/tangram"
)

// From the silhouette alone (no per-cell glyphs) the hard solver recovers the
// tiling: high IoU and well-formed; the coarse reader covers the target but
// never beats the silhouette reader.
func TestHardSolveRecovers(t *testing.T) {
	for name, f := range tangram.Catalog() {
		ps := tangram.PuzzleFromFigure(f)
		hp := tangram.HardFromFigure(f)
		hard := tangram.ScorePuzzle(ps, tangram.SolveHard(hp))
		if hard.IoU < 0.95 {
			t.Errorf("%s: hard IoU=%.3f, want >=0.95", name, hard.IoU)
		}
		if !hard.WellFormed {
			t.Errorf("%s: hard solution not well-formed", name)
		}
		coarse := tangram.ScorePuzzle(ps, tangram.SolveCoarse(hp))
		if coarse.Recall < 0.999 {
			t.Errorf("%s: coarse recall=%.3f, want full cover", name, coarse.Recall)
		}
		if coarse.IoU > hard.IoU+1e-9 {
			t.Errorf("%s: coarse IoU %.3f beat hard %.3f", name, coarse.IoU, hard.IoU)
		}
	}
}

// On a diagonal-heavy figure the coarse cell-occupancy reader loses to the
// silhouette reader - the diagonals carry the tangram information.
func TestHardDiagonalsMatter(t *testing.T) {
	f := tangram.Swan()
	ps := tangram.PuzzleFromFigure(f)
	hp := tangram.HardFromFigure(f)
	hard := tangram.ScorePuzzle(ps, tangram.SolveHard(hp))
	coarse := tangram.ScorePuzzle(ps, tangram.SolveCoarse(hp))
	if coarse.IoU >= hard.IoU-0.02 {
		t.Errorf("swan: coarse IoU %.3f should trail hard %.3f", coarse.IoU, hard.IoU)
	}
	if coarse.Overflow <= 0 {
		t.Errorf("swan: coarse should overflow diagonals, got %.3f", coarse.Overflow)
	}
}

func TestSilhouetteString(t *testing.T) {
	s := tangram.Cat().SilhouetteString(2)
	if !strings.Contains(s, "#") {
		t.Error("silhouette should contain filled pixels")
	}
	if len(strings.Split(strings.TrimRight(s, "\n"), "\n")) < 8 {
		t.Error("silhouette should have multiple rows")
	}
}
