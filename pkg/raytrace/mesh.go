package raytrace

// Mesh is a triangle soup. It keeps a bounding sphere for a cheap first reject;
// when built via NewMesh it also builds a BVH (see bvh.go) for fast traversal of
// large models. A zero-value Mesh (no BVH) still works via the linear path.
type Mesh struct {
	Tris   []Triangle
	center Vec3
	radius float64
	bvh    *bvh
}

// NewMesh builds a mesh, its bounding sphere and a BVH acceleration structure.
func NewMesh(tris []Triangle) *Mesh {
	m := &Mesh{Tris: tris}
	if len(tris) == 0 {
		return m
	}
	var sum Vec3
	for _, t := range tris {
		sum = sum.Add(t.A).Add(t.B).Add(t.C)
	}
	c := sum.Scale(1 / float64(3*len(tris)))
	var rad float64
	for _, t := range tris {
		for _, v := range [3]Vec3{t.A, t.B, t.C} {
			if d := v.Sub(c).Len(); d > rad {
				rad = d
			}
		}
	}
	m.center, m.radius = c, rad
	m.bvh = buildBVH(tris)
	return m
}

// Bound returns the mesh bounding sphere (centre, radius).
func (m *Mesh) Bound() (Vec3, float64) { return m.center, m.radius }

// Intersect implements Object: reject against the bounding sphere, then descend
// the BVH (or scan linearly if no BVH was built), keeping the nearest hit.
func (m *Mesh) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	if len(m.Tris) == 0 || !sphereOverlaps(r, m.center, m.radius, tMin, tMax) {
		return Hit{}, false
	}
	if m.bvh != nil {
		return m.bvh.intersect(m.Tris, r, tMin, tMax)
	}
	return m.scan(r, tMin, tMax)
}

// scan is the brute-force fallback used when no BVH is present.
func (m *Mesh) scan(r Ray, tMin, tMax float64) (Hit, bool) {
	var best Hit
	found := false
	closest := tMax
	for _, t := range m.Tris {
		if h, ok := t.Intersect(r, tMin, closest); ok {
			best, found, closest = h, true, h.T
		}
	}
	return best, found
}
