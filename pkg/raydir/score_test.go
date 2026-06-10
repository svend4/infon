package raydir

import (
	"math"
	"testing"
)

func inSet(v int, set []int) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// Day plays major, night plays minor.
func TestScoreDayNight(t *testing.T) {
	day := ScoreForNight(0.1, 0.5, "")
	if !inSet(4, day.Scale) { // a major third
		t.Errorf("a day score should be major (contain semitone 4): %v", day.Scale)
	}
	night := ScoreForNight(0.9, 0.5, "")
	if inSet(4, night.Scale) || !inSet(3, night.Scale) { // a minor third, no major third
		t.Errorf("a night score should be minor (semitone 3, not 4): %v", night.Scale)
	}
	if night.BPM >= day.BPM {
		t.Errorf("night should be slower than day: night %d day %d", night.BPM, day.BPM)
	}
}

// A livelier world plays faster; calm is unhurried and pentatonic.
func TestScoreTempoAndMood(t *testing.T) {
	quiet := ScoreForNight(0.1, 0.0, "")
	busy := ScoreForNight(0.1, 1.0, "")
	if busy.BPM <= quiet.BPM {
		t.Errorf("a livelier world should play faster: quiet %d busy %d", quiet.BPM, busy.BPM)
	}
	calm := ScoreForNight(0.1, 0.5, "calm")
	if len(calm.Scale) != len(ScalePentMaj) {
		t.Errorf("a calm day should be pentatonic, got scale %v", calm.Scale)
	}
	restless := ScoreForNight(0.1, 0.5, "restless")
	if restless.BPM <= ScoreForNight(0.1, 0.5, "").BPM {
		t.Error("restless should be quicker than neutral")
	}
}

// Every melody note lands on a pitch of the chosen scale.
func TestMelodyInScale(t *testing.T) {
	p := ScoreForNight(0.1, 0.5, "") // major
	for _, f := range NoteFreqs(p, 200, 12345) {
		semi := int(math.Round(12 * math.Log2(f/p.Root)))
		pc := ((semi % 12) + 12) % 12
		if !inSet(pc, p.Scale) {
			t.Fatalf("note %.1f Hz is pitch class %d, not in scale %v", f, pc, p.Scale)
		}
	}
}

// The melody is deterministic in its seed.
func TestMelodyDeterministic(t *testing.T) {
	p := ScoreForNight(0.5, 0.5, "")
	a := MelodyPCM(p, 16000, 1.5, 7)
	b := MelodyPCM(p, 16000, 1.5, 7)
	if len(a) != len(b) {
		t.Fatal("same params should give same length")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("same seed should give identical PCM (first diff at %d)", i)
		}
	}
	c := MelodyPCM(p, 16000, 1.5, 8)
	diff := false
	for i := range a {
		if a[i] != c[i] {
			diff = true
			break
		}
	}
	if !diff {
		t.Error("a different seed should change the melody")
	}
}

// The rendered melody actually makes sound (non-trivial energy) and is a real WAV.
func TestMelodyNotSilentAndWAV(t *testing.T) {
	p := ScoreForNight(0.2, 0.6, "restless")
	pcm := MelodyPCM(p, 16000, 2.0, 3)
	var energy float64
	for _, s := range pcm {
		energy += math.Abs(float64(s))
	}
	if energy/float64(len(pcm)) < 100 {
		t.Errorf("the melody should make sound, mean |amp| %.1f", energy/float64(len(pcm)))
	}
	wav := ScoreWAV(p, 16000, 0.5, 3)
	if len(wav) < 44 || string(wav[0:4]) != "RIFF" {
		t.Error("ScoreWAV should produce a RIFF/WAV file")
	}
}

// The world derives a score from its own state.
func TestWorldScore(t *testing.T) {
	w := NewWorld()
	w.SetTime(0.5) // noon-ish: a day score
	if s := w.Score(); inSet(4, s.Scale) == false {
		t.Errorf("a daytime world should score major, got %v", s.Scale)
	}
	w.SetTime(0.0) // midnight
	if s := w.Score(); inSet(4, s.Scale) {
		t.Errorf("a midnight world should not be major, got %v", s.Scale)
	}
}
