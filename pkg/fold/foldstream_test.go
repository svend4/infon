package fold

import (
	"math"
	"testing"

	t7 "github.com/svend4/infon/pkg/tangram7"
)

func buildAnim() FoldAnim {
	const np, N = 7, 10
	fa := FoldAnim{NPieces: np}
	for f := 0; f < N; f++ {
		fr := make([]byte, np)
		for i := 0; i < np; i++ {
			t := float64(f) / float64(N-1)
			fr[i] = byte(t * float64(120+10*i))
		}
		fa.Frames = append(fa.Frames, fr)
	}
	return fa
}

func sameFrames(a, b FoldAnim) bool {
	if a.NPieces != b.NPieces || len(a.Frames) != len(b.Frames) {
		return false
	}
	for f := range a.Frames {
		if len(a.Frames[f]) != len(b.Frames[f]) {
			return false
		}
		for i := range a.Frames[f] {
			if a.Frames[f][i] != b.Frames[f][i] {
				return false
			}
		}
	}
	return true
}

// Folding lifts the figure out of the plane: flat at angle 0, raised otherwise.
func TestFacesFoldLifts(t *testing.T) {
	sq := t7.Square()
	if z := maxZ(FacesFold(sq, make([]float64, len(sq)))); z != 0 {
		t.Errorf("angle 0 should be flat, max z=%.3f", z)
	}
	ang := make([]float64, len(sq))
	for i := range ang {
		ang[i] = math.Pi / 4
	}
	if z := maxZ(FacesFold(sq, ang)); z <= 0 {
		t.Errorf("folded should lift, max z=%.3f", z)
	}
}

// The animation survives a round trip through the RS channel.
func TestFoldStreamRoundTrip(t *testing.T) {
	a := buildAnim()
	b, err := DecodeFoldStream(EncodeFoldStream(a, 8), 8)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFrames(a, b) {
		t.Error("round trip mismatch")
	}
}

// A burst within the ecc/2 budget is corrected before deserialising.
func TestFoldStreamSurvivesBurst(t *testing.T) {
	a := buildAnim()
	const ecc = 12
	w := EncodeFoldStream(a, ecc)
	c := make([]byte, len(w))
	copy(c, w)
	start := len(w) / 3
	for i := 0; i < ecc/2; i++ {
		c[start+i] ^= 0xFF
	}
	b, err := DecodeFoldStream(c, ecc)
	if err != nil {
		t.Fatalf("decode after burst: %v", err)
	}
	if !sameFrames(a, b) {
		t.Error("burst not recovered")
	}
}
