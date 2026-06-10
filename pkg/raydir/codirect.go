package raydir

import "github.com/svend4/infon/pkg/brain"

// codirect.go is the "debate × director" combination: two brains each author a
// region for a prompt, a referee validates both (renderable) and scores them by
// richness, and the result is the winner plus a MERGED region that unions both
// directors' forms — so two minds make a richer world than either alone, and the
// existing rayscene validator (BuildScene's sanitiser) is the referee that keeps a
// live model honest. Pure data; offline by giving both sides the reference brain.

// CoReport records the outcome of a co-direction.
type CoReport struct {
	ScoreA, ScoreB int
	ObjsA, ObjsB   int
	Winner         string // "A" | "B" | "tie" | "none"
	Merged         int
}

// richness scores a spec: forms + variety of kinds + emitters (a lit, varied
// region beats a bare one).
func richness(s brain.SceneSpec) int {
	kinds := map[string]bool{}
	emit := 0
	for _, o := range s.Objects {
		kinds[o.Kind] = true
		if o.Emit != [3]float64{} {
			emit++
		}
	}
	return len(s.Objects) + 2*len(kinds) + emit
}

// renderable reports whether a spec sanitises to something a renderer can draw.
func renderable(s brain.SceneSpec) bool { return len(BuildScene(s).Objects) > 0 }

// CoDirect asks brain a (promptA) and brain b (promptB) to each author a region,
// validates and scores both, and returns the richer valid one as the winner plus a
// merged region (the winner's region with the other's forms added to the side,
// capped). The CoReport explains the verdict.
func CoDirect(a brain.Brain, promptA string, b brain.Brain, promptB string) (winner, merged brain.SceneSpec, rep CoReport) {
	_, specA, errA := AuthorScene(a, promptA)
	_, specB, errB := AuthorScene(b, promptB)
	validA := errA == nil && renderable(specA)
	validB := errB == nil && renderable(specB)
	rep.ObjsA, rep.ObjsB = len(specA.Objects), len(specB.Objects)
	if validA {
		rep.ScoreA = richness(specA)
	}
	if validB {
		rep.ScoreB = richness(specB)
	}

	switch {
	case validA && validB:
		if rep.ScoreA == rep.ScoreB {
			winner, rep.Winner = specA, "tie"
		} else if rep.ScoreA > rep.ScoreB {
			winner, rep.Winner = specA, "A"
		} else {
			winner, rep.Winner = specB, "B"
		}
		other := specB
		if rep.Winner == "B" {
			other = specA
		}
		merged = mergeScenes(winner, other)
	case validA:
		winner, merged, rep.Winner = specA, specA, "A"
	case validB:
		winner, merged, rep.Winner = specB, specB, "B"
	default:
		winner, merged, rep.Winner = specA, specA, "none"
	}
	rep.Merged = len(merged.Objects)
	return winner, merged, rep
}

// mergeScenes keeps base and adds the other region's non-ground forms to the side,
// capped, re-using base's sky and light — the referee's synthesis of both directors.
func mergeScenes(base, add brain.SceneSpec) brain.SceneSpec {
	out := base
	out.Objects = append([]brain.ObjSpec(nil), base.Objects...)
	for _, o := range add.Objects {
		if o.Kind == "plane" { // one ground only
			continue
		}
		o.X -= 3.5 // set the other director's forms beside the winner's
		o.Z += 1.5
		out.Objects = append(out.Objects, o)
		if len(out.Objects) >= maxRegionObjects {
			break
		}
	}
	return out
}
