package arecibo

import (
	"testing"

	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/tangram"
)

func TestRoundTripAndLegend(t *testing.T) {
	for _, name := range []string{"cat", "swan", "boat"} {
		f, _ := tangram.Named(name)
		m, got, err := Decode(Encode(f))
		if err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		if !m.OK {
			t.Fatalf("%s: checksum not OK", name)
		}
		if got.Number().Cmp(f.Number()) != 0 {
			t.Fatalf("%s: figure mismatch", name)
		}
		if m.Radix != tangram.AlphabetSize {
			t.Fatalf("%s: learned radix %d", name, m.Radix)
		}
		// the legend the receiver LEARNED equals each symbol's real shape
		for d := 0; d < m.Radix; d++ {
			nm := tangram.GlyphForDigit(d)
			for sy := 0; sy < m.S; sy++ {
				for sx := 0; sx < m.S; sx++ {
					if m.LegendBit(d, sx, sy) != glyphset.Coverage(nm, sx, sy, m.S) {
						t.Fatalf("%s: legend symbol %d differs at (%d,%d)", name, d, sx, sy)
					}
				}
			}
		}
	}
}

func TestCorruptionCaught(t *testing.T) {
	f, _ := tangram.Named("cat")
	frame := Encode(f)
	m, _, err := Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	payStart := 9 + (m.Radix + 1) + (m.S + 1) + m.Radix*m.S*m.S + (m.H + 1) + (m.W + 1)
	frame[payStart/8] ^= 1 << uint(7-payStart%8) // flip the first payload bit
	if _, _, err := Decode(frame); err == nil {
		t.Fatal("a flipped payload bit was not caught by the checksum")
	}
}
