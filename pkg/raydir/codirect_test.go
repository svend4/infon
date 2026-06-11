package raydir

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

// stubDirector returns a rayscene with n forms of the given kinds.
type stubDirector struct {
	n     int
	kinds []string
}

func (s stubDirector) Decide(_ brain.Request) (brain.Response, error) {
	objs := []brain.ObjSpec{{Kind: "plane", Color: [3]float64{0.5, 0.5, 0.5}}}
	for i := 0; i < s.n; i++ {
		k := "sphere"
		if len(s.kinds) > 0 {
			k = s.kinds[i%len(s.kinds)]
		}
		objs = append(objs, brain.ObjSpec{Kind: k, X: float64(i) - 1, Y: 1, Z: 3, R: 0.6, Color: [3]float64{0.8, 0.3, 0.3}})
	}
	spec := brain.SceneSpec{Objects: objs, Light: [3]float64{6, 9, -4}, SkyTop: [3]float64{0.4, 0.5, 0.8}, SkyBot: [3]float64{0.8, 0.8, 0.9}}
	ray, _ := json.Marshal(spec)
	return brain.Response{Protocol: brain.Protocol, Kind: "move", Ray: ray}, nil
}

type errDirector struct{}

func (errDirector) Decide(_ brain.Request) (brain.Response, error) {
	return brain.Response{}, errors.New("transport down")
}

func TestCoDirectPicksRicherAndMerges(t *testing.T) {
	a := stubDirector{n: 2}
	b := stubDirector{n: 6, kinds: []string{"sphere", "box", "tree"}}
	winner, merged, rep := CoDirect(a, "x", b, "y")
	if rep.Winner != "B" {
		t.Fatalf("the richer director B should win, got %q (A=%d B=%d)", rep.Winner, rep.ScoreA, rep.ScoreB)
	}
	if rep.ScoreB <= rep.ScoreA {
		t.Errorf("B should score higher: A=%d B=%d", rep.ScoreA, rep.ScoreB)
	}
	if len(merged.Objects) <= len(winner.Objects) {
		t.Errorf("merge should add the other director's forms (winner %d, merged %d)", len(winner.Objects), len(merged.Objects))
	}
	// the merge keeps exactly one ground plane
	planes := 0
	for _, o := range merged.Objects {
		if o.Kind == "plane" {
			planes++
		}
	}
	if planes != 1 {
		t.Errorf("merged region should have exactly one ground plane, got %d", planes)
	}
}

func TestCoDirectInvalidLoses(t *testing.T) {
	winner, _, rep := CoDirect(errDirector{}, "x", stubDirector{n: 3}, "y")
	if rep.Winner != "B" {
		t.Fatalf("a director whose transport failed should lose, got %q", rep.Winner)
	}
	if len(winner.Objects) == 0 {
		t.Error("the surviving director's region should render")
	}
}
