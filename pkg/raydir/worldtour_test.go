package raydir

import (
	"math"
	"testing"
)

func TestTourMorphCorners(t *testing.T) {
	walk := GrayCode() // all 64 hexagrams, one line apart
	m := TourMorph(walk, 1)
	if len(m) != 64 {
		t.Fatalf("perStep=1 should give one vector per hexagram, got %d", len(m))
	}
	for i, v := range m {
		if v.Hexagram() != walk[i] {
			t.Fatalf("corner %d lost its hexagram", i)
		}
	}
	// consecutive corners differ in exactly one axis (the Gray single-line change)
	for i := 1; i < len(m); i++ {
		diff := 0
		for k := 0; k < 6; k++ {
			if m[i][k] != m[i-1][k] {
				diff++
			}
		}
		if diff != 1 {
			t.Fatalf("Gray step %d changed %d axes, want 1", i, diff)
		}
	}
}

func TestTourMorphInterpolates(t *testing.T) {
	walk := []Hexagram{HexagramFromNumber(0b000000), HexagramFromNumber(0b000001)} // flips axis 0
	m := TourMorph(walk, 4)
	if len(m) != 1+4 {
		t.Fatalf("expected 1 + 1*4 frames, got %d", len(m))
	}
	for _, v := range m {
		for k := 0; k < 6; k++ {
			if v[k] < 0 || v[k] > 1 {
				t.Fatalf("axis %d out of range: %v", k, v)
			}
			if k != AxFog && math.Abs(v[k]-0.25) > 1e-9 {
				t.Fatalf("only axis 0 should move; axis %d = %g", k, v[k])
			}
		}
	}
	// the changing axis slides monotonically from 0.25 to 0.75
	if math.Abs(m[0][AxFog]-0.25) > 1e-9 || math.Abs(m[len(m)-1][AxFog]-0.75) > 1e-9 {
		t.Errorf("endpoints should be 0.25 -> 0.75, got %g -> %g", m[0][AxFog], m[len(m)-1][AxFog])
	}
	for i := 1; i < len(m); i++ {
		if m[i][AxFog] <= m[i-1][AxFog] {
			t.Errorf("the changing axis should rise monotonically at step %d", i)
		}
	}
}

func TestTourMorphEmpty(t *testing.T) {
	if TourMorph(nil, 3) != nil {
		t.Error("empty walk should give no frames")
	}
}
