package raydir

import (
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
