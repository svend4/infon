package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestFunnelObjects(t *testing.T) {
	objs := FunnelObjects(raytrace.Vec3{X: 2, Y: 0, Z: 5}, 0)
	if len(objs) != funnelRings*funnelPerRing {
		t.Fatalf("expected %d orbs, got %d", funnelRings*funnelPerRing, len(objs))
	}
	for _, o := range objs {
		sp, ok := o.(raytrace.Sphere)
		if !ok {
			t.Fatalf("funnel orbs should be spheres, got %T", o)
		}
		if sp.Mat.Emit.LenSq() == 0 {
			t.Error("a funnel orb should glow")
		}
		if d := math.Hypot(sp.Center.X-2, sp.Center.Z-5); d > funnelRadius+0.5 {
			t.Errorf("orb outside the funnel radius: %.2f", d)
		}
	}
}

// The funnel narrows toward the bottom: the lowest orbs hug the axis more than the
// rim orbs.
func TestFunnelNarrows(t *testing.T) {
	at := raytrace.Vec3{}
	objs := FunnelObjects(at, 0)
	rim := objs[0]                  // first ring (rim)
	point := objs[len(objs)-1]      // last ring (point)
	rr := math.Hypot(rim.(raytrace.Sphere).Center.X, rim.(raytrace.Sphere).Center.Z)
	pr := math.Hypot(point.(raytrace.Sphere).Center.X, point.(raytrace.Sphere).Center.Z)
	if pr >= rr {
		t.Errorf("the funnel should narrow downward: rim r=%.2f point r=%.2f", rr, pr)
	}
	if point.(raytrace.Sphere).Center.Y >= rim.(raytrace.Sphere).Center.Y {
		t.Error("the point should sit below the rim")
	}
}

// The funnel spins over time: the orbs are in different places at different t.
func TestFunnelAnimates(t *testing.T) {
	at := raytrace.Vec3{}
	a := FunnelObjects(at, 0)[0].(raytrace.Sphere).Center
	b := FunnelObjects(at, 1.0)[0].(raytrace.Sphere).Center
	if a.Sub(b).Len() < 1e-3 {
		t.Error("the funnel should spin: rim orb should move with time")
	}
}

// The world places a funnel (animated) and, with a link, a transit portal.
func TestWorldFunnel(t *testing.T) {
	w := NewWorld()
	base := len(w.SceneWith(nil).Objects)
	w.AddFunnel(raytrace.Vec3{Z: 8})
	if !w.HasFunnels() || !w.HasAnimated() {
		t.Fatal("a world with a funnel should report funnels and animation")
	}
	if got := len(w.SceneWith(nil).Objects); got <= base {
		t.Errorf("a funnel should add objects: base %d after %d", base, got)
	}
	// with a transit link, a portal appears too
	noLink := len(w.SceneWith(nil).Objects)
	w.AddFunnel(raytrace.Vec3{X: 10, Z: 8}, raytrace.Translate(raytrace.Vec3{Z: 40}))
	if len(w.SceneWith(nil).Objects) <= noLink+funnelRings*funnelPerRing-1 {
		t.Error("a linked funnel should add its orbs and a transit portal")
	}
}
