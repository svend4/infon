package scene3d_test

import (
	"testing"

	"github.com/svend4/infon/pkg/scene3d"
)

func buildAnim() scene3d.SceneAnim {
	const ni, N = 5, 8
	a := scene3d.SceneAnim{NItems: ni}
	for f := 0; f < N; f++ {
		items := make([]scene3d.Item, ni)
		for i := 0; i < ni; i++ {
			items[i] = scene3d.Item{
				Piece: byte(i), X: byte(40 + 20*i + 3*f), Y: byte(60 + 10*i),
				Z: byte(100 + 8*f), Yaw: byte((i + f) % 8), Color: byte(i),
			}
		}
		a.Frames = append(a.Frames, items)
	}
	return a
}

func same(a, b scene3d.SceneAnim) bool {
	if a.NItems != b.NItems || len(a.Frames) != len(b.Frames) {
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

func TestItemAndSceneFaces(t *testing.T) {
	it := scene3d.Item{Piece: 0, X: 100, Y: 100, Z: 100, Yaw: 1, Color: 3}
	if len(scene3d.ItemFaces(it)) < 4 {
		t.Error("item should yield a top + walls")
	}
	a := buildAnim()
	if len(scene3d.SceneFaces(a.Frames[0])) == 0 {
		t.Error("empty scene faces")
	}
	if scene3d.Render(a.Frames[0], 200, 200) == nil {
		t.Fatal("nil render")
	}
}

func TestSceneStreamRoundTrip(t *testing.T) {
	a := buildAnim()
	b, err := scene3d.DecodeSceneStream(scene3d.EncodeSceneStream(a, 10), 10)
	if err != nil {
		t.Fatal(err)
	}
	if !same(a, b) {
		t.Error("round trip mismatch")
	}
}

func TestSceneStreamSurvivesBurst(t *testing.T) {
	a := buildAnim()
	const ecc = 12
	w := scene3d.EncodeSceneStream(a, ecc)
	c := make([]byte, len(w))
	copy(c, w)
	nb := len(w) / 255
	burst := nb * (ecc / 2)
	if burst < 1 {
		burst = 1
	}
	start := len(w)/2 - burst/2
	for i := 0; i < burst && start+i < len(w); i++ {
		c[start+i] ^= 0xFF
	}
	b, err := scene3d.DecodeSceneStream(c, ecc)
	if err != nil {
		t.Fatalf("decode after %d-byte burst: %v", burst, err)
	}
	if !same(a, b) {
		t.Error("burst not recovered")
	}
}
