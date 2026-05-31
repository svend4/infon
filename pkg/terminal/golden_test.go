package terminal

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

// Golden tests (roadmap D5): pin the exact ANSI byte stream produced for known
// frames, so any unintended change to the renderer is caught immediately. We
// assert on a normalized, human-readable description of the escape sequence
// rather than embedding raw control bytes, which keeps the test legible and
// stable across editors.

// describe turns an ANSI string into a readable token stream:
//
//	CLEAR, FG(r,g,b), BG(r,g,b), RESET, NL, and literal runs in <...>.
func describe(s string) string {
	var b strings.Builder
	runes := []rune(s)
	i := 0
	flushLit := func(lit *strings.Builder) {
		if lit.Len() > 0 {
			b.WriteString("<" + lit.String() + ">")
			lit.Reset()
		}
	}
	var lit strings.Builder
	for i < len(runes) {
		if runes[i] == '\x1b' {
			flushLit(&lit)
			// Read until a letter terminator (m or H or J).
			j := i + 1
			for j < len(runes) && !isAnsiFinal(runes[j]) {
				j++
			}
			seq := string(runes[i : j+1])
			b.WriteString(token(seq))
			i = j + 1
			continue
		}
		if runes[i] == '\n' {
			flushLit(&lit)
			b.WriteString("NL")
			i++
			continue
		}
		lit.WriteRune(runes[i])
		i++
	}
	flushLit(&lit)
	return b.String()
}

func isAnsiFinal(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func token(seq string) string {
	switch {
	case seq == color.Reset:
		return "RESET"
	case seq == "\x1b[2J" || seq == color.ClearScreen:
		return "CLEAR"
	case strings.HasPrefix(seq, "\x1b[38;2;"):
		return "FG(" + strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b[38;2;"), "m") + ")"
	case strings.HasPrefix(seq, "\x1b[48;2;"):
		return "BG(" + strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b[48;2;"), "m") + ")"
	case strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "H"):
		return "AT(" + strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b["), "H") + ")"
	default:
		return "ESC?"
	}
}

func TestGoldenSingleWhiteCell(t *testing.T) {
	f := NewFrame(1, 1)
	f.SetBlock(0, 0, 'X', color.White, color.Black)

	got := describe(f.Render())
	// ClearScreen (which is ESC[2J ESC[H -> CLEAR + AT), then fg white, bg black,
	// the glyph, a RESET and newline.
	want := "CLEARAT()FG(255;255;255)BG(0;0;0)<X>RESETNL"
	if got != want {
		t.Errorf("golden mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestGoldenColorRunCoalescing(t *testing.T) {
	// Three cells of the SAME color must emit the color escapes once, then three
	// glyphs — proving the renderer's run-length color optimization.
	f := NewFrame(3, 1)
	red := color.NewRGB(200, 10, 10)
	for x := 0; x < 3; x++ {
		f.SetBlock(x, 0, 'A'+rune(x), red, color.Black)
	}
	got := describe(f.Render())
	want := "CLEARAT()FG(200;10;10)BG(0;0;0)<ABC>RESETNL"
	if got != want {
		t.Errorf("golden mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestGoldenColorChangeReemitsEscape(t *testing.T) {
	// Adjacent cells of DIFFERENT fg must re-emit FG between them.
	f := NewFrame(2, 1)
	f.SetBlock(0, 0, 'A', color.NewRGB(10, 20, 30), color.Black)
	f.SetBlock(1, 0, 'B', color.NewRGB(40, 50, 60), color.Black)
	got := describe(f.Render())
	want := "CLEARAT()FG(10;20;30)BG(0;0;0)<A>FG(40;50;60)<B>RESETNL"
	if got != want {
		t.Errorf("golden mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestGoldenDiffRendererSingleCell(t *testing.T) {
	// The diff renderer's incremental output for one changed cell is pinned.
	d := NewDiffRenderer()
	f := NewFrame(5, 2)
	_ = d.Render(f) // prime with blank frame

	f.SetBlock(2, 1, 'Z', color.NewRGB(1, 2, 3), color.NewRGB(4, 5, 6))
	got := describe(d.Render(f))
	// Cursor to row 2 col 3 (1-based), then the colors, glyph, reset.
	want := "AT(2;3)FG(1;2;3)BG(4;5;6)<Z>RESET"
	if got != want {
		t.Errorf("diff golden mismatch:\n got %q\nwant %q", got, want)
	}
}
