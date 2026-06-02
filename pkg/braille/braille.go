// Package braille bridges the sub-cell glyph alphabet to Unicode Braille: each
// marks cell is sampled on a 2x4 dot grid (via glyphset coverage) and mapped to a
// Braille pattern (U+2800..U+28FF). One stream, two senses — a sighted reader
// sees the marks, a blind reader feels/hears the same shape (the tactility the
// vision documents called for).
package braille

import (
	"strings"

	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/pseudo"
)

// dotBit[row][col] is the Braille bit for the dot at that position in the 2x4
// cell (standard numbering: 1-2-3-7 down the left column, 4-5-6-8 down the right).
var dotBit = [4][2]byte{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// Cell maps one sub-cell glyph token to its Braille rune.
func Cell(token string) rune {
	if token == "" {
		return 0x2800
	}
	S := glyphset.Sub
	var bits byte
	for ry := 0; ry < 4; ry++ {
		for cx := 0; cx < 2; cx++ {
			sx := cx * S / 2
			sy := ry * S / 4
			if glyphset.Coverage(token, sx, sy, S) {
				bits |= dotBit[ry][cx]
			}
		}
	}
	return rune(0x2800 + int(bits))
}

// FromMarks renders a marks pseudo-image to Braille rows (one rune per cell).
func FromMarks(s pseudo.Spec) []string {
	if s.Marks == nil {
		return nil
	}
	out := make([]string, len(s.Marks.Rows))
	for y, row := range s.Marks.Rows {
		var b strings.Builder
		for _, tok := range row {
			b.WriteRune(Cell(tok))
		}
		out[y] = b.String()
	}
	return out
}
