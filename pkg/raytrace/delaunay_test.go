package raytrace

import (
	"math"
	"math/rand"
	"testing"
)

// A unit square triangulates into exactly two triangles.
func TestDelaunaySquare(t *testing.T) {
	tris := Delaunay([][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}})
	if len(tris) != 2 {
		t.Fatalf("a square should give 2 triangles, got %d", len(tris))
	}
	for _, tr := range tris {
		for _, i := range tr {
			if i < 0 || i > 3 {
				t.Errorf("triangle index out of range: %v", tr)
			}
		}
	}
}

// The Delaunay empty-circumcircle property: no input point lies strictly inside the
// circumcircle of any triangle. Also: every point is used.
func TestDelaunayProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const n = 40
	pts := make([][2]float64, n)
	for i := range pts {
		pts[i] = [2]float64{rng.Float64() * 100, rng.Float64() * 100}
	}
	tris := Delaunay(pts)
	if len(tris) < n-2 {
		t.Fatalf("too few triangles: %d", len(tris))
	}
	used := map[int]bool{}
	// a tolerant in-circle test (clear interior only), so points essentially on the
	// circumcircle don't count as violations.
	strictInside := func(a, b, c, d [2]float64) bool {
		ax, ay := a[0]-d[0], a[1]-d[1]
		bx, by := b[0]-d[0], b[1]-d[1]
		cx, cy := c[0]-d[0], c[1]-d[1]
		det := (ax*ax+ay*ay)*(bx*cy-cx*by) - (bx*bx+by*by)*(ax*cy-cx*ay) + (cx*cx+cy*cy)*(ax*by-bx*ay)
		return det > 1e-4
	}
	for _, tr := range tris {
		a, b, c := pts[tr[0]], pts[tr[1]], pts[tr[2]]
		used[tr[0]], used[tr[1]], used[tr[2]] = true, true, true
		for i, p := range pts {
			if i == tr[0] || i == tr[1] || i == tr[2] {
				continue
			}
			if strictInside(a, b, c, p) {
				t.Fatalf("Delaunay violated: point %d inside circumcircle of %v", i, tr)
			}
		}
	}
	if len(used) != n {
		t.Errorf("every point should be triangulated, used %d of %d", len(used), n)
	}
}

// DelaunayMesh produces valid, shaded triangles at the points' heights.
func TestDelaunayMesh(t *testing.T) {
	pts := []Vec3{{X: 0, Y: 0, Z: 0}, {X: 4, Y: 2, Z: 0}, {X: 4, Y: 1, Z: 4}, {X: 0, Y: 0, Z: 4}}
	objs := DelaunayMesh(pts, func(p Vec3) Vec3 { return Vec3{X: 0.3, Y: 0.6, Z: 0.3} })
	if len(objs) != 2 {
		t.Fatalf("expected 2 mesh triangles, got %d", len(objs))
	}
	for _, o := range objs {
		if _, ok := o.(Triangle); !ok {
			t.Fatalf("mesh should be triangles, got %T", o)
		}
	}
}

// ScatterTerrain makes an organic landscape: many triangles, heights in [0,amp].
func TestScatterTerrain(t *testing.T) {
	objs := ScatterTerrain(100, 120, 40, 3)
	if len(objs) < 50 {
		t.Fatalf("expected a meshed terrain, got %d triangles", len(objs))
	}
	for _, o := range objs {
		tr := o.(Triangle)
		for _, v := range []Vec3{tr.A, tr.B, tr.C} {
			if math.IsNaN(v.Y) || v.Y < -1e-6 || v.Y > 41 {
				t.Fatalf("terrain height out of range: %.2f", v.Y)
			}
		}
	}
}
