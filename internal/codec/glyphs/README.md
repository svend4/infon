# Glyphs Package

Unicode quadrant block characters and rendering logic.

## Responsibilities

- Define 16 quadrant glyphs (▀▄▌▐▖▗▘▝ etc.)
- Map 4-bit pattern to Unicode character
- Terminal capability detection
- Fallback strategies for limited terminals

## Files

- `glyphs.go` — Glyph definitions and mapping
- `detect.go` — Terminal capability detection
- `fallback.go` — Fallback rendering for limited terminals
