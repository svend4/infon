package raytrace

import (
	"testing"
	"time"
)

// Building a BVH over many coincident triangles (every SAH split ties) must not
// degenerate into a linear chain: the tree stays shallow and the build fast.
func TestBVHIdenticalTrianglesStaysBalanced(t *testing.T) {
	const n = 8192
	tri := Triangle{A: Vec3{X: 0, Y: 0, Z: 5}, B: Vec3{X: 1, Y: 0, Z: 5}, C: Vec3{X: 0, Y: 1, Z: 5}}
	tris := make([]Triangle, n)
	for i := range tris {
		tris[i] = tri
	}
	start := time.Now()
	b := buildBVH(tris)
	if el := time.Since(start); el > time.Second {
		t.Fatalf("building %d identical triangles took %v", n, el)
	}
	var depth func(id, d int) int
	depth = func(id, d int) int {
		nd := b.nodes[id]
		if nd.left < 0 {
			return d
		}
		return max(depth(nd.left, d+1), depth(nd.right, d+1))
	}
	if d := depth(0, 0); d > 64 {
		t.Errorf("BVH depth %d for %d triangles: build degenerated into a chain", d, n)
	}
	r := Ray{Origin: Vec3{X: 0.2, Y: 0.2, Z: 0}, Dir: Vec3{X: 0, Y: 0, Z: 1}}
	if h, ok := b.intersect(tris, r, shadowEps, tFar); !ok || h.T < 4.99 || h.T > 5.01 {
		t.Errorf("ray through the stack: hit=%v t=%v, want t=5", ok, h.T)
	}
}

// Point-sized triangles strung along a line have zero-area boxes, so every SAH
// split costs 0; the build must still split evenly.
func TestBVHZeroAreaTrianglesStaysBalanced(t *testing.T) {
	const n = 4096
	tris := make([]Triangle, n)
	for i := range tris {
		p := Vec3{X: float64(i) * 0.01, Y: 2, Z: 3}
		tris[i] = Triangle{A: p, B: p, C: p}
	}
	start := time.Now()
	b := buildBVH(tris)
	if el := time.Since(start); el > time.Second {
		t.Fatalf("building %d zero-area triangles took %v", n, el)
	}
	var depth func(id, d int) int
	depth = func(id, d int) int {
		nd := b.nodes[id]
		if nd.left < 0 {
			return d
		}
		return max(depth(nd.left, d+1), depth(nd.right, d+1))
	}
	if d := depth(0, 0); d > 64 {
		t.Errorf("BVH depth %d for %d triangles: build degenerated into a chain", d, n)
	}
}
