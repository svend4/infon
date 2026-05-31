package tangram

import "fmt"

// Overlap marks a cell where pieces fight over real area (not merely a shared
// hypotenuse): more than Sub sub-pixels are claimed by two or more pieces.
type Overlap struct {
	Y       int `json:"y"`
	X       int `json:"x"`
	Doubled int `json:"doubled"` // sub-pixels covered by 2+ pieces
}

// Report is the outcome of checking a figure's tiling.
type Report struct {
	Figure        string    `json:"figure"`
	Cells         int       `json:"cells"`          // total cells (H*W)
	FilledCells   int       `json:"filled_cells"`   // cells holding at least one piece
	CompleteCells int       `json:"complete_cells"` // cells whose pieces tile the WHOLE cell
	SubCovered    int       `json:"sub_covered"`    // sub-pixels covered at least once
	SubTotal      int       `json:"sub_total"`      // H*W*Sub*Sub
	Coverage      float64   `json:"coverage"`       // SubCovered / SubTotal
	Overlaps      []Overlap `json:"overlaps,omitempty"`
	WellFormed    bool      `json:"well_formed"` // no area overlaps
}

// Validate checks the figure against the tangram rule — pieces meet edge to
// edge, never stacking. Complementary pairs share a one-pixel-wide hypotenuse,
// which is expected; an overlap is only flagged when more than Sub sub-pixels
// in a cell are double-claimed (a genuine area collision, e.g. two solids).
func (f Figure) Validate() Report {
	S := glyphsetSub()
	g := f.group()
	rep := Report{
		Figure:   f.Name,
		Cells:    f.H * f.W,
		SubTotal: f.H * f.W * S * S,
	}
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			ps := g[[2]int{y, x}]
			if len(ps) == 0 {
				continue
			}
			rep.FilledCells++
			doubled, complete := 0, true
			for sy := 0; sy < S; sy++ {
				for sx := 0; sx < S; sx++ {
					cnt := 0
					for _, p := range ps {
						if coverageAt(p.Glyph, sx, sy, S) {
							cnt++
						}
					}
					if cnt == 0 {
						complete = false
					} else {
						rep.SubCovered++
					}
					if cnt >= 2 {
						doubled++
					}
				}
			}
			if doubled > S { // more than a shared hypotenuse → real collision
				rep.Overlaps = append(rep.Overlaps, Overlap{Y: y, X: x, Doubled: doubled})
			}
			if complete {
				rep.CompleteCells++
			}
		}
	}
	if rep.SubTotal > 0 {
		rep.Coverage = float64(rep.SubCovered) / float64(rep.SubTotal)
	}
	rep.WellFormed = len(rep.Overlaps) == 0
	return rep
}

// Check is Validate reduced to an error, handy in tests and catalog guards.
func (f Figure) Check() error {
	r := f.Validate()
	if !r.WellFormed {
		return fmt.Errorf("tangram %q: %d overlapping cell(s)", f.Name, len(r.Overlaps))
	}
	return nil
}
