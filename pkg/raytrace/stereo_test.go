package raytrace

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// The eye pair straddles the original camera along its horizontal axis.
func TestStereoCameras(t *testing.T) {
	cam := Camera{Pos: Vec3{Y: 2}, Yaw: 0, FOV: 1}
	l, r := StereoCameras(cam, 0.4)
	mid := l.Pos.Add(r.Pos).Scale(0.5)
	if mid.Sub(cam.Pos).Len() > 1e-9 {
		t.Errorf("eye midpoint should be the original camera, got %+v", mid)
	}
	if sep := r.Pos.Sub(l.Pos).Len(); math.Abs(sep-0.4) > 1e-9 {
		t.Errorf("eye separation should be 0.4, got %.4f", sep)
	}
	if math.Abs(l.Pos.Y-2) > 1e-9 || math.Abs(r.Pos.Y-2) > 1e-9 {
		t.Error("a horizontal rig should not move the eyes vertically")
	}
	// for yaw 0 the horizontal axis is world X.
	if math.Abs(r.Pos.Z-l.Pos.Z) > 1e-9 {
		t.Error("at yaw 0 the eyes should differ only in X")
	}
}

// Anaglyph takes red from the left eye and green/blue from the right.
func TestAnaglyphChannels(t *testing.T) {
	left := image.NewRGBA(image.Rect(0, 0, 4, 4))
	right := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			left.SetRGBA(x, y, color.RGBA{R: 200, G: 10, B: 10, A: 255})  // left is red
			right.SetRGBA(x, y, color.RGBA{R: 10, G: 20, B: 180, A: 255}) // right is blue
		}
	}
	out := Anaglyph(left, right)
	r, g, b, _ := out.At(1, 1).RGBA()
	if uint8(r>>8) != 200 {
		t.Errorf("red channel should come from the left eye, got %d", r>>8)
	}
	if uint8(g>>8) != 20 || uint8(b>>8) != 180 {
		t.Errorf("green/blue should come from the right eye, got %d,%d", g>>8, b>>8)
	}
}

// comX is the brightness-weighted horizontal centre of mass of an image.
func comX(img image.Image) float64 {
	b := img.Bounds()
	var sw, swx float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			w := float64(r+g+bl)
			sw += w
			swx += w * float64(x)
		}
	}
	if sw == 0 {
		return 0
	}
	return swx / sw
}

// Parallax: a near object shifts more between the eyes than a far one.
func TestStereoParallax(t *testing.T) {
	cam := Camera{Pos: Vec3{Y: 1}, Yaw: 0, FOV: 1}
	render := func(center Vec3) (float64, float64) {
		s := &Scene{
			Objects:   []Object{Sphere{Center: center, Radius: 1, Mat: Material{Emit: Vec3{X: 5, Y: 5, Z: 5}}}},
			SkyTop:    Vec3{}, SkyBottom: Vec3{},
		}
		s.BuildBVH()
		l, r := StereoCameras(cam, 0.6)
		li := Render(s, l, 96, 96, Options{Samples: 1})
		ri := Render(s, r, 96, 96, Options{Samples: 1})
		return comX(li), comX(ri)
	}
	nl, nr := render(Vec3{Y: 1, Z: 5})  // near
	fl, fr := render(Vec3{Y: 1, Z: 40}) // far
	near := math.Abs(nr - nl)
	far := math.Abs(fr - fl)
	if near <= far {
		t.Errorf("a near object should shift more between the eyes than a far one: near %.2f far %.2f", near, far)
	}
}
