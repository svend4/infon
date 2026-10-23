package babe

import (
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// codecVideoAuditConverters are the quadrant-family converters, which all sample
// a 2x2 block of the source image per cell.
var codecVideoAuditConverters = map[string]func(image.Image, int, int) *terminal.Frame{
	"quadrant":   ImageToFrame,
	"perceptual": ImageToFramePerceptual,
	"optimal": func(img image.Image, w, h int) *terminal.Frame {
		return ImageToFrameMode(img, w, h, ModeOptimal)
	},
}

func codecVideoAuditNear(a, b color.RGB) bool {
	return absI(int(a.R)-int(b.R)) <= 2 && absI(int(a.G)-int(b.G)) <= 2 && absI(int(a.B)-int(b.B)) <= 2
}

// codecVideoAuditCheckSolid asserts that every quadrant of every cell renders as
// c: lit quadrants show fg, unlit ones bg.
func codecVideoAuditCheckSolid(t *testing.T, name string, f *terminal.Frame, c color.RGB) {
	t.Helper()
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			b := f.Blocks[y][x]
			pattern := -1
			for p := 0; p < 16; p++ {
				if glyphs.QuadrantGlyphs[p].Char == b.Glyph {
					pattern = p
					break
				}
			}
			if pattern < 0 {
				t.Errorf("%s: cell (%d,%d) glyph %q is not a quadrant glyph", name, x, y, b.Glyph)
				return
			}
			for q := 0; q < 4; q++ {
				shown := b.Bg
				if pattern&(1<<uint(3-q)) != 0 {
					shown = b.Fg
				}
				if !codecVideoAuditNear(shown, c) {
					t.Errorf("%s: cell (%d,%d) quadrant %d renders %v (glyph %q fg %v bg %v), want %v",
						name, x, y, q, shown, b.Glyph, b.Fg, b.Bg, c)
					return
				}
			}
		}
	}
}

// A cropped image (SubImage) has Bounds().Min != (0,0); the converters must
// sample inside its bounds instead of treating them as starting at the origin.
func TestCodecVideoAuditConvertSubImage(t *testing.T) {
	red := color.RGB{R: 220, G: 20, B: 20}
	blue := color.RGB{R: 20, G: 20, B: 220}
	parent := fillSolid(16, 16, blue)
	for y := 8; y < 16; y++ {
		for x := 8; x < 16; x++ {
			parent.SetRGBA(x, y, gocolor.RGBA{R: red.R, G: red.G, B: red.B, A: 255})
		}
	}
	crop := parent.SubImage(image.Rect(8, 8, 16, 16))

	for name, conv := range codecVideoAuditConverters {
		codecVideoAuditCheckSolid(t, name, conv(crop, 4, 4), red)
	}

	f := SimpleImageToFrame(crop, 4, 4)
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			if fg := f.Blocks[y][x].Fg; fg != red {
				t.Fatalf("simple: cell (%d,%d) fg = %v, want %v", x, y, fg, red)
			}
		}
	}
}

// Upscaling (more sub-pixels than source pixels) must clamp to the image edge
// rather than drop the out-of-range sub-pixels and render them black.
func TestCodecVideoAuditConvertUpscale(t *testing.T) {
	green := color.RGB{R: 30, G: 200, B: 60}
	img := fillSolid(3, 3, green)
	for name, conv := range codecVideoAuditConverters {
		codecVideoAuditCheckSolid(t, name, conv(img, 8, 8), green)
	}
}
