package color

import "testing"

// bw quantizes to pure black or white by luminance — a 1-bit palette, the
// harshest case where dithering matters most.
func bw(c RGB) RGB {
	if c.Luminance() >= 128 {
		return White
	}
	return Black
}

func TestFieldGetSet(t *testing.T) {
	f := NewField(3, 2)
	f.Set(1, 1, Red)
	if f.At(1, 1) != Red {
		t.Errorf("At(1,1) = %v, want red", f.At(1, 1))
	}
	// Out of range is safe.
	f.Set(99, 99, Blue)
	if f.At(99, 99) != Black {
		t.Errorf("out-of-range At = %v, want black", f.At(99, 99))
	}
}

func countWhite(f *Field) int {
	n := 0
	for _, c := range f.Pix {
		if c == White {
			n++
		}
	}
	return n
}

func fillGray(f *Field, level uint8) {
	for i := range f.Pix {
		f.Pix[i] = RGB{R: level, G: level, B: level}
	}
}

func TestFloydSteinbergApproximatesMidGray(t *testing.T) {
	// A field of 50% gray quantized to black/white should come out ~half white
	// after error diffusion — that's the whole point of dithering.
	f := NewField(40, 40)
	fillGray(f, 128)
	f.FloydSteinberg(bw)

	white := countWhite(f)
	total := len(f.Pix)
	ratio := float64(white) / float64(total)
	if ratio < 0.4 || ratio > 0.6 {
		t.Errorf("dithered mid-gray = %.0f%% white, want ~50%%", ratio*100)
	}
}

func TestOrderedDitherApproximatesMidGray(t *testing.T) {
	f := NewField(40, 40)
	fillGray(f, 128)
	f.OrderedDither(bw, 255) // full-amplitude Bayer so the matrix spans 0..255

	ratio := float64(countWhite(f)) / float64(len(f.Pix))
	if ratio < 0.35 || ratio > 0.65 {
		t.Errorf("ordered-dithered mid-gray = %.0f%% white, want ~50%%", ratio*100)
	}
}

func TestDitherSolidColorsStayMostlySolid(t *testing.T) {
	// Pure black stays all black; pure white stays all white (no spurious noise).
	black := NewField(20, 20)
	fillGray(black, 0)
	black.FloydSteinberg(bw)
	if countWhite(black) != 0 {
		t.Errorf("dithered pure black produced %d white pixels", countWhite(black))
	}

	white := NewField(20, 20)
	fillGray(white, 255)
	white.FloydSteinberg(bw)
	if countWhite(white) != len(white.Pix) {
		t.Errorf("dithered pure white lost %d pixels", len(white.Pix)-countWhite(white))
	}
}

func TestOrderedDitherIsDeterministic(t *testing.T) {
	// Ordered dithering is stateless: same input -> identical output (good for
	// frame-to-frame stability in video).
	a := NewField(16, 16)
	b := NewField(16, 16)
	fillGray(a, 100)
	fillGray(b, 100)
	a.OrderedDither(bw, 128)
	b.OrderedDither(bw, 128)
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatalf("ordered dither not deterministic at %d", i)
		}
	}
}

func TestClampToByte(t *testing.T) {
	if clampToByte(-5) != 0 {
		t.Error("negative should clamp to 0")
	}
	if clampToByte(300) != 255 {
		t.Error("over 255 should clamp to 255")
	}
	if clampToByte(127.6) != 128 {
		t.Errorf("rounding: got %d, want 128", clampToByte(127.6))
	}
}
