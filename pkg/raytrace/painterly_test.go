package raytrace

import (
	"image"
	"image/color"
	"testing"
)

// solidNoisy builds a w×h grey image around `base` with deterministic ±amp noise.
func solidNoisy(w, h int, base, amp float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	seed := uint32(12345)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			seed = seed*1664525 + 1013904223 // LCG
			n := (float64(seed>>8&0xffff)/65535*2 - 1) * amp
			v := uint8(clamp01(base+n)*255 + 0.5)
			img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

// u8 converts a 0..1 value to an 8-bit channel (runtime, so non-integer constants
// are allowed).
func u8(v float64) uint8 { return uint8(v*255 + 0.5) }

func lumVariance(img image.Image) float64 {
	b := img.Bounds()
	var sum, sum2 float64
	n := 0.0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			l := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 65535
			sum += l
			sum2 += l * l
			n++
		}
	}
	mean := sum / n
	return sum2/n - mean*mean
}

func lumPx(img image.Image, x, y int) float64 {
	r, g, bl, _ := img.At(x, y).RGBA()
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 65535
}

func distinctColors(img image.Image) int {
	b := img.Bounds()
	seen := map[[3]uint8]bool{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			seen[[3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}] = true
		}
	}
	return len(seen)
}

func TestParsePainterly(t *testing.T) {
	for name, want := range map[string]PainterlyStyle{"oil": StyleOil, "ink": StyleInk, "poster": StylePoster} {
		if got, ok := ParsePainterly(name); !ok || got != want {
			t.Errorf("ParsePainterly(%q) = %v,%v", name, got, ok)
		}
	}
	if _, ok := ParsePainterly("nope"); ok {
		t.Error("unknown style should not parse")
	}
}

// Kuwahara flattens noise but keeps a hard edge.
func TestKuwaharaSmooths(t *testing.T) {
	noisy := solidNoisy(64, 64, 0.5, 0.18)
	vIn := lumVariance(noisy)
	vOut := lumVariance(Kuwahara(noisy, 3))
	if vOut >= vIn*0.6 {
		t.Errorf("Kuwahara should flatten noise: var in %.5f out %.5f", vIn, vOut)
	}

	// a clean step edge: interiors preserved, boundary stays sharp.
	step := image.NewRGBA(image.Rect(0, 0, 64, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 64; x++ {
			v := u8(0.2)
			if x >= 32 {
				v = u8(0.8)
			}
			step.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	out := Kuwahara(step, 3)
	if l := lumPx(out, 5, 8); l > 0.3 {
		t.Errorf("left interior should stay dark, got %.2f", l)
	}
	if l := lumPx(out, 58, 8); l < 0.7 {
		t.Errorf("right interior should stay bright, got %.2f", l)
	}
	jump := lumPx(out, 34, 8) - lumPx(out, 30, 8)
	if jump < 0.4 {
		t.Errorf("the edge should remain sharp, jump only %.2f", jump)
	}
}

// Quantize reduces the number of distinct colours.
func TestQuantizeReducesColors(t *testing.T) {
	grad := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			grad.SetRGBA(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 128, A: 255})
		}
	}
	before := distinctColors(grad)
	out := Quantize(grad, 4)
	after := distinctColors(out)
	if after >= before {
		t.Errorf("quantize should reduce colours: before %d after %d", before, after)
	}
	if after > 4*4*4 {
		t.Errorf("quantize to 4 levels should yield <= 64 colours, got %d", after)
	}
}

// InkEdges darkens edges and leaves flat regions alone.
func TestInkEdgesDarkensEdges(t *testing.T) {
	step := image.NewRGBA(image.Rect(0, 0, 64, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 64; x++ {
			v := u8(0.3)
			if x >= 32 {
				v = u8(0.9)
			}
			step.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	out := InkEdges(step, 0.8)
	if got, want := lumPx(out, 5, 8), lumPx(step, 5, 8); got < want-1e-3 {
		t.Errorf("flat region should be untouched: %.3f vs %.3f", got, want)
	}
	if lumPx(out, 32, 8) >= lumPx(step, 32, 8) {
		t.Errorf("the edge should be darkened: %.3f not < %.3f", lumPx(out, 32, 8), lumPx(step, 32, 8))
	}
}

// Each style preserves the image dimensions; StyleNone is identity.
func TestPainterlyDimsAndIdentity(t *testing.T) {
	img := solidNoisy(48, 32, 0.5, 0.1)
	for _, st := range []PainterlyStyle{StyleOil, StyleInk, StylePoster} {
		out := Painterly(img, st)
		if out.Bounds().Dx() != 48 || out.Bounds().Dy() != 32 {
			t.Errorf("style %v changed dimensions to %v", st, out.Bounds())
		}
	}
	if Painterly(img, StyleNone) != image.Image(img) {
		t.Error("StyleNone should pass the image through unchanged")
	}
}
