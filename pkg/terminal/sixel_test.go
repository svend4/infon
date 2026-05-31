package terminal

import (
	"image"
	gocolor "image/color"
	"strings"
	"testing"
)

func solidImg(w, h int, c gocolor.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestSixelWrapsInDCSandST(t *testing.T) {
	out := EncodeSixel(solidImg(6, 6, gocolor.RGBA{R: 255, A: 255}), 16)
	if !strings.HasPrefix(out, "\x1bP") {
		t.Error("sixel output should start with a DCS (ESC P)")
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("sixel output should end with ST (ESC \\)")
	}
}

func TestSixelEmptyImage(t *testing.T) {
	out := EncodeSixel(image.NewRGBA(image.Rect(0, 0, 0, 0)), 16)
	if out != sixelStart+sixelEnd {
		t.Errorf("empty image should be just DCS+ST, got %q", out)
	}
}

func TestSixelDeclaresPalette(t *testing.T) {
	// A solid red image must declare at least one color register "#0;2;R;G;B".
	out := EncodeSixel(solidImg(12, 12, gocolor.RGBA{R: 255, A: 255}), 16)
	if !strings.Contains(out, "#0;2;") {
		t.Error("expected a palette declaration (#0;2;...)")
	}
	// Red in sixel scale is ;2;100;0;0.
	if !strings.Contains(out, ";2;100;0;0") {
		t.Errorf("expected pure-red palette entry, got: %q", firstColors(out))
	}
}

func TestSixelBandTerminators(t *testing.T) {
	// 12 rows -> 2 bands -> at least 2 band terminators '-'.
	out := EncodeSixel(solidImg(6, 12, gocolor.RGBA{G: 255, A: 255}), 16)
	if strings.Count(out, "-") < 2 {
		t.Errorf("expected >=2 band terminators for 12 rows, got %d", strings.Count(out, "-"))
	}
}

func TestSixelRunLengthEncoding(t *testing.T) {
	// A wide solid image should trigger RLE ("!<count>").
	out := EncodeSixel(solidImg(40, 6, gocolor.RGBA{B: 255, A: 255}), 16)
	if !strings.Contains(out, "!") {
		t.Error("a wide solid run should use run-length encoding")
	}
}

func TestQuantizeReducesColors(t *testing.T) {
	// A gradient image quantized to <=16 colors must yield a small palette.
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 100, A: 255})
		}
	}
	pal, idx := quantize(img, 16)
	if len(pal) == 0 || len(pal) > 64 {
		t.Errorf("palette size %d out of expected range", len(pal))
	}
	if len(idx) != 32*32 {
		t.Errorf("index map size = %d, want 1024", len(idx))
	}
	for _, i := range idx {
		if i < 0 || i >= len(pal) {
			t.Fatalf("pixel index %d out of palette range %d", i, len(pal))
		}
	}
}

// firstColors returns a short prefix of the sixel string for error messages.
func firstColors(s string) string {
	if len(s) > 80 {
		return s[:80]
	}
	return s
}
