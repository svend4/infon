package raydir

import (
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// mirror.go gives the world the dream map's symmetry. The hackers found the
// sleeping world is a mirror of the waking one — "the top of the map is south":
// north and south flip while east and west stay put. So a region reflected across
// the north-south axis is its dream double. Reflecting at the scene-spec level
// keeps it general; reflecting built geometry lets the whole world be doubled into
// an eerie, symmetric place.

// MirrorSpecZ reflects a scene spec north-south (Z -> -Z), leaving east-west (X)
// and height (Y) alone — the dream map's flip.
func MirrorSpecZ(spec brain.SceneSpec) brain.SceneSpec {
	out := spec
	out.Objects = make([]brain.ObjSpec, len(spec.Objects))
	for i, o := range spec.Objects {
		o.Z = -o.Z
		out.Objects[i] = o
	}
	return out
}

// mirrorObjectsZ returns reflected copies of renderable objects across the plane
// Z=zPlane. Triangles swap two vertices so the reflected face still points outward;
// spheres reflect their centre. Other object types are skipped (they stay as the
// originals already in the scene).
func mirrorObjectsZ(objs []raytrace.Object, zPlane float64) []raytrace.Object {
	rz := func(v raytrace.Vec3) raytrace.Vec3 { v.Z = 2*zPlane - v.Z; return v }
	out := make([]raytrace.Object, 0, len(objs))
	for _, o := range objs {
		switch t := o.(type) {
		case raytrace.Triangle:
			out = append(out, raytrace.Triangle{A: rz(t.A), B: rz(t.C), C: rz(t.B), Mat: t.Mat})
		case raytrace.Sphere:
			t.Center = rz(t.Center)
			out = append(out, t)
		}
	}
	return out
}

// SetMirror turns on the dream symmetry: the world's content is doubled by a
// north-south reflection across the plane Z=zPlane.
func (w *World) SetMirror(on bool, zPlane float64) { w.mirror, w.mirrorZ = on, zPlane }
