package raydir

import (
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// memory.go gives the director a memory, in the spirit of svend4/infom's GraphRAG:
// every region grown is remembered (its place name and the prompt that made it),
// and when a new region is authored the most thematically similar past place is
// recalled, so the world can echo itself — "reminiscent of the Forest you saw
// before" — instead of forgetting. Similarity is light token overlap (Jaccard);
// the remembered places also form a graph (edges between similar ones), a small
// knowledge graph of the world that can be laid out and drawn.

var memStop = map[string]bool{
	"a": true, "an": true, "the": true, "of": true, "with": true, "and": true,
	"at": true, "in": true, "on": true, "by": true, "to": true, "is": true,
}

// tokenize lowercases and splits text into content words (stopwords dropped).
func tokenize(s string) map[string]bool {
	out := map[string]bool{}
	word := strings.Builder{}
	flush := func() {
		if word.Len() == 0 {
			return
		}
		w := word.String()
		word.Reset()
		if len(w) >= 3 && !memStop[w] {
			out[w] = true
		}
	}
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' {
			word.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// jaccard is |A∩B| / |A∪B| for two token sets (0 when either is empty).
func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for w := range a {
		if b[w] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// memEntry is one remembered region.
type memEntry struct {
	Index  int
	Name   string
	Prompt string
	toks   map[string]bool
}

// Memory records the regions the director has grown.
type Memory struct {
	entries []memEntry
}

// Remember stores a region's place name and the prompt that authored it.
func (m *Memory) Remember(index int, name, prompt string) {
	m.entries = append(m.entries, memEntry{Index: index, Name: name, Prompt: prompt, toks: tokenize(name + " " + prompt)})
}

// Len is how many regions are remembered.
func (m *Memory) Len() int { return len(m.entries) }

// Recall returns the remembered region most thematically similar to `query` and its
// similarity score (0..1); ok=false if nothing is remembered or nothing overlaps.
func (m *Memory) Recall(query string) (memEntry, float64, bool) {
	qt := tokenize(query)
	best, bestScore := memEntry{}, 0.0
	for _, e := range m.entries {
		if s := jaccard(qt, e.toks); s > bestScore {
			best, bestScore = e, s
		}
	}
	return best, bestScore, bestScore > 0
}

// Graph builds a knowledge graph of the remembered places: a bubble per region,
// with a transit between any two whose thematic similarity is at least threshold.
func (m *Memory) Graph(threshold float64) *BubbleGraph {
	g := NewBubbleGraph()
	ids := make([]int, len(m.entries))
	for i, e := range m.entries {
		ids[i] = g.Add(e.Name, raytrace.Vec3{}, brain.SceneSpec{Name: e.Name})
	}
	for i := 0; i < len(m.entries); i++ {
		for j := i + 1; j < len(m.entries); j++ {
			if jaccard(m.entries[i].toks, m.entries[j].toks) >= threshold {
				g.Link(ids[i], ids[j])
			}
		}
	}
	return g
}
