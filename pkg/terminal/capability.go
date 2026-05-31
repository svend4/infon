package terminal

import (
	"os"
	"strings"
)

// Capability describes what the current terminal can render, so the encoder can
// pick the best graphics mode and degrade gracefully on limited terminals
// (roadmap D2). Detection is heuristic and env-based — it never blocks or writes
// escape codes — because a robust query/response probe needs raw-TTY round trips
// that aren't always available (pipes, CI, dumb terminals).
type Capability struct {
	TrueColor bool // 24-bit color (\x1b[38;2;…)
	Unicode13 bool // Symbols for Legacy Computing (sextants) likely present
	Sixel     bool // Sixel bitmap graphics
	Kitty     bool // Kitty graphics protocol
	ITerm     bool // iTerm2 inline images
}

// DetectCapability inspects the environment (TERM, COLORTERM, TERM_PROGRAM,
// LANG, and a few program-specific vars) and returns a best-effort Capability.
func DetectCapability() Capability {
	env := func(k string) string { return strings.ToLower(os.Getenv(k)) }
	term := env("TERM")
	colorterm := env("COLORTERM")
	termProgram := env("TERM_PROGRAM")
	lang := env("LANG") + env("LC_ALL") + env("LC_CTYPE")

	c := Capability{}

	// True color: COLORTERM=truecolor|24bit is the standard signal; many modern
	// terminals also imply it.
	c.TrueColor = strings.Contains(colorterm, "truecolor") ||
		strings.Contains(colorterm, "24bit") ||
		strings.Contains(term, "kitty") ||
		strings.Contains(term, "alacritty") ||
		termProgram == "iterm.app" ||
		termProgram == "wezterm" ||
		termProgram == "vscode"

	// UTF-8 locale is a prerequisite for the legacy-computing block glyphs.
	utf8 := strings.Contains(lang, "utf-8") || strings.Contains(lang, "utf8")
	// These programs ship fonts with sextant coverage.
	c.Unicode13 = utf8 && (strings.Contains(term, "kitty") ||
		strings.Contains(term, "alacritty") ||
		termProgram == "wezterm" ||
		termProgram == "iterm.app" ||
		termProgram == "contour" ||
		termProgram == "rio")

	c.Kitty = strings.Contains(term, "kitty") || termProgram == "ghostty"
	c.ITerm = termProgram == "iterm.app" || termProgram == "wezterm"
	// Sixel: a handful of terminals advertise it via TERM or are known to support it.
	c.Sixel = strings.Contains(term, "sixel") ||
		strings.Contains(term, "mlterm") ||
		strings.Contains(term, "foot") ||
		termProgram == "wezterm" ||
		strings.Contains(term, "contour")

	return c
}

// BestBlitMode names the highest-fidelity *glyph* mode this terminal should use.
// (Pixel protocols like Sixel/Kitty are handled separately by the renderer.)
// Returns one of: "sextant", "halfblock", "quadrant".
func (c Capability) BestBlitMode() string {
	switch {
	case c.Unicode13:
		return "sextant"
	case c.TrueColor:
		return "halfblock"
	default:
		return "quadrant"
	}
}

// SupportsPixelProtocol reports whether a true bitmap protocol is available, so
// the caller can choose to send a real image instead of glyph approximation.
func (c Capability) SupportsPixelProtocol() bool {
	return c.Sixel || c.Kitty || c.ITerm
}
