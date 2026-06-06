// Package sona sonifies a tangram figure: it reads the figure's cells in the
// same row-major order that pkg/tangram reads them to form the figure's NUMBER,
// and turns each cell-digit into a musical note. The shape you SEE becomes the
// tune you HEAR — the third face of the project's "see, hear, summon" triad
// (the same address that renders a figure and summons a creature also sings).
//
// The mapping is deterministic and pose-explicit: digit 0 (a blank cell) is a
// rest; digits 1..15 map onto a major-pentatonic scale climbing ~three octaves,
// so two figures sound different exactly when their grids differ.
package sona

import (
	"math"
	"math/big"
	"time"

	"github.com/svend4/infon/pkg/tangram"
)

// DefaultStep is the duration of one cell (one note or rest).
const DefaultStep = 160 * time.Millisecond

// DefaultRate is the synthesis sample rate (Hz), mono 16-bit PCM.
const DefaultRate = 44100

// Note is one step of the melody. A Rest is a silent cell (digit 0).
type Note struct {
	Rest  bool
	Digit int           // 0..15, the figure's cell glyph index
	Midi  int           // MIDI pitch (meaningless when Rest)
	Freq  float64       // Hz (0 when Rest)
	Dur   time.Duration // step length
}

// pentatonic scale degrees (semitone offsets) — a friendly, never-dissonant set.
var penta = []int{0, 2, 4, 7, 9}

// MidiFor maps a non-zero cell digit (1..15) to a MIDI pitch. The 15 non-blank
// glyphs climb the pentatonic scale from A3 (57) across three octaves.
func MidiFor(digit int) int {
	i := digit - 1
	if i < 0 {
		i = 0
	}
	return 57 + 12*(i/len(penta)) + penta[i%len(penta)]
}

// FreqFor converts a MIDI pitch to a frequency in Hz (A4 = 69 = 440 Hz).
func FreqFor(midi int) float64 { return 440 * math.Pow(2, float64(midi-69)/12.0) }

// Digits returns the figure's H*W cells read row-major as base-AlphabetSize
// digits — exactly the sequence pkg/tangram folds into Figure.Number (leading
// blank cells preserved). This is the figure "as read", cell by cell.
func Digits(f tangram.Figure) []int {
	total := f.H * f.W
	if total <= 0 {
		return nil
	}
	base := big.NewInt(int64(tangram.AlphabetSize))
	q := new(big.Int).Set(f.Number())
	r := new(big.Int)
	digs := make([]int, total)
	for i := total - 1; i >= 0; i-- {
		q.QuoRem(q, base, r)
		digs[i] = int(r.Int64())
	}
	return digs
}

// MelodyWithStep turns a figure into a melody using the given step duration.
func MelodyWithStep(f tangram.Figure, step time.Duration) []Note {
	if step <= 0 {
		step = DefaultStep
	}
	ds := Digits(f)
	notes := make([]Note, 0, len(ds))
	for _, d := range ds {
		if d == 0 {
			notes = append(notes, Note{Rest: true, Digit: 0, Dur: step})
			continue
		}
		m := MidiFor(d)
		notes = append(notes, Note{Digit: d, Midi: m, Freq: FreqFor(m), Dur: step})
	}
	return notes
}

// Melody is MelodyWithStep at the default step.
func Melody(f tangram.Figure) []Note { return MelodyWithStep(f, DefaultStep) }

func clip16(v float64) int16 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}

// Synth renders notes to mono 16-bit PCM at the given sample rate. Each voiced
// note is a soft sine stack (fundamental + two harmonics) under a click-free
// attack/release envelope; rests are silence.
func Synth(notes []Note, rate int) []int16 {
	if rate <= 0 {
		rate = DefaultRate
	}
	var out []int16
	for _, n := range notes {
		ns := int(n.Dur.Seconds() * float64(rate))
		if ns <= 0 {
			continue
		}
		if n.Rest || n.Freq <= 0 {
			out = append(out, make([]int16, ns)...)
			continue
		}
		atk := int(0.012 * float64(rate))
		rel := ns / 4
		if max := int(0.06 * float64(rate)); rel > max {
			rel = max
		}
		for i := 0; i < ns; i++ {
			t := float64(i) / float64(rate)
			s := 0.60*math.Sin(2*math.Pi*n.Freq*t) +
				0.25*math.Sin(2*math.Pi*2*n.Freq*t) +
				0.12*math.Sin(2*math.Pi*3*n.Freq*t)
			env := 1.0
			if atk > 0 && i < atk {
				env = float64(i) / float64(atk)
			} else if rel > 0 && i > ns-rel {
				env = float64(ns-i) / float64(rel)
			}
			out = append(out, clip16(env*0.28*s*32767))
		}
	}
	return out
}

// WAV wraps mono 16-bit PCM samples in a canonical RIFF/WAVE container.
func WAV(pcm []int16, rate int) []byte {
	if rate <= 0 {
		rate = DefaultRate
	}
	const channels, bits = 1, 16
	dataLen := len(pcm) * 2
	byteRate := rate * channels * bits / 8
	blockAlign := channels * bits / 8
	b := make([]byte, 0, 44+dataLen)
	put32 := func(v uint32) { b = append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24)) }
	put16 := func(v uint16) { b = append(b, byte(v), byte(v>>8)) }
	b = append(b, "RIFF"...)
	put32(uint32(36 + dataLen))
	b = append(b, "WAVE"...)
	b = append(b, "fmt "...)
	put32(16)
	put16(1) // PCM
	put16(channels)
	put32(uint32(rate))
	put32(uint32(byteRate))
	put16(uint16(blockAlign))
	put16(bits)
	b = append(b, "data"...)
	put32(uint32(dataLen))
	for _, s := range pcm {
		b = append(b, byte(uint16(s)), byte(uint16(s)>>8))
	}
	return b
}
