package terminal

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

// controlGlyphs are glyph values a decoded network frame can carry that a
// terminal would act on instead of printing: C0 controls, DEL, C1 controls and
// values that are not Unicode scalar values.
var controlGlyphs = []rune{
	0x00, 0x07, '\b', '\t', '\n', '\r', 0x1b, 0x1f,
	0x7f, 0x80, 0x90, 0x9b, 0x9d, 0x9f,
	-1, 0xD800, 0x110000,
}

func oneCell(glyph rune) *Frame {
	f := NewFrame(1, 1)
	f.SetBlock(0, 0, glyph, color.NewRGB(255, 255, 255), color.NewRGB(0, 0, 0))
	return f
}

func TestRenderNeutralisesControlGlyphs(t *testing.T) {
	want := oneCell(' ').Render()
	for _, r := range controlGlyphs {
		if got := oneCell(r).Render(); got != want {
			t.Errorf("Render with glyph %U = %q, want it rendered as a space %q", r, got, want)
		}
	}
}

func TestDiffRenderNeutralisesControlGlyphs(t *testing.T) {
	for _, r := range controlGlyphs {
		ref := NewDiffRenderer()
		ref.Render(oneCell('A'))
		want := ref.Render(oneCell(' '))

		d := NewDiffRenderer()
		d.Render(oneCell('A'))
		if got := d.Render(oneCell(r)); got != want {
			t.Errorf("DiffRenderer with glyph %U = %q, want it rendered as a space %q", r, got, want)
		}
	}
}

func TestRenderKeepsPrintableGlyphs(t *testing.T) {
	for _, r := range []rune{' ', 'A', '~', 0xA0, 'é', '█', '▀', '┌', 0x2800, 0x1FB00} {
		if out := oneCell(r).Render(); !strings.Contains(out, string(r)) {
			t.Errorf("Render dropped printable glyph %U: %q", r, out)
		}
		d := NewDiffRenderer()
		d.Render(oneCell('Z'))
		if out := d.Render(oneCell(r)); !strings.Contains(out, string(r)) {
			t.Errorf("DiffRenderer dropped printable glyph %U: %q", r, out)
		}
	}
}
