package terminal

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

// DrawText places one rune per column. Ranging over a string yields byte
// offsets, so multibyte UTF-8 (Cyrillic, '♥', box drawing) used to land at
// x+byteOffset, leaving gaps and pushing text off the right edge.
func TestDrawTextOneColumnPerRune(t *testing.T) {
	f := NewFrame(12, 1)
	f.Fill('.', color.Black, color.Black)
	f.DrawText(1, 0, "♥ Привет", color.White, color.Black)

	var got []rune
	for _, b := range f.Blocks[0] {
		got = append(got, b.Glyph)
	}
	if want := ".♥ Привет..."; string(got) != want {
		t.Errorf("row = %q, want %q", string(got), want)
	}
}

// Text that starts left of or above the frame is clipped, not a panic.
func TestDrawTextClipsNegativeOrigin(t *testing.T) {
	f := NewFrame(4, 2)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DrawText panicked: %v", r)
		}
	}()
	f.DrawText(-2, 0, "abcdef", color.White, color.Black)
	f.DrawText(0, -1, "zz", color.White, color.Black)
	var got []rune
	for _, b := range f.Blocks[0] {
		got = append(got, b.Glyph)
	}
	if want := "cdef"; string(got) != want {
		t.Errorf("row = %q, want %q", string(got), want)
	}
}
