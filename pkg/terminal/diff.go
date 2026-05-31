package terminal

import (
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/color"
)

// MoveCursor returns the ANSI escape that moves the cursor to a 1-based
// (row, col) position.
func MoveCursor(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// HideCursor / ShowCursor toggle the terminal cursor (useful for animation).
const (
	HideCursor = "\x1b[?25l"
	ShowCursor = "\x1b[?25h"
)

// DiffRenderer emits only the cells that changed since the previous frame,
// using cursor positioning instead of clearing and redrawing the whole screen.
// This removes the flicker and most of the bytes of a full repaint, which makes
// it the right renderer for fast, interactive animation (games, live synthesis).
//
// It is not safe for concurrent use; drive it from a single render loop.
type DiffRenderer struct {
	prev *Frame
}

// NewDiffRenderer creates a renderer whose first Render is a full repaint.
func NewDiffRenderer() *DiffRenderer { return &DiffRenderer{} }

// Reset forgets the previous frame, so the next Render is a full repaint. Call
// it after anything else writes to the screen (e.g. a status line).
func (d *DiffRenderer) Reset() { d.prev = nil }

// Render returns the ANSI needed to turn the previously rendered frame into f.
// The first call (or any size change) returns a full repaint; subsequent calls
// return only the changed cell-runs. An unchanged frame returns "".
func (d *DiffRenderer) Render(f *Frame) string {
	// Full repaint on first frame or a size change.
	if d.prev == nil || d.prev.Width != f.Width || d.prev.Height != f.Height {
		d.prev = cloneFrame(f)
		return f.Render()
	}

	var buf strings.Builder
	for y := 0; y < f.Height; y++ {
		x := 0
		for x < f.Width {
			if blocksEqual(f.Blocks[y][x], d.prev.Blocks[y][x]) {
				x++
				continue
			}
			// Start of a changed run: position the cursor once, then write the
			// run, tracking colors so we only emit escapes on change.
			buf.WriteString(MoveCursor(y+1, x+1))
			var lastFg, lastBg color.RGB
			first := true
			for x < f.Width && !blocksEqual(f.Blocks[y][x], d.prev.Blocks[y][x]) {
				b := f.Blocks[y][x]
				if first || b.Fg != lastFg {
					buf.WriteString(b.Fg.FgString())
					lastFg = b.Fg
				}
				if first || b.Bg != lastBg {
					buf.WriteString(b.Bg.BgString())
					lastBg = b.Bg
				}
				first = false
				buf.WriteRune(b.Glyph)
				x++
			}
			buf.WriteString(color.Reset)
		}
	}

	d.prev = cloneFrame(f)
	return buf.String()
}

func blocksEqual(a, b Block) bool {
	return a.Glyph == b.Glyph && a.Fg == b.Fg && a.Bg == b.Bg
}

func cloneFrame(f *Frame) *Frame {
	c := &Frame{Width: f.Width, Height: f.Height, Blocks: make([][]Block, f.Height)}
	for y := range f.Blocks {
		row := make([]Block, f.Width)
		copy(row, f.Blocks[y])
		c.Blocks[y] = row
	}
	return c
}
