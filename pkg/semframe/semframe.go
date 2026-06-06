// Package semframe encodes SEMANTIC P-frames for pseudo-images: instead of
// sending a whole spec each frame, send only the cells (and palette) that
// changed since the previous frame. This is inter-frame compression at the level
// of MEANING, not pixels — a keyframe (full spec) followed by tiny deltas. When
// little moves, a "video" frame costs tens of bytes.
package semframe

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/pseudo"
)

// CellChange is one grid cell whose color token changed.
type CellChange struct {
	Y int    `json:"y"`
	X int    `json:"x"`
	C string `json:"c"`
}

// GridDelta is a semantic P-frame for a grid/pixels pseudo-image.
type GridDelta struct {
	Palette *string      `json:"palette,omitempty"` // non-nil = palette changed
	Cells   []CellChange `json:"cells,omitempty"`
}

func gridOf(s pseudo.Spec) *pseudo.Grid {
	if s.Grid != nil {
		return s.Grid
	}
	return s.Pixels
}

// Diff computes the delta from prev to next. ok=false when the two specs are not
// both grid/pixels of identical dimensions — the caller should send a keyframe.
func Diff(prev, next pseudo.Spec) (GridDelta, bool) {
	pg, ng := gridOf(prev), gridOf(next)
	if pg == nil || ng == nil || len(pg.Rows) != len(ng.Rows) {
		return GridDelta{}, false
	}
	var d GridDelta
	if prev.Palette != next.Palette {
		p := next.Palette
		d.Palette = &p
	}
	for y := range ng.Rows {
		if len(pg.Rows[y]) != len(ng.Rows[y]) {
			return GridDelta{}, false
		}
		for x := range ng.Rows[y] {
			if pg.Rows[y][x] != ng.Rows[y][x] {
				d.Cells = append(d.Cells, CellChange{Y: y, X: x, C: ng.Rows[y][x]})
			}
		}
	}
	return d, true
}

// Apply reconstructs the next spec from prev + delta.
func Apply(prev pseudo.Spec, d GridDelta) pseudo.Spec {
	out := prev
	pg := gridOf(prev)
	rows := make([][]string, len(pg.Rows))
	for y := range pg.Rows {
		rows[y] = append([]string(nil), pg.Rows[y]...)
	}
	for _, c := range d.Cells {
		if c.Y >= 0 && c.Y < len(rows) && c.X >= 0 && c.X < len(rows[c.Y]) {
			rows[c.Y][c.X] = c.C
		}
	}
	out.Grid = &pseudo.Grid{Rows: rows}
	out.Pixels = nil
	if out.Format == "" {
		out.Format = pseudo.FormatGrid
	}
	if d.Palette != nil {
		out.Palette = *d.Palette
	}
	return out
}

// Bytes returns the JSON size of any value (helper for measuring savings).
func Bytes(v any) int {
	b, _ := json.Marshal(v)
	return len(b)
}
