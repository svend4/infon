// scenebvh.go adds a top-level bounding-volume hierarchy over a whole scene's
// objects (spheres, triangles, meshes), so scenes with many objects traverse in
// O(log n) instead of testing every object. It is built with a surface-area
// heuristic (SAH): the split that minimises expected traversal cost. Unbounded
// objects (the infinite Plane) can't go in a box tree, so they are kept in a
// small linear "rest" list tested alongside. Opt in with Scene.BuildBVH().
package raytrace

import "sort"

// objectBounds returns the world-space AABB of a built-in object, or ok=false for
// unbounded ones (Plane).
func objectBounds(o Object) (aabb, bool) {
	switch v := o.(type) {
	case Sphere:
		r := Vec3{X: v.Radius, Y: v.Radius, Z: v.Radius}
		return aabb{min: v.Center.Sub(r), max: v.Center.Add(r)}, true
	case Triangle:
		return triBounds(v), true
	case *Mesh:
		c, r := v.Bound()
		rr := Vec3{X: r, Y: r, Z: r}
		return aabb{min: c.Sub(rr), max: c.Add(rr)}, true
	default:
		return aabb{}, false
	}
}

func (b aabb) area() float64 {
	d := b.max.Sub(b.min)
	if d.X < 0 || d.Y < 0 || d.Z < 0 {
		return 0
	}
	return 2 * (d.X*d.Y + d.Y*d.Z + d.Z*d.X)
}

type objNode struct {
	box          aabb
	left, right  int
	start, count int
}

type objBVH struct {
	nodes []objNode
	idx   []int
	boxes []aabb
	objs  []Object // bounded objects
	rest  []Object // unbounded objects (planes), tested linearly
}

// buildObjBVH partitions the scene's objects into a SAH BVH plus a linear rest.
func buildObjBVH(objects []Object) *objBVH {
	t := &objBVH{}
	for _, o := range objects {
		if box, ok := objectBounds(o); ok {
			t.objs = append(t.objs, o)
			t.boxes = append(t.boxes, box)
		} else {
			t.rest = append(t.rest, o)
		}
	}
	n := len(t.objs)
	t.idx = make([]int, n)
	for i := range t.idx {
		t.idx[i] = i
	}
	if n > 0 {
		t.build(0, n)
	}
	return t
}

func (t *objBVH) build(start, count int) int {
	box := emptyAABB()
	for i := 0; i < count; i++ {
		box = box.union(t.boxes[t.idx[start+i]])
	}
	id := len(t.nodes)
	t.nodes = append(t.nodes, objNode{box: box, left: -1, right: -1, start: start, count: count})
	if count <= 2 {
		return id
	}

	// choose the axis with the largest centroid spread.
	cb := emptyAABB()
	for i := 0; i < count; i++ {
		cb = cb.expand(t.boxes[t.idx[start+i]].centroid())
	}
	ext := cb.max.Sub(cb.min)
	ax := 0
	if ext.Y > axis(ext, ax) {
		ax = 1
	}
	if ext.Z > axis(ext, ax) {
		ax = 2
	}
	sub := t.idx[start : start+count]
	sort.Slice(sub, func(a, b int) bool {
		return axis(t.boxes[sub[a]].centroid(), ax) < axis(t.boxes[sub[b]].centroid(), ax)
	})

	// SAH sweep: prefix areas from the left, suffix areas from the right.
	left := make([]float64, count)
	lb := emptyAABB()
	for i := 0; i < count; i++ {
		lb = lb.union(t.boxes[sub[i]])
		left[i] = lb.area() * float64(i+1)
	}
	bestK, bestCost := -1, 1e300
	rb := emptyAABB()
	for i := count - 1; i >= 1; i-- {
		rb = rb.union(t.boxes[sub[i]])
		cost := left[i-1] + rb.area()*float64(count-i)
		if cost < bestCost {
			bestCost, bestK = cost, i
		}
	}
	if bestK <= 0 {
		bestK = count / 2 // degenerate: split in the middle
	}

	li := t.build(start, bestK)
	ri := t.build(start+bestK, count-bestK)
	t.nodes[id].left, t.nodes[id].right = li, ri
	t.nodes[id].start, t.nodes[id].count = 0, 0
	return id
}

func (t *objBVH) closest(r Ray, tMin, tMax float64) (Hit, bool) {
	var best Hit
	found := false
	near := tMax
	for _, o := range t.rest {
		if h, ok := o.Intersect(r, tMin, near); ok {
			best, found, near = h, true, h.T
		}
	}
	if len(t.nodes) > 0 {
		stack := make([]int, 0, 48)
		stack = append(stack, 0)
		for len(stack) > 0 {
			nd := t.nodes[stack[len(stack)-1]]
			stack = stack[:len(stack)-1]
			if !nd.box.hit(r, tMin, near) {
				continue
			}
			if nd.left < 0 {
				for i := 0; i < nd.count; i++ {
					if h, ok := t.objs[t.idx[nd.start+i]].Intersect(r, tMin, near); ok {
						best, found, near = h, true, h.T
					}
				}
				continue
			}
			stack = append(stack, nd.left, nd.right)
		}
	}
	return best, found
}

func (t *objBVH) anyHit(r Ray, tMin, tMax float64) bool {
	for _, o := range t.rest {
		if _, ok := o.Intersect(r, tMin, tMax); ok {
			return true
		}
	}
	if len(t.nodes) == 0 {
		return false
	}
	stack := make([]int, 0, 48)
	stack = append(stack, 0)
	for len(stack) > 0 {
		nd := t.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		if !nd.box.hit(r, tMin, tMax) {
			continue
		}
		if nd.left < 0 {
			for i := 0; i < nd.count; i++ {
				if _, ok := t.objs[t.idx[nd.start+i]].Intersect(r, tMin, tMax); ok {
					return true
				}
			}
			continue
		}
		stack = append(stack, nd.left, nd.right)
	}
	return false
}

// BuildBVH builds a top-level SAH BVH over the scene's current objects. Call it
// after populating Objects; closest/shadow queries then use it automatically.
func (s *Scene) BuildBVH() { s.sbvh = buildObjBVH(s.Objects) }
