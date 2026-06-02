package deltastream_test

import (
	"testing"

	"github.com/svend4/infon/pkg/deltastream"
)

func build() deltastream.Anim {
	const w, n = 6, 12
	a := deltastream.Anim{Width: w}
	for f := 0; f < n; f++ {
		fr := make([]byte, w)
		for i := 0; i < w; i++ {
			fr[i] = byte((i*7 + f*3) % 251)
		}
		a.Frames = append(a.Frames, fr)
	}
	return a
}

func same(a, b deltastream.Anim) bool {
	if a.Width != b.Width || len(a.Frames) != len(b.Frames) {
		return false
	}
	for f := range a.Frames {
		for i := range a.Frames[f] {
			if a.Frames[f][i] != b.Frames[f][i] {
				return false
			}
		}
	}
	return true
}

func TestRoundTrip(t *testing.T) {
	a := build()
	b, err := deltastream.Decode(deltastream.Encode(a, 8), 8)
	if err != nil {
		t.Fatal(err)
	}
	if !same(a, b) {
		t.Error("round trip mismatch")
	}
}

func TestSurvivesBurst(t *testing.T) {
	a := build()
	const ecc = 12
	w := deltastream.Encode(a, ecc)
	c := make([]byte, len(w))
	copy(c, w)
	nb := len(w) / 255
	burst := nb * (ecc / 2)
	for i := 0; i < burst && i < len(c); i++ {
		c[len(c)/3+i] ^= 0xFF
	}
	b, err := deltastream.Decode(c, ecc)
	if err != nil {
		t.Fatalf("decode after burst: %v", err)
	}
	if !same(a, b) {
		t.Error("burst not recovered")
	}
}
