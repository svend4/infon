package sona

import (
	"testing"
	"time"
)

func TestSeriesMelodyRisesWithValue(t *testing.T) {
	notes := SeriesMelody([]int{1, 5, 9, 20}, 100*time.Millisecond)
	if len(notes) != 4 {
		t.Fatalf("got %d notes, want 4", len(notes))
	}
	for i := 1; i < len(notes); i++ {
		if notes[i].Midi < notes[i-1].Midi {
			t.Fatalf("rising values should not lower the pitch: note %d midi %d < %d", i, notes[i].Midi, notes[i-1].Midi)
		}
	}
	for _, n := range notes {
		if n.Rest || n.Freq != FreqFor(n.Midi) {
			t.Fatalf("voiced note expected: %+v", n)
		}
	}
}

func TestSeriesMelodyFlat(t *testing.T) {
	notes := SeriesMelody([]int{7, 7, 7}, 0)
	for i := 1; i < len(notes); i++ {
		if notes[i].Midi != notes[0].Midi {
			t.Fatalf("flat series should be one pitch")
		}
	}
}

func TestSeriesMelodyDeterministicAndEmpty(t *testing.T) {
	a := SeriesMelody([]int{3, 1, 4, 1, 5, 9}, DefaultStep)
	b := SeriesMelody([]int{3, 1, 4, 1, 5, 9}, DefaultStep)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
	if SeriesMelody(nil, 0) != nil {
		t.Fatal("empty series should yield nil")
	}
}
