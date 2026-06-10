package raytrace

import (
	"image"
	"image/color"
	"testing"
)

// firstBrightX returns the x of the first bright pixel in a row, or -1.
func firstBrightX(img image.Image, y int) int {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		r, g, bl, _ := img.At(x, y).RGBA()
		if (float64(r)+float64(g)+float64(bl))/3/65535 > 0.4 {
			return x
		}
	}
	return -1
}

// A vertical line is sheared: its x at the top differs from its x at the bottom.
func TestPerspectiveSkew(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 80))
	for y := 0; y < 80; y++ {
		img.SetRGBA(40, y, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // a vertical line at x=40
	}
	out := PerspectiveSkew(img, 0.4)
	top := firstBrightX(out, 8)
	bot := firstBrightX(out, 72)
	if top < 0 || bot < 0 {
		t.Fatal("the line should survive the skew")
	}
	if top == bot {
		t.Errorf("the line should lean (different x top vs bottom): top %d bot %d", top, bot)
	}
}

// Tunnel vision darkens the corners while keeping the centre.
func TestTunnelVision(t *testing.T) {
	img := solidGray(80, 80, 0.9)
	out := TunnelVision(img, 0.9)
	if lumPx(out, 40, 40) < 0.6 {
		t.Errorf("the centre should stay bright, got %.2f", lumPx(out, 40, 40))
	}
	if lumPx(out, 2, 2) >= lumPx(out, 40, 40) {
		t.Errorf("the corner should be darkened into the tunnel")
	}
}

// Double vision makes one bright dot into two.
func TestDoubleVision(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for y := 28; y < 32; y++ {
		for x := 18; x < 22; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	out := DoubleVision(img, 24, 0)
	if lumPx(out, 20, 30) < 0.3 {
		t.Error("the original dot should remain")
	}
	if lumPx(out, 44, 30) < 0.2 {
		t.Error("a ghost dot should appear shifted by the offset")
	}
}

func TestDreamOpticsDims(t *testing.T) {
	img := solidGray(48, 32, 0.5)
	for _, o := range []image.Image{PerspectiveSkew(img, 0.3), TunnelVision(img, 0.5), DoubleVision(img, 5, 3)} {
		if o.Bounds().Dx() != 48 || o.Bounds().Dy() != 32 {
			t.Errorf("optics changed dimensions to %v", o.Bounds())
		}
	}
}
