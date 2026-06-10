package raydir

import (
	"math"

	"github.com/svend4/infon/internal/vision"
	"github.com/svend4/infon/pkg/brain"
)

// detect.go closes the last link of the "semantic video" chain. A vision model
// returns labeled detections (vision.Detection — normalized boxes), and
// SceneFromDetections maps them to a rayscene: a person becomes a standing figure,
// a tree a tree, a car a low box, and so on. Where SceneFromImage guesses a mood
// from colour bands, this places *recognised things*. The wire already carries a
// SceneSpec in about one UDP packet, so camera -> detections -> rayscene ->
// re-render transmits the meaning of a scene, not its pixels. The label table
// covers the common COCO classes; an unknown label falls back to a neutral box.

type detProfile struct {
	shape string     // tree | sphere | boxTall | boxLow | box
	color [3]float64 // base colour
	metal float64
}

func profileFor(label string) detProfile {
	warm := [3]float64{0.85, 0.6, 0.5}
	green := [3]float64{0.3, 0.6, 0.3}
	steel := [3]float64{0.42, 0.46, 0.55}
	fur := [3]float64{0.7, 0.58, 0.42}
	switch label {
	case "person":
		return detProfile{"boxTall", warm, 0}
	case "tree", "potted plant", "plant":
		return detProfile{"tree", green, 0}
	case "car", "truck", "bus", "train":
		return detProfile{"boxLow", steel, 0.5}
	case "motorcycle", "bicycle", "boat":
		return detProfile{"boxLow", [3]float64{0.5, 0.42, 0.4}, 0.3}
	case "bird", "cat", "dog", "sheep", "cow", "horse", "bear", "elephant", "zebra", "giraffe":
		return detProfile{"sphere", fur, 0}
	case "chair", "couch", "bed", "dining table", "bench", "toilet":
		return detProfile{"boxLow", [3]float64{0.6, 0.5, 0.4}, 0}
	case "tv", "laptop", "keyboard", "microwave", "oven", "refrigerator", "book", "clock":
		return detProfile{"boxLow", [3]float64{0.4, 0.42, 0.48}, 0.3}
	case "bottle", "wine glass", "cup", "vase", "cell phone", "remote":
		return detProfile{"sphere", [3]float64{0.5, 0.7, 0.8}, 0}
	case "bowl", "sports ball", "apple", "orange", "frisbee":
		return detProfile{"sphere", [3]float64{0.85, 0.5, 0.3}, 0}
	case "traffic light", "fire hydrant", "stop sign", "parking meter":
		return detProfile{"boxTall", [3]float64{0.85, 0.5, 0.3}, 0.2}
	default:
		return detProfile{"box", [3]float64{0.6, 0.6, 0.62}, 0.1}
	}
}

// objForDetection places one detection in the world: its box centre maps to X
// (left/right) and Z (objects lower in the frame are nearer), its width to size,
// its label to a shape.
func objForDetection(d vision.Detection, spread float64) brain.ObjSpec {
	cx := d.X + d.W/2
	by := clamp01d(d.Y + d.H)
	// The camera is right-handed (+world-X renders on the image's left), so a
	// detection on the left of the frame maps to a larger world X.
	wx := (0.5 - cx) * spread
	z := 3 + (1-by)*9 // bottom of frame (by~1) -> near; top -> far
	s := clampfd(d.W*spread, 0.6, 3.0)
	p := profileFor(d.Label)
	o := brain.ObjSpec{X: wx, Z: z, Color: p.color, Metal: p.metal, Rough: 0.5}
	switch p.shape {
	case "tree":
		o.Kind, o.R = "tree", s*0.9
	case "sphere":
		o.Kind = "sphere"
		o.R = s * 0.45
		o.Y = o.R
	case "boxTall": // an upright figure
		o.Kind = "box"
		hy := s * 0.9
		o.S, o.Y = [3]float64{s * 0.28, hy, s * 0.22}, hy
	case "boxLow": // a low, wide object (vehicle, furniture)
		o.Kind = "box"
		hy := s * 0.4
		o.S, o.Y = [3]float64{s * 0.7, hy, s * 0.5}, hy
	default: // a cube
		o.Kind = "box"
		hy := s * 0.5
		o.S, o.Y = [3]float64{hy, hy, hy}, hy
	}
	return o
}

// SceneFromDetections composes a rayscene from a vision model's detections.
func SceneFromDetections(dets []vision.Detection) brain.SceneSpec {
	spec := brain.SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: [3]float64{0.45, 0.6, 0.85}, SkyBot: [3]float64{0.85, 0.88, 0.95},
		Objects: []brain.ObjSpec{{Kind: "plane", Color: [3]float64{0.5, 0.52, 0.5}}},
	}
	const spread = 12.0
	for _, d := range dets {
		if d.Confidence > 0 && d.Confidence < 0.25 {
			continue // skip weak detections (0 = unscored, kept)
		}
		spec.Objects = append(spec.Objects, objForDetection(d, spread))
	}
	spec.Objects = append(spec.Objects, brain.ObjSpec{X: 4, Y: 7, Z: 1, R: 0.8, Emit: [3]float64{16, 16, 15}})
	if !hasRenderable(spec) { // never an empty world
		spec.Objects = append(spec.Objects, brain.ObjSpec{Y: 1, Z: 5, R: 1, Color: [3]float64{0.7, 0.7, 0.72}})
	}
	spec.Name = "Detected scene"
	return spec
}

func clamp01d(v float64) float64 { return clampfd(v, 0, 1) }
func clampfd(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
