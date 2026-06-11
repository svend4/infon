package raydir

import (
	"math"
	"testing"
)

func TestSeasonalWorldOscillates(t *testing.T) {
	s := NewSeasonalWorld(14, 14, 30, 7, 80)
	mn, mx := 1<<30, 0
	for i := 0; i < 240; i++ { // three years
		s.Step()
		p := s.Population()
		if p < mn {
			mn = p
		}
		if p > mx {
			mx = p
		}
	}
	if s.Population() == 0 {
		t.Fatal("the seasonal world went extinct — life should persist between rains")
	}
	if mx-mn < 25 {
		t.Errorf("expected a clear seasonal boom/bust, got swing %d..%d", mn, mx)
	}
}

func TestSeasonalRainMovesAndWetnessCycles(t *testing.T) {
	s := NewSeasonalWorld(14, 14, 10, 1, 100)
	minW, maxW := 2.0, -1.0
	minZ, maxZ := math.Inf(1), math.Inf(-1)
	for i := 0; i < 100; i++ { // a full year
		w, z := s.Wetness(), s.RainBandZ()
		minW, maxW = math.Min(minW, w), math.Max(maxW, w)
		minZ, maxZ = math.Min(minZ, z), math.Max(maxZ, z)
		s.Step()
	}
	if maxW-minW < 0.8 {
		t.Errorf("wetness should swing across the year: %.2f..%.2f", minW, maxW)
	}
	if maxZ-minZ < 20 {
		t.Errorf("the rain front should sweep across the world: %.1f..%.1f", minZ, maxZ)
	}
}

func TestSeasonalDeterministic(t *testing.T) {
	run := func() int {
		s := NewSeasonalWorld(12, 12, 20, 3, 90)
		for i := 0; i < 180; i++ {
			s.Step()
		}
		return s.Population()
	}
	if a, b := run(), run(); a != b {
		t.Errorf("same seed must replay identically: %d vs %d", a, b)
	}
}

func TestSeasonName(t *testing.T) {
	s := NewSeasonalWorld(8, 8, 5, 1, 8) // a year of 8 ticks: 2 ticks per season
	want := []string{"spring", "spring", "summer", "summer", "autumn", "autumn", "winter", "winter"}
	got := []string{s.SeasonName()}
	for i := 0; i < 7; i++ {
		s.Step()
		got = append(got, s.SeasonName())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tick %d: season %q, want %q", i, got[i], want[i])
		}
	}
}
