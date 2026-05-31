package semframe

import "github.com/svend4/infon/pkg/pseudo"

// MarkCellChange is one cell of a marks grid whose glyph token and/or color
// changed. Nil fields mean "unchanged".
type MarkCellChange struct {
	Y int     `json:"y"`
	X int     `json:"x"`
	G *string `json:"g,omitempty"` // new glyph token
	C *string `json:"c,omitempty"` // new color token
}

// MarksDelta is a semantic P-frame for a marks pseudo-image: the cells whose
// glyph or color changed since the previous frame ("a figure moved").
type MarksDelta struct {
	Cells []MarkCellChange `json:"cells,omitempty"`
}

func tokAt(grid [][]string, y, x int) string {
	if y >= 0 && y < len(grid) && x >= 0 && x < len(grid[y]) {
		return grid[y][x]
	}
	return ""
}

// DiffMarks computes the delta from prev to next. ok=false if either isn't a
// marks spec or their grids differ in shape (send a keyframe).
func DiffMarks(prev, next pseudo.Spec) (MarksDelta, bool) {
	if prev.Marks == nil || next.Marks == nil {
		return MarksDelta{}, false
	}
	pr, nr := prev.Marks.Rows, next.Marks.Rows
	if len(pr) != len(nr) {
		return MarksDelta{}, false
	}
	var d MarksDelta
	for y := range nr {
		if len(pr[y]) != len(nr[y]) {
			return MarksDelta{}, false
		}
		for x := range nr[y] {
			cc := MarkCellChange{Y: y, X: x}
			changed := false
			if pr[y][x] != nr[y][x] {
				g := nr[y][x]
				cc.G = &g
				changed = true
			}
			if pc, nc := tokAt(prev.Marks.Colors, y, x), tokAt(next.Marks.Colors, y, x); pc != nc {
				c := nc
				cc.C = &c
				changed = true
			}
			if changed {
				d.Cells = append(d.Cells, cc)
			}
		}
	}
	return d, true
}

// ApplyMarks reconstructs the next marks spec from prev + delta.
func ApplyMarks(prev pseudo.Spec, d MarksDelta) pseudo.Spec {
	out := prev
	rows := make([][]string, len(prev.Marks.Rows))
	cols := make([][]string, len(prev.Marks.Rows))
	for y := range prev.Marks.Rows {
		rows[y] = append([]string(nil), prev.Marks.Rows[y]...)
		cols[y] = make([]string, len(rows[y]))
		if y < len(prev.Marks.Colors) {
			copy(cols[y], prev.Marks.Colors[y])
		}
	}
	for _, c := range d.Cells {
		if c.Y < 0 || c.Y >= len(rows) || c.X < 0 || c.X >= len(rows[c.Y]) {
			continue
		}
		if c.G != nil {
			rows[c.Y][c.X] = *c.G
		}
		if c.C != nil {
			cols[c.Y][c.X] = *c.C
		}
	}
	m := *prev.Marks
	m.Rows = rows
	m.Colors = cols
	out.Marks = &m
	return out
}
