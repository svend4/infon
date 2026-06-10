package raydir

import (
	"math/rand"
	"strings"
)

// hexagram.go reads the world from an I-Ching hexagram, the way the dream hackers'
// "DNA of the tonal" does — six lines, two trigrams, sixty-four readings. A
// hexagram is a compact, meaningful seed: its two trigrams (heaven, lake, fire,
// thunder, wind, water, mountain, earth) name an upper and a lower theme, which
// compose a director prompt, and its six-bit number gives a deterministic seed. So
// casting a hexagram conjures one specific, reproducible world. (Kin to the
// I-Ching-style positional id in pkg/tangram.)

// trigram is one of the eight three-line figures, with a name and a theme.
type trigram struct {
	name  string
	theme string
}

// trigrams indexed by their 3-bit value (bit i = line i, bottom-up; 1 = yang).
var trigrams = [8]trigram{
	{"Earth", "a plain of stone and earth"},            // 000
	{"Mountain", "mountains, rock and crystals"},       // 001
	{"Water", "deep water and a shore"},                // 010
	{"Wind", "wind through a forest of trees"},         // 011
	{"Thunder", "a storm over a forest"},               // 100
	{"Fire", "fire and a glowing beacon"},              // 101
	{"Lake", "a calm lake with water and reflections"}, // 110
	{"Heaven", "an open sky of drifting clouds"},       // 111
}

// Hexagram is six lines (true = yang), lines[0] the bottom.
type Hexagram struct {
	Lines [6]bool
}

// lowerIdx and upperIdx are the 3-bit values of the lower and upper trigrams.
func (h Hexagram) lowerIdx() int {
	return b2i(h.Lines[0]) | b2i(h.Lines[1])<<1 | b2i(h.Lines[2])<<2
}
func (h Hexagram) upperIdx() int {
	return b2i(h.Lines[3]) | b2i(h.Lines[4])<<1 | b2i(h.Lines[5])<<2
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Number is the hexagram's six-bit value, 0..63.
func (h Hexagram) Number() int { return h.lowerIdx() | h.upperIdx()<<3 }

// Name is the upper/lower trigram pair, e.g. "Water over Mountain".
func (h Hexagram) Name() string {
	return trigrams[h.upperIdx()].name + " over " + trigrams[h.lowerIdx()].name
}

// Prompt composes a director prompt from the two trigrams' themes — the world the
// hexagram conjures.
func (h Hexagram) Prompt() string {
	return trigrams[h.upperIdx()].theme + " above " + trigrams[h.lowerIdx()].theme
}

// Seed is a deterministic seed derived from the hexagram's number.
func (h Hexagram) Seed() int64 {
	return int64(h.Number())*2654435761 + 1
}

// CastHexagram casts six lines deterministically from a seed (like throwing coins).
func CastHexagram(seed int64) Hexagram {
	rng := rand.New(rand.NewSource(seed))
	var h Hexagram
	for i := range h.Lines {
		h.Lines[i] = rng.Intn(2) == 1
	}
	return h
}

// ParseHexagram reads six lines from a string of 1/0 or y/n (bottom-to-top), e.g.
// "101010" or "yynnyn"; ok=false if it isn't six valid characters.
func ParseHexagram(s string) (Hexagram, bool) {
	s = strings.TrimSpace(s)
	if len(s) != 6 {
		return Hexagram{}, false
	}
	var h Hexagram
	for i, c := range s {
		switch c {
		case '1', 'y', 'Y':
			h.Lines[i] = true
		case '0', 'n', 'N':
			h.Lines[i] = false
		default:
			return Hexagram{}, false
		}
	}
	return h, true
}
