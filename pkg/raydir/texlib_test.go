package raydir

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// Built-in named textures resolve; unknown names don't.
func TestTextureLibrary(t *testing.T) {
	for _, name := range []string{"checker", "marble", "wood", "stone", "clouds"} {
		if TextureFor(name) == nil {
			t.Errorf("built-in texture %q should be registered", name)
		}
	}
	if TextureFor("definitely-not-a-texture") != nil {
		t.Error("unknown texture name should resolve to nil")
	}
}

// A named texture on any surface sets Material.Tex; an unknown one leaves it flat
// (the object still renders with its colour, not dropped).
func TestSurfaceTextured(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "box", Y: 1, Z: 5, S: [3]float64{1, 1, 1}, Tex: "checker"},
	}})
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	h, ok := nearestHit(s, ray)
	if !ok || h.Mat.Tex == nil {
		t.Fatalf("textured box should carry a Material.Tex (ok=%v)", ok)
	}

	flat := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "box", Y: 1, Z: 5, S: [3]float64{1, 1, 1}, Color: [3]float64{0.8, 0.2, 0.2}, Tex: "bogus"},
	}})
	h2, ok2 := nearestHit(flat, ray)
	if !ok2 || h2.Mat.Tex != nil {
		t.Error("unknown texture should leave the surface flat-coloured, not textured")
	}
}

// A textured mesh placement applies the texture via the instance material
// override (Tex counts as a material).
func TestMeshTextured(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "mesh", Name: "crystal", Y: 1, Z: 5, R: 2, Tex: "marble"},
	}})
	if _, ok := s.Objects[0].(*raytrace.Instance); !ok {
		t.Fatalf("mesh should be an Instance, got %T", s.Objects[0])
	}
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	h, ok := s.Objects[0].Intersect(ray, 1e-9, 1e9)
	if !ok || h.Mat.Tex == nil {
		t.Errorf("textured mesh should carry Material.Tex via override (ok=%v)", ok)
	}
}

// Image files in a directory register as named, UV-sampled textures.
func TestLoadTextureDir(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 1, color.RGBA{0, 255, 0, 255})
	f, err := os.Create(filepath.Join(dir, "brick.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	n, err := LoadTextureDir(dir)
	if err != nil || n != 1 {
		t.Fatalf("LoadTextureDir = %d,%v, want 1,nil", n, err)
	}
	if TextureFor("brick") == nil {
		t.Fatal("brick.png should register as 'brick'")
	}
}

func TestRefSceneTexKeyword(t *testing.T) {
	_, spec, _ := AuthorScene(brain.Local{}, "a marble statue in the plaza")
	found := false
	for _, o := range spec.Objects {
		if o.Tex == "marble" {
			found = true
		}
	}
	if !found {
		t.Error("'marble' prompt should texture a surface with marble")
	}
}

// nearestHit intersects every object and returns the closest hit (s.closest is
// unexported, so tests reconstruct it via the exported Object.Intersect).
func nearestHit(s *raytrace.Scene, r raytrace.Ray) (raytrace.Hit, bool) {
	best, ok := raytrace.Hit{T: 1e30}, false
	for _, o := range s.Objects {
		if h, hit := o.Intersect(r, 1e-9, 1e9); hit && h.T < best.T {
			best, ok = h, true
		}
	}
	return best, ok
}
