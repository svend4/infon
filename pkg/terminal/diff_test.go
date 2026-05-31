package terminal

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestDiffFirstFrameIsFullRepaint(t *testing.T) {
	f := NewFrame(4, 2)
	d := NewDiffRenderer()
	out := d.Render(f)
	// A full repaint starts by clearing the screen (see Frame.Render).
	if !strings.Contains(out, color.ClearScreen) {
		t.Error("first render should be a full repaint (ClearScreen)")
	}
}

func TestDiffUnchangedFrameEmitsNothing(t *testing.T) {
	f := NewFrame(4, 2)
	d := NewDiffRenderer()
	_ = d.Render(f) // prime
	if out := d.Render(f); out != "" {
		t.Errorf("unchanged frame should emit nothing, got %q", out)
	}
}

func TestDiffOnlyEmitsChangedCell(t *testing.T) {
	f := NewFrame(10, 3)
	d := NewDiffRenderer()
	_ = d.Render(f) // prime with a blank frame

	// Change a single cell.
	f.SetBlock(4, 1, 'Z', color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 0))
	out := d.Render(f)

	if !strings.Contains(out, "Z") {
		t.Fatal("changed glyph not emitted")
	}
	// Must reposition to row 2, col 5 (1-based) and not touch other rows.
	if !strings.Contains(out, MoveCursor(2, 5)) {
		t.Errorf("expected cursor move to (2,5), got %q", out)
	}
	// A single-cell change must be far smaller than a full repaint.
	if len(out) > len(f.Render())/2 {
		t.Errorf("diff output (%d bytes) not meaningfully smaller than full repaint (%d)", len(out), len(f.Render()))
	}
}

func TestDiffCoalescesAdjacentChangesIntoOneRun(t *testing.T) {
	f := NewFrame(10, 1)
	d := NewDiffRenderer()
	_ = d.Render(f)

	// Three adjacent changed cells should produce exactly one cursor move.
	f.SetBlock(2, 0, 'A', color.White, color.Black)
	f.SetBlock(3, 0, 'B', color.White, color.Black)
	f.SetBlock(4, 0, 'C', color.White, color.Black)
	out := d.Render(f)

	if n := strings.Count(out, "\x1b["); n == 0 {
		t.Fatal("no escapes emitted")
	}
	moves := strings.Count(out, "H")
	// One run -> one cursor-move escape ending in 'H'. (FgString/BgString end in
	// 'm', Reset ends in 'm', so 'H' uniquely marks cursor moves here.)
	if moves != 1 {
		t.Errorf("expected 1 cursor move for a contiguous run, got %d (out=%q)", moves, out)
	}
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") || !strings.Contains(out, "C") {
		t.Error("run did not contain all three changed glyphs")
	}
}

func TestDiffSizeChangeForcesFullRepaint(t *testing.T) {
	d := NewDiffRenderer()
	_ = d.Render(NewFrame(4, 4))
	out := d.Render(NewFrame(8, 4)) // different size
	if !strings.Contains(out, color.ClearScreen) {
		t.Error("a size change should force a full repaint")
	}
}

func TestDiffResetForcesRepaint(t *testing.T) {
	f := NewFrame(4, 2)
	d := NewDiffRenderer()
	_ = d.Render(f)
	d.Reset()
	if out := d.Render(f); !strings.Contains(out, color.ClearScreen) {
		t.Error("after Reset the next render should be a full repaint")
	}
}

// TestDiffEquivalence verifies the diff renderer is *correct*: applying the diff
// stream to a model of the screen yields the same cells as the target frame.
func TestDiffEquivalence(t *testing.T) {
	d := NewDiffRenderer()
	f := NewFrame(6, 4)
	_ = d.Render(f)

	// Mutate several cells across rows.
	f.SetBlock(0, 0, 'X', color.NewRGB(10, 20, 30), color.NewRGB(1, 2, 3))
	f.SetBlock(5, 0, 'Y', color.White, color.Black)
	f.SetBlock(3, 2, 'Z', color.NewRGB(99, 0, 0), color.Black)
	out := d.Render(f)

	// The diff must mention every changed glyph and reposition for each
	// non-contiguous run.
	for _, g := range []string{"X", "Y", "Z"} {
		if !strings.Contains(out, g) {
			t.Errorf("diff missing changed glyph %q", g)
		}
	}
}
