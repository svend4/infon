package raydir

import "testing"

func scaleEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestScoreForVectorMapping(t *testing.T) {
	day := ScoreForVector(SceneVector{AxSun: 0.9})
	night := ScoreForVector(SceneVector{AxSun: 0.1})
	if day.Root <= night.Root {
		t.Errorf("day should be higher pitched than night: %.1f vs %.1f", day.Root, night.Root)
	}
	warm := ScoreForVector(SceneVector{AxWarm: 0.9, AxDensity: 0.9})
	cold := ScoreForVector(SceneVector{AxWarm: 0.1, AxDensity: 0.9})
	if !scaleEq(warm.Scale, ScaleMajor) {
		t.Errorf("a warm busy world should be major")
	}
	if !scaleEq(cold.Scale, ScaleMinor) {
		t.Errorf("a cool busy world should be minor")
	}
	dense := ScoreForVector(SceneVector{AxDensity: 0.9})
	sparse := ScoreForVector(SceneVector{AxDensity: 0.1})
	if dense.BPM <= sparse.BPM {
		t.Errorf("a denser world should play faster: %d vs %d", dense.BPM, sparse.BPM)
	}
	clear := ScoreForVector(SceneVector{AxDensity: 0.6})
	foggy := ScoreForVector(SceneVector{AxDensity: 0.6, AxFog: 1, AxGlow: 0})
	if foggy.BPM >= clear.BPM {
		t.Errorf("fog should slow the tempo: %d vs %d", foggy.BPM, clear.BPM)
	}
	bright := ScoreForVector(SceneVector{AxGlow: 1})
	dull := ScoreForVector(SceneVector{AxGlow: 0})
	if bright.Bright <= dull.Bright {
		t.Errorf("glow should brighten the timbre: %.2f vs %.2f", bright.Bright, dull.Bright)
	}
}

func TestTourPCM(t *testing.T) {
	walk := GrayCode()
	pcm := TourPCM(walk, 8000, 0.05, 1)
	if len(pcm) == 0 {
		t.Fatal("the tour should produce audio")
	}
	// roughly one phrase per world
	if got, want := len(pcm), int(0.05*8000)*len(walk); got < want/2 {
		t.Errorf("tour too short: %d samples for %d worlds", got, len(walk))
	}
}

// signChanges counts zero crossings (a cheap proxy for pitch).
func signChanges(s []int16) int {
	n := 0
	for i := 1; i < len(s); i++ {
		if (s[i-1] < 0) != (s[i] < 0) {
			n++
		}
	}
	return n
}

func TestSonifySeriesPitchTracksValue(t *testing.T) {
	rate := 8000
	pcm := SonifySeries([]float64{0, 1}, rate, 0.25, 220, 440) // low then high
	if len(pcm) != 2*int(0.25*float64(rate)) {
		t.Fatalf("unexpected length %d", len(pcm))
	}
	half := len(pcm) / 2
	lo := signChanges(pcm[:half])
	hi := signChanges(pcm[half:])
	if hi <= lo {
		t.Errorf("a higher value should sound higher: crossings %d then %d", lo, hi)
	}
}

func TestSonifySeriesEmpty(t *testing.T) {
	if SonifySeries(nil, 8000, 0.1, 200, 200) != nil {
		t.Error("empty series should give no audio")
	}
}
