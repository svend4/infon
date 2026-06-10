package raydir

import (
	"math"
	"testing"
)

func vdist(a, b SceneVector) float64 {
	s := 0.0
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}

func TestSceneVectorHexagram(t *testing.T) {
	v := SceneVector{0.9, 0.1, 0.8, 0.2, 0.7, 0.3} // yang, yin, yang, yin, yang, yin
	want := Hexagram{Lines: [6]bool{true, false, true, false, true, false}}
	if got := v.Hexagram(); got != want {
		t.Errorf("Hexagram() = %v, want %v", got.Lines, want.Lines)
	}
	// VectorFromHexagram round-trips through the signs
	for n := 0; n < 64; n++ {
		h := HexagramFromNumber(n)
		if VectorFromHexagram(h).Hexagram() != h {
			t.Fatalf("VectorFromHexagram(%06b) lost its signs", n)
		}
	}
}

func TestSceneReadRoundTrip(t *testing.T) {
	// axes chosen clear of 0.5 and of the density quantisation edges so the reader
	// recovers both the values (to tolerance) and the signs (exactly).
	in := SceneVector{0.7, 0.8, 0.8, 0.7, 0.6, 0.2}
	spec := in.SceneSpec()
	out := ReadScene(spec)
	for i := range in {
		tol := 0.03
		if i == AxDensity {
			tol = 0.07 // density survives only to the nearest form (1/8)
		}
		if math.Abs(in[i]-out[i]) > tol {
			t.Errorf("axis %d: read %.3f, want %.3f (tol %.2f)", i, out[i], in[i], tol)
		}
	}
	if in.Hexagram() != out.Hexagram() {
		t.Errorf("reader lost the hexagram: %v -> %v", in.Hexagram().Lines, out.Hexagram().Lines)
	}
}

func TestReadSceneRecoversSigns(t *testing.T) {
	// the hexagram must survive the round trip for many worlds, with axes set away
	// from the boundaries so each sign is unambiguous.
	for n := 0; n < 64; n++ {
		h := HexagramFromNumber(n)
		v := VectorFromHexagram(h) // yang->0.75, yin->0.25
		if got := ReadScene(v.SceneSpec()).Hexagram(); got != h {
			t.Fatalf("hex %06b: reader returned %06b", n, got.Number())
		}
	}
}

func TestLearningDirectorConverges(t *testing.T) {
	vm := ViewerModel{Pref: SceneVector{0.8, 0.2, 0.85, 0.7, 0.6, 0.2}, Sigma: 0.3}
	d := NewLearningDirector(1)
	start := vdist(d.Best(), vm.Pref)
	for i := 0; i < 250; i++ {
		d.Round(vm.Engagement)
	}
	end := vdist(d.Best(), vm.Pref)
	if end > start*0.35 {
		t.Errorf("director should converge toward the viewer's taste: start %.3f -> end %.3f", start, end)
	}
	if d.Engagement() < 0.9 {
		t.Errorf("engagement should climb high, got %.3f", d.Engagement())
	}
	// having learned the taste, the chosen hexagram should match the viewer's
	if d.Best().Hexagram() != vm.Pref.Hexagram() {
		t.Errorf("learned hexagram %06b != viewer's %06b",
			d.Best().Hexagram().Number(), vm.Pref.Hexagram().Number())
	}
}

func TestSceneVectorMood(t *testing.T) {
	cases := []struct {
		v    SceneVector
		want string
	}{
		{SceneVector{0.0, 0.9, 0.9, 0.9, 0.7, 0.9}, "joyful"},  // warm, bright, busy, glowing
		{SceneVector{0.0, 0.9, 0.1, 0.8, 0.2, 0.1}, "serene"},  // warm, bright, calm
		{SceneVector{0.9, 0.1, 0.9, 0.1, 0.7, 0.6}, "ominous"}, // foggy, cold, busy
		{SceneVector{0.9, 0.1, 0.1, 0.1, 0.2, 0.0}, "somber"},  // foggy, cold, still
	}
	for _, c := range cases {
		if got := c.v.Mood(); got != c.want {
			t.Errorf("Mood(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}
