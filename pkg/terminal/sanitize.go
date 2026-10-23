package terminal

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeGlyph returns r if it is safe to write to a terminal as a cell, and a
// space otherwise. Glyphs can come from decoded network frames or model output,
// so C0 controls, DEL, C1 controls and invalid runes must not reach the
// terminal, where they would act as control sequences instead of being drawn.
func SanitizeGlyph(r rune) rune {
	if !utf8.ValidRune(r) || unicode.IsControl(r) {
		return ' '
	}
	return r
}

// SanitizeText removes control characters (C0, DEL, C1) from untrusted text,
// such as a remote peer's chat message, before it is printed. Printable
// Unicode is kept; invalid UTF-8 becomes U+FFFD.
func SanitizeText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
