package raydir

import (
	"testing"

	"github.com/svend4/infon/internal/vision"
)

func TestSceneFromDetectionsKinds(t *testing.T) {
	dets := []vision.Detection{
		{Label: "tree", Confidence: 0.9, X: 0.05, Y: 0.2, W: 0.18, H: 0.6},
		{Label: "person", Confidence: 0.95, X: 0.45, Y: 0.45, W: 0.1, H: 0.4},
		{Label: "car", Confidence: 0.9, X: 0.7, Y: 0.55, W: 0.28, H: 0.22},
	}
	spec := SceneFromDetections(dets)
	kinds := map[string]int{}
	for _, o := range spec.Objects {
		kinds[o.Kind]++
		for _, c := range o.Color {
			if c < 0 || c > 1 {
				t.Errorf("colour out of range in %+v", o)
			}
		}
	}
	if kinds["tree"] == 0 {
		t.Error("a tree detection should produce a tree")
	}
	if kinds["box"] < 2 {
		t.Errorf("person and car should both be boxes, got %d boxes", kinds["box"])
	}
	scene := BuildScene(spec)
	if scene == nil || len(scene.Objects) == 0 {
		t.Fatal("detected scene did not build")
	}
}

func TestDetectionTallVsLow(t *testing.T) {
	person := objForDetection(vision.Detection{Label: "person", X: 0.4, Y: 0.4, W: 0.1, H: 0.4}, 12)
	car := objForDetection(vision.Detection{Label: "car", X: 0.4, Y: 0.5, W: 0.3, H: 0.2}, 12)
	if person.S[1] <= person.S[0] {
		t.Errorf("a person should be taller than wide, S=%v", person.S)
	}
	if car.S[0] <= car.S[1] {
		t.Errorf("a car should be wider than tall, S=%v", car.S)
	}
}

func TestDetectionPlacement(t *testing.T) {
	left := objForDetection(vision.Detection{Label: "x", X: 0.0, Y: 0.5, W: 0.1, H: 0.2}, 12)
	right := objForDetection(vision.Detection{Label: "x", X: 0.9, Y: 0.5, W: 0.1, H: 0.2}, 12)
	// +world-X renders on the image's left, so a left-of-frame detection maps to a
	// larger world X than a right-of-frame one.
	if left.X <= right.X {
		t.Errorf("a left-of-frame detection should map to a larger world X (%v vs %v)", left.X, right.X)
	}
	near := objForDetection(vision.Detection{Label: "x", X: 0.4, Y: 0.85, W: 0.1, H: 0.1}, 12) // bottom of frame
	far := objForDetection(vision.Detection{Label: "x", X: 0.4, Y: 0.05, W: 0.1, H: 0.1}, 12)  // top of frame
	if near.Z >= far.Z {
		t.Errorf("a detection lower in the frame should be nearer (smaller Z): near=%.1f far=%.1f", near.Z, far.Z)
	}
}

func TestDetectionWeakSkipped(t *testing.T) {
	dets := []vision.Detection{{Label: "person", Confidence: 0.1, X: 0.4, Y: 0.4, W: 0.1, H: 0.4}}
	spec := SceneFromDetections(dets)
	// only the plane + the sun should remain (the weak detection is dropped)
	for _, o := range spec.Objects {
		if o.Kind == "box" {
			t.Error("a low-confidence detection should be skipped")
		}
	}
}
