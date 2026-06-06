package sona

import "time"

// series.go sonifies a numeric series (e.g. a match's per-tick total HP) rather
// than a figure's cells: each value becomes a pentatonic note whose pitch rises
// with the value, so a story told in numbers — a world losing energy as units
// fall — is heard as a descending tune. Deterministic.

func seriesRange(v []int) (mn, mx int) {
	mn, mx = v[0], v[0]
	for _, x := range v {
		if x < mn {
			mn = x
		}
		if x > mx {
			mx = x
		}
	}
	return mn, mx
}

// SeriesMelody maps a value series to a major-pentatonic melody (one voiced note
// per value), pitch rising with the value across ~two octaves.
func SeriesMelody(values []int, step time.Duration) []Note {
	if step <= 0 {
		step = DefaultStep
	}
	if len(values) == 0 {
		return nil
	}
	mn, mx := seriesRange(values)
	notes := make([]Note, 0, len(values))
	for _, v := range values {
		idx := len(penta) * 2 / 2 // default mid (=len(penta), i.e. octave 1 root)
		if mx > mn {
			idx = (v - mn) * (2*len(penta) - 1) / (mx - mn) // 0 .. 2*len(penta)-1
		}
		m := 57 + 12*(idx/len(penta)) + penta[idx%len(penta)]
		notes = append(notes, Note{Midi: m, Freq: FreqFor(m), Dur: step})
	}
	return notes
}
