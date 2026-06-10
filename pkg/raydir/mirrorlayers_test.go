package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func TestMirrorSpecZ(t *testing.T) {
	in := brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "tree", X: 3, Y: 1, Z: 7}}}
	out := MirrorSpecZ(in)
	o := out.Objects[0]
	if o.Z != -7 || o.X != 3 || o.Y != 1 {
		t.Errorf("N/S flip should negate Z only, got %+v", o)
	}
	if in.Objects[0].Z != 7 {
		t.Error("MirrorSpecZ should not mutate the input")
	}
}

func TestMirrorObjectsZ(t *testing.T) {
	objs := []raytrace.Object{
		raytrace.Sphere{Center: raytrace.Vec3{X: 1, Y: 2, Z: 10}, Radius: 1},
		raytrace.Triangle{A: raytrace.Vec3{Z: 4}, B: raytrace.Vec3{X: 1, Z: 4}, C: raytrace.Vec3{Z: 6}},
	}
	m := mirrorObjectsZ(objs, 0)
	if len(m) != 2 {
		t.Fatalf("expected 2 mirrored objects, got %d", len(m))
	}
	sp := m[0].(raytrace.Sphere)
	if sp.Center.Z != -10 || sp.Center.X != 1 {
		t.Errorf("sphere should reflect Z about 0, got %+v", sp.Center)
	}
	tr := m[1].(raytrace.Triangle)
	// winding swap: reflected B comes from original C (Z=6 -> -6)
	if tr.B.Z != -6 || tr.C.Z != -4 {
		t.Errorf("triangle winding should swap on reflection, got B=%+v C=%+v", tr.B, tr.C)
	}
}

func TestWorldMirror(t *testing.T) {
	w := NewWorld()
	w.AddDecor(raytrace.Sphere{Center: raytrace.Vec3{Y: 1, Z: 10}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 1}}})
	base := len(w.SceneWith(nil).Objects)
	w.SetMirror(true, 0)
	objs := w.SceneWith(nil).Objects
	if len(objs) <= base {
		t.Fatalf("mirroring should add reflected objects: base %d after %d", base, len(objs))
	}
	found := false
	for _, o := range objs {
		if sp, ok := o.(raytrace.Sphere); ok && math.Abs(sp.Center.Z+10) < 1e-6 {
			found = true
		}
	}
	if !found {
		t.Error("the world should contain the sphere's north-south reflection at Z=-10")
	}
}

func TestLayerSky(t *testing.T) {
	lum := func(v raytrace.Vec3) float64 { return v.X + v.Y + v.Z }
	upT, _ := LayerSky(LayerUpper)
	loT, _ := LayerSky(LayerLower)
	if lum(upT) <= lum(loT) {
		t.Errorf("the upper world should be brighter than the lower: up %.2f low %.2f", lum(upT), lum(loT))
	}
}

func TestWorldLayer(t *testing.T) {
	w := NewWorld()
	w.SetLayer(LayerLower)
	if w.Layer() != LayerLower {
		t.Error("SetLayer should record the layer")
	}
	dark := w.SkyTop.X + w.SkyTop.Y + w.SkyTop.Z
	w.SetLayer(LayerUpper)
	bright := w.SkyTop.X + w.SkyTop.Y + w.SkyTop.Z
	if bright <= dark {
		t.Errorf("the upper layer should repaint a brighter sky than the lower: up %.2f low %.2f", bright, dark)
	}
}

func TestDescentTunnel(t *testing.T) {
	at := raytrace.Vec3{X: 2, Y: 0, Z: 5}
	objs := DescentTunnel(at, 6)
	if len(objs) == 0 {
		t.Fatal("a descent tunnel should have geometry")
	}
	minY := math.Inf(1)
	for _, o := range objs {
		tr, ok := o.(raytrace.Triangle)
		if !ok {
			continue
		}
		for _, v := range []raytrace.Vec3{tr.A, tr.B, tr.C} {
			if d := math.Hypot(v.X-at.X, v.Z-at.Z); d > 3.4 {
				t.Fatalf("tunnel wall should stay within its radius, got %.2f", d)
			}
			minY = math.Min(minY, v.Y)
		}
	}
	if minY > at.Y-5 {
		t.Errorf("the tunnel should descend, deepest Y was %.2f", minY)
	}
}
