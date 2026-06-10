package raydir

import (
	"image"
	"image/color"
	"testing"
)

// A painted landscape (blue sky, green ground, a bright sun top-right, a red blob)
// becomes a scene whose sky is blue, ground green, with a sun and a red object.
func TestSceneFromImage(t *testing.T) {
	const W, H = 120, 80
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			var c color.RGBA
			switch {
			case y < H/2:
				c = color.RGBA{60, 110, 230, 255} // blue sky
			default:
				c = color.RGBA{60, 170, 70, 255} // green ground
			}
			img.Set(x, y, c)
		}
	}
	// a bright sun, top-right
	for y := 5; y < 20; y++ {
		for x := W - 22; x < W-6; x++ {
			img.Set(x, y, color.RGBA{255, 250, 220, 255})
		}
	}
	// a red object, lower-middle-left
	for y := H/2 + 8; y < H/2+26; y++ {
		for x := 18; x < 40; x++ {
			img.Set(x, y, color.RGBA{220, 40, 40, 255})
		}
	}

	spec := SceneFromImage(img)
	if spec.SkyTop[2] <= spec.SkyTop[0] {
		t.Errorf("sky should be blue (B>R), got %v", spec.SkyTop)
	}
	// the floor plane should be green-dominant
	var plane *[3]float64
	for i := range spec.Objects {
		if spec.Objects[i].Kind == "plane" {
			plane = &spec.Objects[i].Color
		}
	}
	if plane == nil || plane[1] <= plane[0] || plane[1] <= plane[2] {
		t.Errorf("ground should be green-dominant, got %v", plane)
	}
	hasSun, hasRed := false, false
	for _, o := range spec.Objects {
		if o.Emit != [3]float64{} {
			hasSun = true
		}
		if o.Color[0] > 0.5 && o.Color[0] > o.Color[1]*1.5 {
			hasRed = true
		}
	}
	if !hasSun {
		t.Error("the bright patch should become a sun (emissive object)")
	}
	if !hasRed {
		t.Error("the red blob should become a red object")
	}
	if !hasRenderable(spec) {
		t.Error("the derived scene should be renderable")
	}
}

// A degenerate image still yields a valid, renderable scene.
func TestSceneFromImageTiny(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if !hasRenderable(SceneFromImage(img)) {
		t.Error("even a 1x1 image should produce a renderable fallback scene")
	}
}
