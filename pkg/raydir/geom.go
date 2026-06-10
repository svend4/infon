package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// geom.go builds the scene graph's named primitives as triangles, so a brain can
// place recognizable objects (boxes, pyramids, cylinders, trees, houses) by name
// and have them reconstructed locally from ~nothing — meaning, not geometry.

func tri(a, b, c raytrace.Vec3, m raytrace.Material) raytrace.Object {
	return raytrace.Triangle{A: a, B: b, C: c, Mat: m}
}

// boxObjects builds an axis-aligned cuboid (12 triangles) centred at c.
func boxObjects(c, half raytrace.Vec3, mat raytrace.Material) []raytrace.Object {
	mk := func(sx, sy, sz float64) raytrace.Vec3 {
		return raytrace.Vec3{X: c.X + sx*half.X, Y: c.Y + sy*half.Y, Z: c.Z + sz*half.Z}
	}
	p := [8]raytrace.Vec3{
		mk(-1, -1, -1), mk(1, -1, -1), mk(1, 1, -1), mk(-1, 1, -1),
		mk(-1, -1, 1), mk(1, -1, 1), mk(1, 1, 1), mk(-1, 1, 1),
	}
	quad := func(a, b, cc, d int) []raytrace.Object {
		return []raytrace.Object{tri(p[a], p[b], p[cc], mat), tri(p[a], p[cc], p[d], mat)}
	}
	var t []raytrace.Object
	t = append(t, quad(0, 1, 2, 3)...)
	t = append(t, quad(4, 5, 6, 7)...)
	t = append(t, quad(0, 1, 5, 4)...)
	t = append(t, quad(3, 2, 6, 7)...)
	t = append(t, quad(1, 2, 6, 5)...)
	t = append(t, quad(0, 3, 7, 4)...)
	return t
}

// pyramidObjects builds a square-based pyramid: base centred at c with half-extents
// half, apex at c.Y+half.Y, base at c.Y-half.Y (6 triangles).
func pyramidObjects(c, half raytrace.Vec3, mat raytrace.Material) []raytrace.Object {
	b0 := raytrace.Vec3{X: c.X - half.X, Y: c.Y - half.Y, Z: c.Z - half.Z}
	b1 := raytrace.Vec3{X: c.X + half.X, Y: c.Y - half.Y, Z: c.Z - half.Z}
	b2 := raytrace.Vec3{X: c.X + half.X, Y: c.Y - half.Y, Z: c.Z + half.Z}
	b3 := raytrace.Vec3{X: c.X - half.X, Y: c.Y - half.Y, Z: c.Z + half.Z}
	ap := raytrace.Vec3{X: c.X, Y: c.Y + half.Y, Z: c.Z}
	return []raytrace.Object{
		tri(b0, b1, b2, mat), tri(b0, b2, b3, mat),
		tri(b0, b1, ap, mat), tri(b1, b2, ap, mat), tri(b2, b3, ap, mat), tri(b3, b0, ap, mat),
	}
}

// ring returns seg points around a circle of radius r at height y, centred on (cx,cz).
func ring(cx, cz, y, r float64, seg int) []raytrace.Vec3 {
	pts := make([]raytrace.Vec3, seg)
	for i := 0; i < seg; i++ {
		a := float64(i) / float64(seg) * 2 * math.Pi
		pts[i] = raytrace.Vec3{X: cx + math.Cos(a)*r, Y: y, Z: cz + math.Sin(a)*r}
	}
	return pts
}

// cylinderObjects builds a capped cylinder of radius r and half-height hh centred
// at c (seg segments).
func cylinderObjects(c raytrace.Vec3, r, hh float64, mat raytrace.Material, seg int) []raytrace.Object {
	top, bot := c.Y+hh, c.Y-hh
	rb := ring(c.X, c.Z, bot, r, seg)
	rt := ring(c.X, c.Z, top, r, seg)
	cb := raytrace.Vec3{X: c.X, Y: bot, Z: c.Z}
	ct := raytrace.Vec3{X: c.X, Y: top, Z: c.Z}
	var t []raytrace.Object
	for i := 0; i < seg; i++ {
		j := (i + 1) % seg
		t = append(t, tri(rb[i], rb[j], rt[j], mat), tri(rb[i], rt[j], rt[i], mat)) // side
		t = append(t, tri(cb, rb[j], rb[i], mat))                                   // bottom
		t = append(t, tri(ct, rt[i], rt[j], mat))                                   // top
	}
	return t
}

// coneObjects builds a cone of base radius r and height h with its base at `base`.
func coneObjects(base raytrace.Vec3, r, h float64, mat raytrace.Material, seg int) []raytrace.Object {
	rb := ring(base.X, base.Z, base.Y, r, seg)
	cb := base
	ap := raytrace.Vec3{X: base.X, Y: base.Y + h, Z: base.Z}
	var t []raytrace.Object
	for i := 0; i < seg; i++ {
		j := (i + 1) % seg
		t = append(t, tri(rb[i], rb[j], ap, mat)) // side
		t = append(t, tri(cb, rb[j], rb[i], mat)) // base
	}
	return t
}

// treeObjects builds a tree: a brown trunk cylinder and a foliage cone (its colour
// is the material's colour, or green by default).
func treeObjects(c raytrace.Vec3, scale float64, mat raytrace.Material) []raytrace.Object {
	if scale <= 0 {
		scale = 1
	}
	trunk := raytrace.Material{Color: raytrace.Vec3{X: 0.4, Y: 0.26, Z: 0.13}}
	foliage := mat
	if foliage.Color.LenSq() == 0 {
		foliage.Color = raytrace.Vec3{X: 0.16, Y: 0.5, Z: 0.22}
	}
	out := cylinderObjects(raytrace.Vec3{X: c.X, Y: c.Y + 0.4*scale, Z: c.Z}, 0.14*scale, 0.4*scale, trunk, 10)
	return append(out, coneObjects(raytrace.Vec3{X: c.X, Y: c.Y + 0.7*scale, Z: c.Z}, 0.7*scale, 1.5*scale, foliage, 14)...)
}

// crystalTris builds a faceted gem (a stretched octahedron, 8 triangles) at unit
// scale around c — registered as the "crystal" instanceable mesh.
func crystalTris(c raytrace.Vec3, s float64) []raytrace.Object {
	mat := raytrace.Material{Color: raytrace.Vec3{X: 0.4, Y: 0.7, Z: 0.95}, Spec: 0.6, Shine: 80}
	px := raytrace.Vec3{X: c.X + s, Y: c.Y, Z: c.Z}
	nx := raytrace.Vec3{X: c.X - s, Y: c.Y, Z: c.Z}
	pz := raytrace.Vec3{X: c.X, Y: c.Y, Z: c.Z + s}
	nz := raytrace.Vec3{X: c.X, Y: c.Y, Z: c.Z - s}
	py := raytrace.Vec3{X: c.X, Y: c.Y + s*1.5, Z: c.Z}
	ny := raytrace.Vec3{X: c.X, Y: c.Y - s*1.5, Z: c.Z}
	return []raytrace.Object{
		tri(py, px, pz, mat), tri(py, pz, nx, mat), tri(py, nx, nz, mat), tri(py, nz, px, mat),
		tri(ny, pz, px, mat), tri(ny, nx, pz, mat), tri(ny, nz, nx, mat), tri(ny, px, nz, mat),
	}
}

// rockTris builds a low-poly rock (an irregular, flattened octahedron) at unit
// scale around c — registered as the "rock" instanceable mesh.
func rockTris(c raytrace.Vec3, s float64) []raytrace.Object {
	mat := raytrace.Material{Color: raytrace.Vec3{X: 0.42, Y: 0.4, Z: 0.38}, Rough: 0.7}
	px := raytrace.Vec3{X: c.X + s*1.1, Y: c.Y + 0.1*s, Z: c.Z}
	nx := raytrace.Vec3{X: c.X - s*0.9, Y: c.Y, Z: c.Z + 0.1*s}
	pz := raytrace.Vec3{X: c.X + 0.1*s, Y: c.Y, Z: c.Z + s}
	nz := raytrace.Vec3{X: c.X, Y: c.Y, Z: c.Z - s*1.1}
	py := raytrace.Vec3{X: c.X + 0.1*s, Y: c.Y + s*0.7, Z: c.Z - 0.1*s}
	ny := raytrace.Vec3{X: c.X, Y: c.Y - s*0.4, Z: c.Z}
	return []raytrace.Object{
		tri(py, px, pz, mat), tri(py, pz, nx, mat), tri(py, nx, nz, mat), tri(py, nz, px, mat),
		tri(ny, pz, px, mat), tri(ny, nx, pz, mat), tri(ny, nz, nx, mat), tri(ny, px, nz, mat),
	}
}

// houseObjects builds a house: a box body (the material's colour, or tan) under a
// pyramid roof.
func houseObjects(c raytrace.Vec3, scale float64, mat raytrace.Material) []raytrace.Object {
	if scale <= 0 {
		scale = 1
	}
	walls := mat
	if walls.Color.LenSq() == 0 {
		walls.Color = raytrace.Vec3{X: 0.82, Y: 0.74, Z: 0.6}
	}
	roof := raytrace.Material{Color: raytrace.Vec3{X: 0.5, Y: 0.18, Z: 0.15}}
	body := boxObjects(raytrace.Vec3{X: c.X, Y: c.Y + 0.5*scale, Z: c.Z}, raytrace.Vec3{X: 0.8 * scale, Y: 0.5 * scale, Z: 0.8 * scale}, walls)
	return append(body, pyramidObjects(raytrace.Vec3{X: c.X, Y: c.Y + 1.35*scale, Z: c.Z}, raytrace.Vec3{X: 0.95 * scale, Y: 0.35 * scale, Z: 0.95 * scale}, roof)...)
}
