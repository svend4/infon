package vision

import (
	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// EdgeOverlay composites an edge contour onto an existing terminal.Frame using
// Braille dots (2x4 sub-cells per character), so edges appear as crisp line art
// on top of the base picture. Pixels of the edge map whose magnitude exceeds
// threshold light their Braille dot; cells with any lit dots draw the overlay
// glyph in fg over the existing cell, leaving other cells untouched.
//
// This is the model-free C3 overlay: it works on any frame from any source.
func EdgeOverlay(frame *terminal.Frame, em *EdgeMap, threshold uint8, fg color.RGB) {
	if em.Width == 0 || em.Height == 0 {
		return
	}
	// Each cell maps to a 2x4 block of the edge map.
	cellW := em.Width / frame.Width
	cellH := em.Height / frame.Height
	if cellW < 2 {
		cellW = 2
	}
	if cellH < 4 {
		cellH = 4
	}

	for cy := 0; cy < frame.Height; cy++ {
		for cx := 0; cx < frame.Width; cx++ {
			// Braille dots[0..7]: left col rows 0-2 then bottom (6), right col
			// rows 0-2 then bottom (7). Sample the 2x4 region for this cell.
			var dots [8]bool
			any := false
			layout := [8][2]int{
				{0, 0}, {0, 1}, {0, 2}, {1, 0},
				{1, 1}, {1, 2}, {0, 3}, {1, 3},
			}
			for i, lr := range layout {
				// Map sub-cell (0..1, 0..3) into the edge map.
				ex := cx*cellW + lr[0]*cellW/2 + cellW/4
				ey := cy*cellH + lr[1]*cellH/4 + cellH/8
				if em.At(ex, ey) >= threshold {
					dots[i] = true
					any = true
				}
			}
			if !any {
				continue // no edge here: leave the underlying cell alone
			}
			glyph := glyphs.BraillePattern(dots)
			// Keep the existing background; draw the contour in fg.
			bg := frame.Blocks[cy][cx].Bg
			frame.SetBlock(cx, cy, glyph, fg, bg)
		}
	}
}
