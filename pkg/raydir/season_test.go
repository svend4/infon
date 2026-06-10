package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// The seasons cycle in order as you walk forward.
func TestSeasonAtCycles(t *testing.T) {
	cases := []struct {
		z    float64
		name string
	}{
		{5, "spring"}, {seasonLen + 5, "summer"}, {2*seasonLen + 5, "autumn"}, {3*seasonLen + 5, "winter"},
		{4*seasonLen + 5, "spring"}, // wrapped back around
	}
	for _, c := range cases {
		if got := SeasonAt(c.z).Name; got != c.name {
			t.Errorf("SeasonAt(%.0f) = %q, want %q", c.z, got, c.name)
		}
	}
}

// The cycle is periodic over the full year.
func TestSeasonPeriodic(t *testing.T) {
	year := 4 * seasonLen
	for _, z := range []float64{0, 17, 33, 58, 95, 150} {
		a, b := SeasonAt(z), SeasonAt(z+year)
		if a.Foliage.Sub(b.Foliage).Len() > 1e-9 || math.Abs(a.Snow-b.Snow) > 1e-9 {
			t.Errorf("season at z=%.0f and z+year differ: %+v vs %+v", z, a, b)
		}
	}
}

// Each season has the palette character you'd expect.
func TestSeasonPalettes(t *testing.T) {
	summer := SeasonAt(seasonLen + 5).Foliage
	if !(summer.Y > summer.X && summer.Y > summer.Z) {
		t.Errorf("summer foliage should be green-dominant, got %+v", summer)
	}
	autumn := SeasonAt(2*seasonLen + 5).Foliage
	if !(autumn.X > autumn.Y && autumn.Y > autumn.Z) {
		t.Errorf("autumn foliage should be warm (R>G>B), got %+v", autumn)
	}
	winter := SeasonAt(3*seasonLen + 5)
	if winter.Snow < 0.8 {
		t.Errorf("winter should lie deep in snow, got Snow=%.2f", winter.Snow)
	}
	lum := func(v raytrace.Vec3) float64 { return v.X + v.Y + v.Z }
	if lum(winter.Ground) < lum(SeasonAt(seasonLen+5).Ground) {
		t.Error("winter ground (snow) should be brighter than summer ground")
	}
}

// The cross-fade is continuous: a small step in z is a small step in palette.
func TestSeasonBlendSmooth(t *testing.T) {
	maxJump := 0.0
	for z := 0.0; z < 4*seasonLen; z += 1 {
		d := SeasonAt(z).Foliage.Sub(SeasonAt(z + 1).Foliage).Len()
		if d > maxJump {
			maxJump = d
		}
	}
	if maxJump > 0.1 {
		t.Errorf("season palette should change smoothly, biggest 1-unit jump was %.3f", maxJump)
	}
}

// seasonTintSpec recolours trees and leaves other kinds alone.
func TestSeasonTintSpec(t *testing.T) {
	tree := brain.ObjSpec{Kind: "tree", Color: [3]float64{0.1, 0.5, 0.1}}
	tinted := seasonTintSpec(tree, 2*seasonLen+5) // autumn
	want := SeasonAt(2*seasonLen + 5).Foliage
	if tinted.Color != [3]float64{want.X, want.Y, want.Z} {
		t.Errorf("autumn tree should take autumn foliage, got %v", tinted.Color)
	}
	box := brain.ObjSpec{Kind: "box", Color: [3]float64{0.2, 0.2, 0.2}}
	if seasonTintSpec(box, 2*seasonLen+5).Color != box.Color {
		t.Error("non-tree specs should be left unchanged")
	}
}

// seasonFloor whitens the ground under lying snow.
func TestSeasonFloorSnow(t *testing.T) {
	base := raytrace.Plane{C1: raytrace.Vec3{X: 0.5, Y: 0.5, Z: 0.5}, C2: raytrace.Vec3{X: 0.3, Y: 0.3, Z: 0.3}}
	winter := seasonFloor(base, SeasonAt(3*seasonLen+5))
	lum := func(v raytrace.Vec3) float64 { return v.X + v.Y + v.Z }
	if lum(winter.C1) <= lum(base.C1) {
		t.Errorf("snowy floor should be brighter: base %.2f winter %.2f", lum(base.C1), lum(winter.C1))
	}
}

// A seasonal world reports its frontier season and tints the floor in the scene.
func TestWorldSeasonal(t *testing.T) {
	w := NewWorld()
	w.SetSeasonal(true)
	if !w.Seasonal() {
		t.Fatal("SetSeasonal(true) should turn the cycle on")
	}
	// place a region deep in winter; the frontier season should report winter.
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 3*seasonLen + 5}, Spec: brain.SceneSpec{
		Objects: []brain.ObjSpec{{Kind: "tree", X: 0, Z: 0, R: 1}},
	}})
	if w.Season().Name != "winter" {
		t.Errorf("frontier in winter should report winter, got %q", w.Season().Name)
	}
}
