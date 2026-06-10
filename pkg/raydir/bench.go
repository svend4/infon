package raydir

import "github.com/svend4/infon/pkg/brain"

// bench.go evaluates a director (any brain) by asking it to author many scenes and
// scoring the results: are they renderable, how rich, how varied? It gives a quick,
// objective read on a live model before a session — built on the same AuthorScene
// path the world uses, so it measures exactly what walkers would see.

// BenchResult is a director's score over a batch of prompts.
type BenchResult struct {
	N          int            // prompts asked
	Renderable int            // scenes with at least one renderable object
	Objects    int            // total objects authored
	Kinds      map[string]int // object kinds used (variety)
	Errors     int            // transport failures
}

// AvgObjects is the mean object count per renderable scene.
func (r BenchResult) AvgObjects() float64 {
	if r.Renderable == 0 {
		return 0
	}
	return float64(r.Objects) / float64(r.Renderable)
}

// Variety is the number of distinct object kinds the director used.
func (r BenchResult) Variety() int { return len(r.Kinds) }

// BenchDirector asks the brain to author a scene for each prompt and tallies how
// renderable, rich and varied the results are.
func BenchDirector(b brain.Brain, prompts []string) BenchResult {
	res := BenchResult{Kinds: map[string]int{}}
	for _, p := range prompts {
		res.N++
		_, spec, err := AuthorScene(b, p)
		if err != nil {
			res.Errors++
			continue
		}
		if hasRenderable(spec) {
			res.Renderable++
		}
		for _, o := range spec.Objects {
			if !validObj(o) {
				continue
			}
			res.Objects++
			k := o.Kind
			if k == "" {
				k = "sphere"
			}
			res.Kinds[k]++
		}
	}
	return res
}
