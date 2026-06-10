package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestMandalaCount(t *testing.T) {
	motif := []raytrace.Object{
		raytrace.Sphere{Center: raytrace.Vec3{X: 3, Y: 1}, Radius: 0.5},
		raytrace.Sphere{Center: raytrace.Vec3{X: 5, Y: 2}, Radius: 0.3},
	}
	if got := len(Mandala(motif, 6, false, raytrace.Vec3{})); got != 12 {
		t.Errorf("6-fold of 2 objects should be 12, got %d", got)
	}
	if got := len(Mandala(motif, 6, true, raytrace.Vec3{})); got != 24 {
		t.Errorf("6-fold with mirror should double to 24, got %d", got)
	}
}

// The mandala has the rotational symmetry of its group: rotating every output
// sphere by 2π/fold lands it on another output sphere (closure under the rotation).
func TestMandalaRotationalSymmetry(t *testing.T) {
	center := raytrace.Vec3{X: 1, Z: 2}
	fold := 8
	motif := []raytrace.Object{raytrace.Sphere{Center: raytrace.Vec3{X: 4, Y: 1, Z: 2}, Radius: 0.4}}
	out := Mandala(motif, fold, false, center)
	centers := make([]raytrace.Vec3, len(out))
	for i, o := range out {
		centers[i] = o.(raytrace.Sphere).Center
	}
	ang := 2 * math.Pi / float64(fold)
	for _, c := range centers {
		r := rotYAbout(c, center, ang)
		found := false
		for _, d := range centers {
			if r.Sub(d).Len() < 1e-6 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("rotating %v by one step left the symmetry set", c)
		}
	}
	// and they all sit at the motif's radius from the axis (a ring)
	for _, c := range centers {
		if d := math.Hypot(c.X-center.X, c.Z-center.Z); math.Abs(d-3) > 1e-6 {
			t.Errorf("mandala copies should share the motif radius (3), got %.3f", d)
		}
	}
}

// A reflected triangle keeps outward-facing winding (vertices B,C swapped).
func TestMandalaTriangleWinding(t *testing.T) {
	tri := raytrace.Triangle{A: raytrace.Vec3{X: 2}, B: raytrace.Vec3{X: 3, Z: 1}, C: raytrace.Vec3{X: 2, Z: 2}}
	out := Mandala([]raytrace.Object{tri}, 1, true, raytrace.Vec3{})
	if len(out) != 2 {
		t.Fatalf("1-fold with mirror should give 2, got %d", len(out))
	}
	refl := out[1].(raytrace.Triangle)
	// reflected B should come from the original C (mirrored), i.e. winding swapped
	if math.Abs(refl.B.Z-2) > 1e-9 {
		t.Errorf("reflection should swap winding (B from original C), got B=%v", refl.B)
	}
}
