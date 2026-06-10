package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// symmetry.go takes the principle behind svend4/meta's hexsym — generate structure
// by acting with a symmetry group (Pólya) — and applies it to scenes: a motif
// (any set of objects) is replicated under the dihedral group D_n about a vertical
// axis (n rotations, optionally with a mirror), making a perfectly symmetric
// mandala. It complements the SDF mandala and the N/S mirror with group-built,
// arbitrary-geometry symmetry.

// rotYAbout rotates p about the vertical axis through c by angle (radians).
func rotYAbout(p, c raytrace.Vec3, angle float64) raytrace.Vec3 {
	dx, dz := p.X-c.X, p.Z-c.Z
	cs, sn := math.Cos(angle), math.Sin(angle)
	return raytrace.Vec3{X: c.X + dx*cs - dz*sn, Y: p.Y, Z: c.Z + dx*sn + dz*cs}
}

// reflectXAbout mirrors p across the plane X = c.X.
func reflectXAbout(p, c raytrace.Vec3) raytrace.Vec3 {
	return raytrace.Vec3{X: 2*c.X - p.X, Y: p.Y, Z: p.Z}
}

// applyObj returns a copy of o with every vertex/centre mapped by fn; for a
// reflection (flip) a triangle's winding is swapped so its face still points out.
func applyObj(o raytrace.Object, fn func(raytrace.Vec3) raytrace.Vec3, flip bool) raytrace.Object {
	switch t := o.(type) {
	case raytrace.Triangle:
		if flip {
			return raytrace.Triangle{A: fn(t.A), B: fn(t.C), C: fn(t.B), Mat: t.Mat}
		}
		return raytrace.Triangle{A: fn(t.A), B: fn(t.B), C: fn(t.C), Mat: t.Mat}
	case raytrace.Sphere:
		t.Center = fn(t.Center)
		return t
	}
	return o // other kinds pass through unchanged
}

// Mandala replicates a motif under the dihedral group D_fold about the vertical
// axis through center: `fold` rotations, plus their mirror images when mirror is
// set — a group-built mandala. Returns fold*len (or 2*fold*len) objects.
func Mandala(motif []raytrace.Object, fold int, mirror bool, center raytrace.Vec3) []raytrace.Object {
	if fold < 1 {
		fold = 1
	}
	out := make([]raytrace.Object, 0, len(motif)*fold*2)
	for k := 0; k < fold; k++ {
		ang := 2 * math.Pi * float64(k) / float64(fold)
		for _, o := range motif {
			out = append(out, applyObj(o, func(p raytrace.Vec3) raytrace.Vec3 { return rotYAbout(p, center, ang) }, false))
		}
	}
	if mirror {
		rotated := append([]raytrace.Object(nil), out...)
		for _, o := range rotated {
			out = append(out, applyObj(o, func(p raytrace.Vec3) raytrace.Vec3 { return reflectXAbout(p, center) }, true))
		}
	}
	return out
}
