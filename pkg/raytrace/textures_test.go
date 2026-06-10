package raytrace

import (
	"image"
	"image/color"
	"testing"
)

func TestCheckerTexAlternates(t *testing.T) {
	c := CheckerTex{A: Vec3{1, 1, 1}, B: Vec3{0, 0, 0}, Scale: 1}
	if c.At(0, 0, Vec3{0.5, 0.5, 0.5}) != (Vec3{1, 1, 1}) {
		t.Error("cell (0,0,0) should be A")
	}
	if c.At(0, 0, Vec3{1.5, 0.5, 0.5}) != (Vec3{0, 0, 0}) {
		t.Error("adjacent cell should be B")
	}
}

func TestImageTexSamples(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 1, color.RGBA{B: 255, A: 255})
	tex := ImageTex{Img: img}
	if tl := tex.At(0, 0, Vec3{}); tl.X < 0.8 || tl.Z > 0.2 {
		t.Errorf("top-left should be red-ish, got %+v", tl)
	}
	if br := tex.At(1, 1, Vec3{}); br.Z < 0.8 || br.X > 0.2 {
		t.Errorf("bottom-right should be blue-ish, got %+v", br)
	}
}

func TestMaterialTexOverridesColor(t *testing.T) {
	plain := Hit{Mat: Material{Color: Vec3{0.1, 0.2, 0.3}}}
	if plain.albedo() != (Vec3{0.1, 0.2, 0.3}) {
		t.Errorf("no texture should use Color, got %+v", plain.albedo())
	}
	tex := Hit{Mat: Material{Color: Vec3{0.1, 0.2, 0.3}, Tex: SolidTex{C: Vec3{0.9, 0, 0}}}}
	if tex.albedo() != (Vec3{0.9, 0, 0}) {
		t.Errorf("texture should override Color, got %+v", tex.albedo())
	}
}

func TestSkyTexEnvironmentMap(t *testing.T) {
	// An empty scene with a solid green sky texture: any ray returns green.
	sc := &Scene{SkyTex: SolidTex{C: Vec3{0, 1, 0}}}
	got := sc.Trace(Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 0, 1}})
	if got.Y < 0.9 || got.X > 0.1 || got.Z > 0.1 {
		t.Errorf("env-map sky should be green, got %+v", got)
	}
}
