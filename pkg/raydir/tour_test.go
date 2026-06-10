package raydir

import (
	"image"
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// Catmull-Rom passes through its inner control points at the segment ends.
func TestCatmullRomEndpoints(t *testing.T) {
	p0 := raytrace.Vec3{X: -1}
	p1 := raytrace.Vec3{X: 0}
	p2 := raytrace.Vec3{X: 1}
	p3 := raytrace.Vec3{X: 2}
	if got := CatmullRom(p0, p1, p2, p3, 0); got.Sub(p1).Len() > 1e-9 {
		t.Errorf("t=0 should be p1, got %+v", got)
	}
	if got := CatmullRom(p0, p1, p2, p3, 1); got.Sub(p2).Len() > 1e-9 {
		t.Errorf("t=1 should be p2, got %+v", got)
	}
}

// The tour starts and ends on its waypoints and passes near each one.
func TestTourPassesWaypoints(t *testing.T) {
	pts := []raytrace.Vec3{{X: 0, Z: 0}, {X: 4, Z: 6}, {X: -2, Z: 12}, {X: 5, Z: 18}}
	tr := NewTour(pts, 1)
	if tr.PosAt(0).Sub(pts[0]).Len() > 1e-6 {
		t.Errorf("the tour should start at the first waypoint, got %+v", tr.PosAt(0))
	}
	if tr.PosAt(1).Sub(pts[len(pts)-1]).Len() > 1e-6 {
		t.Errorf("the tour should end at the last waypoint, got %+v", tr.PosAt(1))
	}
	for _, wp := range pts { // the spline passes through every control point
		best := math.Inf(1)
		for k := 0; k <= 400; k++ {
			if d := tr.PosAt(float64(k) / 400).Sub(wp).Len(); d < best {
				best = d
			}
		}
		if best > 0.2 {
			t.Errorf("the tour should pass near waypoint %+v, closest %.3f", wp, best)
		}
	}
}

// Arc-length pacing makes equal steps in u cover roughly equal distance.
func TestTourConstantSpeed(t *testing.T) {
	pts := []raytrace.Vec3{{X: 0}, {X: 3, Z: 8}, {X: -4, Z: 14}, {X: 6, Z: 26}}
	tr := NewTour(pts, 1)
	mn, mx := math.Inf(1), 0.0
	for k := 0; k < 10; k++ { // exact tenths, so the final step isn't a degenerate sliver
		u := float64(k) / 10
		d := tr.PosAt(u + 0.1).Sub(tr.PosAt(u)).Len()
		mn = math.Min(mn, d)
		mx = math.Max(mx, d)
	}
	if mx > mn*2.0 { // would be far more uneven without arc-length pacing
		t.Errorf("steps should be roughly equal length: min %.2f max %.2f", mn, mx)
	}
}

// The camera looks the way it travels.
func TestTourCameraLooksAhead(t *testing.T) {
	// a path running straight along +Z: yaw ~ 0.
	z := NewTour([]raytrace.Vec3{{Z: 0}, {Z: 10}, {Z: 20}}, 1)
	if y := z.CameraAt(0.4).Yaw; math.Abs(y) > 0.2 {
		t.Errorf("travelling +Z should look near yaw 0, got %.3f", y)
	}
	// a path running along +X: yaw ~ pi/2.
	x := NewTour([]raytrace.Vec3{{X: 0}, {X: 10}, {X: 20}}, 1)
	if y := x.CameraAt(0.4).Yaw; math.Abs(y-math.Pi/2) > 0.2 {
		t.Errorf("travelling +X should look near yaw pi/2, got %.3f", y)
	}
}

// A tour can be built from the world's landmarks (in index order, at height).
func TestTourFromLandmarks(t *testing.T) {
	marks := []Landmark{
		{Index: 2, At: raytrace.Vec3{X: 5, Z: 40}},
		{Index: 0, At: raytrace.Vec3{X: 0, Z: 8}},
		{Index: 1, At: raytrace.Vec3{X: 2, Z: 22}},
	}
	tr := TourFromLandmarks(marks, 2.0, 1)
	if len(tr.Points) != 3 {
		t.Fatalf("expected 3 waypoints, got %d", len(tr.Points))
	}
	if tr.Points[0].Z != 8 || tr.Points[2].Z != 40 {
		t.Errorf("waypoints should be ordered by landmark index, got %v", tr.Points)
	}
	if tr.Points[0].Y != 2.0 {
		t.Errorf("waypoints should sit at the given height, got Y=%.1f", tr.Points[0].Y)
	}
}

// Camera motion along the tour engages motion blur: the frame changes when the
// shutter spans a stretch of the path (vs a zero-span, static shot).
func TestTourMotionBlurEngages(t *testing.T) {
	var objs []raytrace.Object
	for i := 0; i < 6; i++ {
		objs = append(objs, raytrace.Sphere{
			Center: raytrace.Vec3{X: float64(i-3) * 1.5, Y: 1 + float64(i%3), Z: 5 + float64(i)},
			Radius: 0.5, Mat: raytrace.Material{Emit: raytrace.Vec3{X: 6, Y: 6, Z: 6}},
		})
	}
	scene := &raytrace.Scene{Objects: objs}
	scene.BuildBVH()
	tr := NewTour([]raytrace.Vec3{{Y: 2, Z: 0}, {Y: 2, Z: 8}, {Y: 2, Z: 16}}, 1) // dolly forward
	opt := raytrace.PathOptions{Samples: 24, MaxDepth: 2, Seed: 1, Sobol: true}

	static := raytrace.PathRenderMotion(scene, tr.CameraAt(0.4), tr.CameraAt(0.4), 80, 60, opt)
	moving := raytrace.PathRenderMotion(scene, tr.CameraAt(0.4), tr.CameraAt(0.55), 80, 60, opt)

	lum := func(im image.Image, x, y int) float64 {
		r, g, b, _ := im.At(x, y).RGBA()
		return (float64(r) + float64(g) + float64(b)) / 3 / 65535
	}
	diff := 0
	bb := static.Bounds()
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			if math.Abs(lum(static, x, y)-lum(moving, x, y)) > 0.05 {
				diff++
			}
		}
	}
	if diff < 50 {
		t.Errorf("camera motion over the tour should blur (change) the frame, only %d px differ", diff)
	}
}
