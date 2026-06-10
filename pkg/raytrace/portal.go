// portal.go adds a portal: a finite rectangle that, when a ray crosses it, sends
// the ray on into a linked place by an affine transform. Looking at a portal you
// see straight through to wherever it links — a far part of the world, a rotated
// copy, or (linking to just behind itself) an endless corridor. It is the
// renderer's door to non-Euclidean, Escher-like space. The teleport itself lives
// in the path tracer (Material.Link); this file is just the doorway geometry.
package raytrace

// Portal is a flat rectangular doorway. The quad spans Center ± U ± V, with the
// outward normal U×V; a ray that hits it is teleported by Link (see Material.Link).
type Portal struct {
	Center, U, V Vec3
	link         *Transform
	n            Vec3
	mat          Material
}

// NewPortal builds a portal rectangle centred at center with half-axes u and v
// (so it spans center ± u ± v), linking through the affine transform link. A ray
// that crosses it continues from link.Apply(hit) in direction link.ApplyDir(dir).
func NewPortal(center, u, v Vec3, link Transform) *Portal {
	lp := link
	return &Portal{
		Center: center, U: u, V: v,
		link: &lp,
		n:    u.Cross(v).Norm(),
		mat:  Material{Color: Vec3{X: 0.02, Y: 0.02, Z: 0.03}, Link: &lp}, // dark frame if a renderer can't teleport
	}
}

// Intersect tests the ray against the portal rectangle.
func (p *Portal) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	denom := r.Dir.Dot(p.n)
	if denom < geomEps && denom > -geomEps {
		return Hit{}, false // parallel to the portal plane
	}
	t := p.Center.Sub(r.Origin).Dot(p.n) / denom
	if t < tMin || t > tMax {
		return Hit{}, false
	}
	d := r.At(t).Sub(p.Center)
	uu := d.Dot(p.U) / p.U.LenSq() // in [-1,1] inside the rectangle
	vv := d.Dot(p.V) / p.V.LenSq()
	if uu < -1 || uu > 1 || vv < -1 || vv > 1 {
		return Hit{}, false
	}
	n, front := orient(p.n, r.Dir)
	return Hit{T: t, P: r.At(t), N: n, U: (uu + 1) / 2, V: (vv + 1) / 2, Front: front, Mat: p.mat}, true
}
