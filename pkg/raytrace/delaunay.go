// delaunay.go builds terrain the InfoAquarium way (svend4/in4n): a set of scattered
// points is Delaunay-triangulated on the ground plane (Bowyer-Watson), then lifted
// to heights and turned into a triangle mesh the path tracer renders. Unlike the
// regular voxel height field, the mesh follows an irregular point cloud — good for
// organic, low-poly landscapes or for meshing a cloud of landmarks.
package raytrace

import (
	"math"
	"math/rand"
)

// Delaunay returns the Delaunay triangulation of 2-D points as index triples into
// `pts` (Bowyer-Watson with a super-triangle). Triangles are counter-clockwise.
func Delaunay(pts [][2]float64) [][3]int {
	n := len(pts)
	if n < 3 {
		return nil
	}
	work := append([][2]float64(nil), pts...)
	minx, miny := pts[0][0], pts[0][1]
	maxx, maxy := minx, miny
	for _, p := range pts {
		minx, maxx = math.Min(minx, p[0]), math.Max(maxx, p[0])
		miny, maxy = math.Min(miny, p[1]), math.Max(maxy, p[1])
	}
	dm := math.Max(maxx-minx, maxy-miny)*10 + 1
	midx, midy := (minx+maxx)/2, (miny+maxy)/2
	work = append(work, [2]float64{midx - 2*dm, midy - dm}, [2]float64{midx, midy + 2*dm}, [2]float64{midx + 2*dm, midy - dm})
	s0, s1, s2 := n, n+1, n+2

	type tri struct{ a, b, c int }
	ccw := func(a, b, c int) tri { // orient counter-clockwise
		if (work[b][0]-work[a][0])*(work[c][1]-work[a][1])-(work[b][1]-work[a][1])*(work[c][0]-work[a][0]) < 0 {
			b, c = c, b
		}
		return tri{a, b, c}
	}
	tris := []tri{ccw(s0, s1, s2)}

	for p := 0; p < n; p++ {
		px := work[p]
		type edge struct{ u, v int }
		count := map[edge]int{}
		norm := func(u, v int) edge {
			if u > v {
				u, v = v, u
			}
			return edge{u, v}
		}
		kept := tris[:0:0]
		for _, t := range tris {
			if inCircum(work[t.a], work[t.b], work[t.c], px) {
				count[norm(t.a, t.b)]++
				count[norm(t.b, t.c)]++
				count[norm(t.c, t.a)]++
			} else {
				kept = append(kept, t)
			}
		}
		tris = kept
		for e, c := range count {
			if c == 1 { // a boundary edge of the cavity
				tris = append(tris, ccw(e.u, e.v, p))
			}
		}
	}

	var out [][3]int
	for _, t := range tris {
		if t.a < n && t.b < n && t.c < n { // drop triangles touching the super-triangle
			out = append(out, [3]int{t.a, t.b, t.c})
		}
	}
	return out
}

// inCircum reports whether d lies strictly inside the circumcircle of the
// counter-clockwise triangle a,b,c (the Delaunay in-circle predicate).
func inCircum(a, b, c, d [2]float64) bool {
	ax, ay := a[0]-d[0], a[1]-d[1]
	bx, by := b[0]-d[0], b[1]-d[1]
	cx, cy := c[0]-d[0], c[1]-d[1]
	det := (ax*ax+ay*ay)*(bx*cy-cx*by) -
		(bx*bx+by*by)*(ax*cy-cx*ay) +
		(cx*cx+cy*cy)*(ax*by-bx*ay)
	return det > 1e-12
}

// DelaunayMesh triangulates points (x,z used on the ground; y is the height) and
// returns the lifted triangle mesh, each face coloured by shade(centroid).
func DelaunayMesh(points []Vec3, shade func(Vec3) Vec3) []Object {
	pts := make([][2]float64, len(points))
	for i, p := range points {
		pts[i] = [2]float64{p.X, p.Z}
	}
	var out []Object
	for _, t := range Delaunay(pts) {
		a, b, c := points[t[0]], points[t[1]], points[t[2]]
		col := Vec3{X: 0.5, Y: 0.5, Z: 0.5}
		if shade != nil {
			col = shade(a.Add(b).Add(c).Scale(1.0 / 3))
		}
		out = append(out, Triangle{A: a, B: b, C: c, Mat: Material{Color: col, Rough: 0.6}})
	}
	return out
}

// ScatterTerrain builds an organic low-poly landscape: n jittered points over a
// square of side `extent`, lifted by fractal noise to amplitude `amp`, Delaunay-
// meshed with a height-banded palette. Deterministic in the seed.
func ScatterTerrain(n int, extent, amp float64, seed int64) []Object {
	rng := rand.New(rand.NewSource(seed))
	pts := make([]Vec3, 0, n)
	// a jittered grid covers the area without gaps or clumps.
	g := int(math.Sqrt(float64(n)))
	if g < 2 {
		g = 2
	}
	cell := extent / float64(g)
	for i := 0; i < g; i++ {
		for j := 0; j < g; j++ {
			x := -extent/2 + (float64(i)+rng.Float64())*cell
			z := -extent/2 + (float64(j)+rng.Float64())*cell
			h := 0.5 * (FBM(Vec3{X: x * 0.04, Z: z * 0.04}, 5, 2, 0.5) + 1) * amp
			pts = append(pts, Vec3{X: x, Y: h, Z: z})
		}
	}
	shade := func(p Vec3) Vec3 {
		switch t := p.Y / amp; {
		case t < 0.30:
			return Vec3{X: 0.16, Y: 0.34, Z: 0.5}
		case t < 0.5:
			return Vec3{X: 0.3, Y: 0.5, Z: 0.28}
		case t < 0.75:
			return Vec3{X: 0.45, Y: 0.4, Z: 0.34}
		default:
			return Vec3{X: 0.92, Y: 0.93, Z: 0.97}
		}
	}
	return DelaunayMesh(pts, shade)
}
