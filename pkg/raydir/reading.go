package raydir

import (
	"fmt"
	"math/rand"
	"strings"
)

// reading.go adds the divinatory half of the I-Ching that the Gray walk lacked:
// changing lines. A reading casts six lines by the three-coin method; each line is
// young or old, and the OLD lines are "changing" — they flip — so a primary
// hexagram A transforms into a relating hexagram B. That is the oracle's narrative
// arc, A -> B, and it is rendered (cmd/rayreading) as a transition: the world of A
// dissolves into voxel blocks and the world of B forms from them (materialize.go).
// It binds infon to pro2 deeper than the static Gray walk: a reading is a story.

// Reading is an I-Ching reading: a primary hexagram and which lines are changing.
type Reading struct {
	Primary  Hexagram
	Changing [6]bool
}

// CastReading casts a reading from a seed by the three-coin method: each line sums
// three coins (2 or 3 each) to 6..9 — 6 old-yin and 9 old-yang are the changing
// lines, 7 young-yang and 8 young-yin are stable. Deterministic.
func CastReading(seed int64) Reading {
	rng := rand.New(rand.NewSource(seed))
	var r Reading
	for i := 0; i < 6; i++ {
		v := 0
		for c := 0; c < 3; c++ {
			v += 2 + rng.Intn(2) // each coin 2 or 3 -> v in 6..9
		}
		r.Primary.Lines[i] = v == 7 || v == 9 // yang on young-yang / old-yang
		r.Changing[i] = v == 6 || v == 9      // old lines change
	}
	return r
}

// Mask is the six-bit changing-line mask (bit 0 = bottom line).
func (r Reading) Mask() int {
	m := 0
	for i := 0; i < 6; i++ {
		if r.Changing[i] {
			m |= 1 << i
		}
	}
	return m
}

// Relating is the hexagram the reading becomes: the primary with its changing lines
// flipped. With no changing lines it equals the primary.
func (r Reading) Relating() Hexagram {
	h := r.Primary
	for i := 0; i < 6; i++ {
		if r.Changing[i] {
			h.Lines[i] = !h.Lines[i]
		}
	}
	return h
}

// Stable reports whether nothing changes (no changing lines): the answer stands.
func (r Reading) Stable() bool { return r.Mask() == 0 }

// ChangingLines lists the 1-based positions of the changing lines (bottom = 1).
func (r Reading) ChangingLines() []int {
	var out []int
	for i := 0; i < 6; i++ {
		if r.Changing[i] {
			out = append(out, i+1)
		}
	}
	return out
}

// String renders the reading as "A -> B (changing lines ...)" or "A (stable)".
func (r Reading) String() string {
	if r.Stable() {
		return r.Primary.Name() + " (stable)"
	}
	ls := make([]string, 0, 6)
	for _, n := range r.ChangingLines() {
		ls = append(ls, fmt.Sprintf("%d", n))
	}
	return fmt.Sprintf("%s → %s (changing lines %s)", r.Primary.Name(), r.Relating().Name(), strings.Join(ls, ","))
}
