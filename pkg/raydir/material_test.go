package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// objMaterial maps the subsurface and thin-film spec fields onto the material.
func TestObjMaterialSSSAndFilm(t *testing.T) {
	m := objMaterial(brain.ObjSpec{SSS: 1.35, SSSRad: 0.6, Color: [3]float64{0.8, 0.2, 0.2}})
	if m.SSS != 1.35 || m.SSSRadius != 0.6 {
		t.Errorf("SSS fields should map, got SSS=%.2f rad=%.2f", m.SSS, m.SSSRadius)
	}
	if m.SSSColor != (raytrace.Vec3{X: 0.8, Y: 0.2, Z: 0.2}) {
		t.Errorf("the base colour should tint the subsurface, got %+v", m.SSSColor)
	}
	f := objMaterial(brain.ObjSpec{Film: 320, FilmIOR: 1.33})
	if f.Film != 320 || f.FilmIOR != 1.33 {
		t.Errorf("thin-film fields should map, got film=%.0f ior=%.2f", f.Film, f.FilmIOR)
	}
}

// A spec carrying only SSS or Film counts as having a material (for mesh overrides).
func TestSpecHasMaterialMaterials(t *testing.T) {
	if !specHasMaterial(brain.ObjSpec{Kind: "mesh", SSS: 1.3}) {
		t.Error("an SSS-only spec should count as having a material")
	}
	if !specHasMaterial(brain.ObjSpec{Kind: "mesh", Film: 200}) {
		t.Error("a film-only spec should count as having a material")
	}
}

// Material fields are clamped to sane ranges.
func TestClampMaterialFields(t *testing.T) {
	o := clampObj(brain.ObjSpec{SSS: 99, SSSRad: 999, Film: 99999, FilmIOR: 99})
	if o.SSS > 3 || o.SSSRad > 20 || o.Film > 2000 || o.FilmIOR > 3 {
		t.Errorf("material fields should be clamped, got %+v", o)
	}
}

// The reference director understands material keywords.
func TestAuthorMaterialKeywords(t *testing.T) {
	hasSSS := func(prompt string) bool {
		_, spec, _ := AuthorScene(brain.Local{}, prompt)
		for _, o := range spec.Objects {
			if o.SSS > 0 {
				return true
			}
		}
		return false
	}
	hasFilm := func(prompt string) bool {
		_, spec, _ := AuthorScene(brain.Local{}, prompt)
		for _, o := range spec.Objects {
			if o.Film > 0 {
				return true
			}
		}
		return false
	}
	if !hasSSS("a wax statue glowing softly") {
		t.Error("'wax' should author a subsurface material")
	}
	if !hasFilm("an iridescent soap bubble") {
		t.Error("'soap'/'iridescent' should author a thin-film material")
	}
	if hasSSS("a plain stone wall") {
		t.Error("an ordinary prompt should not be subsurface")
	}
}

// A subsurface object actually renders (lit, non-black) through the wired pipeline.
func TestSubsurfaceRenders(t *testing.T) {
	scene := BuildScene(brain.SceneSpec{
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: [3]float64{0.6, 0.6, 0.6}},
			{X: 0, Y: 1, Z: 5, R: 1, Color: [3]float64{0.9, 0.7, 0.6}, SSS: 1.35, SSSRad: 0.7},
			{X: 2, Y: 5, Z: 3, R: 0.8, Emit: [3]float64{16, 16, 15}},
		},
	})
	scene.BuildBVH()
	cam := raytrace.Camera{Pos: raytrace.Vec3{Y: 1.2, Z: 0}, Yaw: 0, FOV: 1}
	im := raytrace.PathRender(scene, cam, 48, 48, raytrace.PathOptions{Samples: 24, MaxDepth: 5, Seed: 1, NEE: true, MIS: true, Sobol: true})
	if meanLumLin(im) < 1e-3 {
		t.Error("a lit subsurface scene should not render black")
	}
}
