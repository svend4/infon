package raytrace

import (
	"math"
	"sort"
)

// bvh is a bounding-volume hierarchy over a mesh's triangles: a binary tree of
// axis-aligned boxes that turns the O(n) triangle scan into O(log n) for large
// models. Built once (median split on the longest centroid axis), traversed with
// an explicit stack and front-to-far box rejection.
type bvh struct {
	nodes []bvhNode
	idx   []int // triangle indices, partitioned into the leaves
}

type bvhNode struct {
	box          aabb
	left, right  int // child node indices; left < 0 marks a leaf
	start, count int // triangle range in idx (leaves only)
}

func axis(v Vec3, a int) float64 {
	switch a {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

// aabb is an axis-aligned bounding box.
type aabb struct{ min, max Vec3 }

func emptyAABB() aabb {
	inf := math.Inf(1)
	return aabb{min: Vec3{inf, inf, inf}, max: Vec3{-inf, -inf, -inf}}
}

func (b aabb) expand(p Vec3) aabb {
	return aabb{
		min: Vec3{math.Min(b.min.X, p.X), math.Min(b.min.Y, p.Y), math.Min(b.min.Z, p.Z)},
		max: Vec3{math.Max(b.max.X, p.X), math.Max(b.max.Y, p.Y), math.Max(b.max.Z, p.Z)},
	}
}

func (b aabb) union(o aabb) aabb { return b.expand(o.min).expand(o.max) }

func (b aabb) centroid() Vec3 { return b.min.Add(b.max).Scale(0.5) }

func triBounds(t Triangle) aabb { return emptyAABB().expand(t.A).expand(t.B).expand(t.C) }

// hit is the slab test: does the ray's [tMin,tMax] interval cross the box?
func (b aabb) hit(r Ray, tMin, tMax float64) bool {
	for a := 0; a < 3; a++ {
		o, d := axis(r.Origin, a), axis(r.Dir, a)
		mn, mx := axis(b.min, a), axis(b.max, a)
		if d == 0 {
			if o < mn || o > mx {
				return false
			}
			continue
		}
		inv := 1 / d
		t0, t1 := (mn-o)*inv, (mx-o)*inv
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		if t0 > tMin {
			tMin = t0
		}
		if t1 < tMax {
			tMax = t1
		}
		if tMax <= tMin {
			return false
		}
	}
	return true
}

func buildBVH(tris []Triangle) *bvh {
	n := len(tris)
	if n == 0 {
		return nil
	}
	b := &bvh{idx: make([]int, n)}
	boxes := make([]aabb, n)
	cents := make([]Vec3, n)
	for i, t := range tris {
		b.idx[i] = i
		boxes[i] = triBounds(t)
		cents[i] = boxes[i].centroid()
	}

	var build func(start, count int) int
	build = func(start, count int) int {
		box := emptyAABB()
		for i := 0; i < count; i++ {
			box = box.union(boxes[b.idx[start+i]])
		}
		id := len(b.nodes)
		b.nodes = append(b.nodes, bvhNode{box: box, left: -1, right: -1, start: start, count: count})
		if count <= 4 {
			return id // leaf
		}
		// split with the surface-area heuristic on the longest centroid axis:
		// sort by centroid, then sweep for the partition of least expected cost.
		cb := emptyAABB()
		for i := 0; i < count; i++ {
			cb = cb.expand(cents[b.idx[start+i]])
		}
		ext := cb.max.Sub(cb.min)
		ax := 0
		if ext.Y > axis(ext, ax) {
			ax = 1
		}
		if ext.Z > axis(ext, ax) {
			ax = 2
		}
		sub := b.idx[start : start+count]
		sort.Slice(sub, func(i, j int) bool {
			return axis(cents[sub[i]], ax) < axis(cents[sub[j]], ax)
		})
		left := make([]float64, count)
		lb := emptyAABB()
		for i := 0; i < count; i++ {
			lb = lb.union(boxes[sub[i]])
			left[i] = lb.area() * float64(i+1)
		}
		leftCount, bestCost := count/2, 1e300
		rb := emptyAABB()
		for i := count - 1; i >= 1; i-- {
			rb = rb.union(boxes[sub[i]])
			if cost := left[i-1] + rb.area()*float64(count-i); cost < bestCost {
				bestCost, leftCount = cost, i
			}
		}
		li := build(start, leftCount)
		ri := build(start+leftCount, count-leftCount)
		b.nodes[id].left, b.nodes[id].right = li, ri
		b.nodes[id].start, b.nodes[id].count = 0, 0 // internal node
		return id
	}
	build(0, n)
	return b
}

func (b *bvh) intersect(tris []Triangle, r Ray, tMin, tMax float64) (Hit, bool) {
	if len(b.nodes) == 0 {
		return Hit{}, false
	}
	var best Hit
	found := false
	closest := tMax
	stack := make([]int, 0, 64)
	stack = append(stack, 0)
	for len(stack) > 0 {
		ni := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		nd := b.nodes[ni]
		if !nd.box.hit(r, tMin, closest) {
			continue
		}
		if nd.left < 0 { // leaf
			for i := 0; i < nd.count; i++ {
				if h, ok := tris[b.idx[nd.start+i]].Intersect(r, tMin, closest); ok {
					best, found, closest = h, true, h.T
				}
			}
			continue
		}
		stack = append(stack, nd.left, nd.right)
	}
	return best, found
}
