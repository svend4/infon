// Package arenacast is the bridge that fuses the three bodies of the project
// into one core: it renders a live pkg/arena match as a marks pseudo-image
// (pkg/pseudo), so the agent-world can be streamed through pkg/semvideo exactly
// like a TVCP semantic video call — a keyframe plus semframe P-frame deltas
// wrapped in Reed-Solomon, a few bytes per tick that survive packet loss.
// "Watch the war of two neural nets as a 200-byte/s call."
package arenacast

import (
	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/pseudo"
)

// FactionGlyph maps a faction to a sub-cell glyph token so factions are
// distinguishable by SHAPE alone (and so MarksIoU measures real meaning).
func FactionGlyph(f uint8) string {
	if f == 0 {
		return "tri-ul" // ◤ blue
	}
	return "tri-lr" // ◢ red
}

// FactionColor maps a faction to a color token (resolved by the renderer).
func FactionColor(f uint8) string {
	if f == 0 {
		return "skyblue"
	}
	return "orange"
}

// Blank is an all-empty marks spec of the board's shape — what a spectator shows
// before the first keyframe arrives.
func Blank(w, h int) pseudo.Spec {
	rows := make([][]string, h)
	cols := make([][]string, h)
	for y := 0; y < h; y++ {
		rows[y] = make([]string, w)
		cols[y] = make([]string, w)
	}
	return pseudo.Spec{Format: pseudo.FormatMarks, Cols: w, Rows: h,
		Marks: &pseudo.MarkArt{Rows: rows, Colors: cols, Bg: "navy"}}
}

// Spec renders the current arena board as a marks pseudo-image: one glyph token
// (shape+color by faction) per living unit over blank terrain. The grid shape is
// constant for a match, so consecutive Specs diff into tiny semframe P-frames.
func Spec(a *arena.Arena) pseudo.Spec {
	s := Blank(a.W, a.H)
	for _, u := range a.Units {
		if !u.Alive {
			continue
		}
		x, y := int(u.X), int(u.Y)
		if x < 0 || x >= a.W || y < 0 || y >= a.H {
			continue
		}
		s.Marks.Rows[y][x] = FactionGlyph(u.Faction)
		s.Marks.Colors[y][x] = FactionColor(u.Faction)
	}
	return s
}
