package vision

import (
	"context"
	"image"
)

// Detection is one structured result from a vision model: a labeled bounding box
// (normalized 0..1 coordinates, resolution-independent) plus optional keypoints
// (e.g. face landmarks). This is the *analysis* output that drives graphics
// overlays, the external-model tier of roadmap C3.
type Detection struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	// Box in normalized coords (0..1): X,Y top-left corner, W,H size.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
	// Points are optional normalized landmarks (e.g. face mesh), each {x,y} 0..1.
	Points [][2]float64 `json:"points,omitempty"`
}

// FrameAnalyzer is the seam an external vision model plugs into: give it a frame,
// get structured detections back. Implementations may be a local HTTP sidecar
// (YOLO / MediaPipe) or a cloud vision API; a pure-Go fallback can implement it
// too. Analyze must honor ctx.
type FrameAnalyzer interface {
	// Name identifies the analyzer (e.g. "yolo", "mediapipe", "cloud-vision").
	Name() string
	// Analyze returns detections for img (may be empty; nil error means "ran").
	Analyze(ctx context.Context, img image.Image) ([]Detection, error)
}

// PixelBox converts a normalized detection box to pixel coordinates within a
// width×height grid (e.g. terminal cells), clamped to bounds.
func (d Detection) PixelBox(width, height int) (x0, y0, x1, y1 int) {
	x0 = clampInt(int(d.X*float64(width)), 0, width-1)
	y0 = clampInt(int(d.Y*float64(height)), 0, height-1)
	x1 = clampInt(int((d.X+d.W)*float64(width)), 0, width-1)
	y1 = clampInt(int((d.Y+d.H)*float64(height)), 0, height-1)
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	return
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
