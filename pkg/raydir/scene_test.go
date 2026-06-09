package raydir

import (
	"image"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// BuildScene must map the spec's material fields onto the path tracer's materials.
func TestBuildSceneMaterials(t *testing.T) {
	spec := SceneSpec{
		Light:  [3]float64{1, 2, 3},
		SkyTop: [3]float64{0.1, 0.2, 0.3},
		Objects: []ObjSpec{
			{Kind: "plane", Y: 0, Color: [3]float64{0.5, 0.5, 0.5}},
			{X: 1, Y: 1, Z: 1, R: 2, Glass: 1.5},
			{X: -1, Y: 1, Z: 0, R: 1, Metal: 1, Color: [3]float64{1, 0.8, 0.3}, Rough: 0.2},
			{X: 0, Y: 5, Z: 0, R: 0.5, Emit: [3]float64{10, 10, 10}},
		},
	}
	s := BuildScene(spec)
	if len(s.Objects) != 4 {
		t.Fatalf("want 4 objects, got %d", len(s.Objects))
	}
	if s.Light != (raytrace.Vec3{X: 1, Y: 2, Z: 3}) {
		t.Errorf("light = %v", s.Light)
	}
	if _, ok := s.Objects[0].(raytrace.Plane); !ok {
		t.Errorf("object 0 should be a plane, got %T", s.Objects[0])
	}
	glass, ok := s.Objects[1].(raytrace.Sphere)
	if !ok || glass.Mat.Glass != 1.5 || glass.Radius != 2 {
		t.Errorf("glass sphere not mapped: %+v", s.Objects[1])
	}
	metal := s.Objects[2].(raytrace.Sphere)
	if metal.Mat.Metal != 1 || metal.Mat.Rough != 0.2 || metal.Mat.Color.X != 1 {
		t.Errorf("metal sphere not mapped: %+v", metal.Mat)
	}
	emit := s.Objects[3].(raytrace.Sphere)
	if emit.Mat.Emit.X != 10 {
		t.Errorf("emissive sphere not mapped: %+v", emit.Mat)
	}
}

// The reference author returns a renderable scene that uses the full material set:
// a plane, a diffuse, a mirror, a glass sphere and an emissive light.
func TestAuthorSceneReference(t *testing.T) {
	s, spec, err := AuthorScene(RefSceneBrain{}, "a simple studio scene")
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Objects) < 4 {
		t.Fatalf("author should emit several objects, got %d", len(spec.Objects))
	}
	var hasGlass, hasEmit, hasPlane bool
	for _, o := range s.Objects {
		switch v := o.(type) {
		case raytrace.Plane:
			hasPlane = true
		case raytrace.Sphere:
			if v.Mat.Glass > 0 {
				hasGlass = true
			}
			if v.Mat.Emit.LenSq() > 0 {
				hasEmit = true
			}
		}
	}
	if !hasGlass || !hasEmit || !hasPlane {
		t.Errorf("authored scene missing material variety: glass=%v emit=%v plane=%v", hasGlass, hasEmit, hasPlane)
	}
}

// The author visibly responds to the prompt (a "gold" keyword turns the diffuse
// sphere into a metal).
func TestAuthorScenePromptVaries(t *testing.T) {
	_, plain, _ := AuthorScene(RefSceneBrain{}, "studio")
	_, gold, _ := AuthorScene(RefSceneBrain{}, "a gold sphere")
	if plain.Objects[1].Metal != 0 {
		t.Errorf("default sphere should be non-metal, got %.2f", plain.Objects[1].Metal)
	}
	if gold.Objects[1].Metal != 1 {
		t.Errorf("'gold' prompt should make the sphere metallic, got %.2f", gold.Objects[1].Metal)
	}
}

// End to end: a brain-authored scene actually renders to a lit image.
func TestAuthoredSceneRenders(t *testing.T) {
	s, _, err := AuthorScene(RefSceneBrain{}, "studio")
	if err != nil {
		t.Fatal(err)
	}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.5, Z: -6}, Pitch: -0.2, FOV: 1.0}
	img := raytrace.PathRender(s, cam, 64, 48, raytrace.PathOptions{Samples: 16, MaxDepth: 4, Seed: 1, NEE: true})
	if mean := meanLuminance(img); mean < 3 {
		t.Errorf("authored scene should render a lit image, mean luminance = %.2f", mean)
	}
}

func meanLuminance(im image.Image) float64 {
	b := im.Bounds()
	var s float64
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := im.At(x, y).RGBA()
			s += 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(bl>>8)
			n++
		}
	}
	return s / float64(n)
}
