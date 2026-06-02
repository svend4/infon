package sona

import (
	"math/big"
	"testing"
	"time"

	"github.com/svend4/infon/pkg/tangram"
)

func TestMelodyDeterministic(t *testing.T) {
	a := Melody(tangram.Cat())
	b := Melody(tangram.Cat())
	if len(a) != len(b) {
		t.Fatalf("len differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("note %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestMelodyLengthMatchesCells(t *testing.T) {
	f := tangram.Cat()
	if got, want := len(Melody(f)), f.H*f.W; got != want {
		t.Fatalf("melody has %d notes, want one per cell = %d", got, want)
	}
}

// Digits must be the exact base-AlphabetSize expansion of Figure.Number.
func TestDigitsRebuildNumber(t *testing.T) {
	f := tangram.Swan()
	base := big.NewInt(int64(tangram.AlphabetSize))
	n := new(big.Int)
	for _, d := range Digits(f) {
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(d)))
	}
	if n.Cmp(f.Number()) != 0 {
		t.Fatalf("rebuilt %s != Number %s", n, f.Number())
	}
}

func TestRestMatchesBlankCells(t *testing.T) {
	f := tangram.Cat()
	ds := Digits(f)
	notes := Melody(f)
	blanks := 0
	for i, d := range ds {
		if (d == 0) != notes[i].Rest {
			t.Fatalf("cell %d digit=%d but Rest=%v", i, d, notes[i].Rest)
		}
		if d == 0 {
			blanks++
		}
	}
	if blanks == 0 {
		t.Fatal("expected at least one blank cell (rest) in Cat")
	}
}

func TestMelodyChangesWithShape(t *testing.T) {
	cat := Melody(tangram.Cat())
	swan := Melody(tangram.Swan())
	same := len(cat) == len(swan)
	if same {
		for i := range cat {
			if cat[i] != swan[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("different figures produced identical melodies")
	}
}

func TestPitchMapping(t *testing.T) {
	// digit 1 -> base of the scale; higher digits never go below it.
	if MidiFor(1) != 57 {
		t.Fatalf("MidiFor(1)=%d, want 57 (A3)", MidiFor(1))
	}
	if FreqFor(69) < 439.9 || FreqFor(69) > 440.1 {
		t.Fatalf("FreqFor(69)=%.3f, want 440", FreqFor(69))
	}
	for d := 2; d <= 15; d++ {
		if MidiFor(d) < MidiFor(1) {
			t.Fatalf("MidiFor(%d)=%d below base", d, MidiFor(d))
		}
	}
}

func TestSynthLength(t *testing.T) {
	rate := 8000
	notes := []Note{
		{Freq: 440, Dur: 100 * time.Millisecond},
		{Rest: true, Dur: 50 * time.Millisecond},
		{Freq: 660, Dur: 100 * time.Millisecond},
	}
	want := 0
	for _, n := range notes {
		want += int(n.Dur.Seconds() * float64(rate))
	}
	if got := len(Synth(notes, rate)); got != want {
		t.Fatalf("synth produced %d samples, want %d", got, want)
	}
}

func TestWAVHeader(t *testing.T) {
	rate := 8000
	pcm := Synth(Melody(tangram.Cat()), rate)
	w := WAV(pcm, rate)
	if len(w) != 44+len(pcm)*2 {
		t.Fatalf("wav len %d, want %d", len(w), 44+len(pcm)*2)
	}
	if string(w[0:4]) != "RIFF" || string(w[8:12]) != "WAVE" ||
		string(w[12:16]) != "fmt " || string(w[36:40]) != "data" {
		t.Fatalf("bad RIFF/WAVE/fmt/data tags")
	}
	dataLen := int(w[40]) | int(w[41])<<8 | int(w[42])<<16 | int(w[43])<<24
	if dataLen != len(pcm)*2 {
		t.Fatalf("data chunk len %d, want %d", dataLen, len(pcm)*2)
	}
}
