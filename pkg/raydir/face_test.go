package raydir

import "testing"

func TestDemoFace(t *testing.T) {
	f := DemoFace(0)
	if len(f) < 20 {
		t.Fatalf("a face should have a fair few keypoints, got %d", len(f))
	}
	for _, p := range f {
		if p[0] < 0 || p[0] > 1 || p[1] < 0 || p[1] > 1 {
			t.Fatalf("keypoint out of 0..1: %v", p)
		}
	}
	// seeds vary the face (mouth curve / eye height)
	same := true
	g := DemoFace(2)
	for i := range f {
		if f[i] != g[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different seeds should vary the face")
	}
}
