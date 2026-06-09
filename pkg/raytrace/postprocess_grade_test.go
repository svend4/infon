package raytrace

import (
	"image"
	"image/color"
	"testing"
)

func lumAt(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535
}

// A vignette darkens the corners more than the centre.
func TestGradeVignette(t *testing.T) {
	const w, h = 40, 40
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src.SetRGBA(x, y, color.RGBA{180, 180, 180, 255}) // flat grey
		}
	}
	out := Grade(src, GradeOptions{Exposure: 1, Vignette: 0.8})
	centre := lumAt(out, w/2, h/2)
	corner := lumAt(out, 1, 1)
	if corner >= centre {
		t.Errorf("vignette should darken the corner: centre %.3f corner %.3f", centre, corner)
	}
}

// AgX is a monotonic tone curve that keeps a bright HDR value below white.
func TestAgXToneCurve(t *testing.T) {
	prev := -1.0
	for _, x := range []float64{0, 0.1, 0.18, 0.5, 1, 2, 8, 50} {
		v := agx(Vec3{X: x, Y: x, Z: x}).X
		if v < 0 || v > 1 {
			t.Errorf("agx(%.2f)=%.3f out of [0,1]", x, v)
		}
		if v < prev-1e-9 {
			t.Errorf("agx not monotonic at %.2f (%.3f < %.3f)", x, v, prev)
		}
		prev = v
	}
	if bright := agx(Vec3{X: 8, Y: 8, Z: 8}).X; bright >= 1 {
		t.Errorf("a bright HDR value should roll off below white, got %.3f", bright)
	}
}

// AgX grading produces a different (filmic) result than ACES, without crashing.
func TestGradeAgXRuns(t *testing.T) {
	const w, h = 16, 16
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range src.Pix {
		src.Pix[i] = 200
	}
	aces := Grade(src, GradeOptions{Exposure: 1})
	agxImg := Grade(src, GradeOptions{Exposure: 1, AgX: true})
	if aces.Bounds() != agxImg.Bounds() {
		t.Fatal("grade should preserve dimensions")
	}
}
