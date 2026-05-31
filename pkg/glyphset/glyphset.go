// Package glyphset turns the hand-drawn sub-cell alphabet (dots, diagonals,
// triangles, crossed box) into two things:
//
//  1. a DIAGONAL/TRIANGLE render primitive — RenderTriangles maps an image to a
//     terminal.Frame using triangle glyphs (◤◥◣◢) and the quadrant set, so slanted
//     edges (sun rays, sails, mountains, heraldic charges) come out as crisp
//     diagonals instead of the quadrant codec's stair-steps; and
//  2. a digitized catalog (Marks) of the alphabet, with a Chart renderer — the
//     clean, code-backed version of the paper sheet.
//
// coverage() is the single source of truth: it knows the S×S fill mask of every
// glyph, which drives both the renderer (best-match) and a faithful rasterizer.
package glyphset

import (
	"image"
	stdcolor "image/color"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// sub is the sub-sampling resolution per cell axis (sub×sub mask).
const sub = 8

// coverage reports whether sub-pixel (x,y) of an S×S cell is "on" (foreground)
// for glyph r. Filled glyphs cover an area; line glyphs cover a thin stroke.
func coverage(r rune, x, y, S int) bool {
	h := S / 2
	switch r {
	case ' ':
		return false
	case '█':
		return true
	case '▀':
		return y < h
	case '▄':
		return y >= h
	case '▌':
		return x < h
	case '▐':
		return x >= h
	case '▘':
		return x < h && y < h
	case '▝':
		return x >= h && y < h
	case '▖':
		return x < h && y >= h
	case '▗':
		return x >= h && y >= h
	case '▞': // tr + bl
		return (x >= h && y < h) || (x < h && y >= h)
	case '▚': // tl + br
		return (x < h && y < h) || (x >= h && y >= h)
	case '▛': // all but br
		return !(x >= h && y >= h)
	case '▜': // all but bl
		return !(x < h && y >= h)
	case '▙': // all but tr
		return !(x >= h && y < h)
	case '▟': // all but tl
		return !(x < h && y < h)
	case '◤': // upper-left triangle (hypotenuse = anti-diagonal)
		return x+y <= S-1
	case '◢': // lower-right triangle (complement of ◤)
		return x+y >= S-1
	case '◥': // upper-right triangle (hypotenuse = main diagonal)
		return x >= y
	case '◣': // lower-left triangle (complement of ◥)
		return x <= y
	case '╲': // main diagonal line
		return abs(x-y) <= 1
	case '╱': // anti-diagonal line
		return abs(x+y-(S-1)) <= 1
	case '╳': // both diagonals
		return abs(x-y) <= 1 || abs(x+y-(S-1)) <= 1
	case '□': // box border
		return x == 0 || x == S-1 || y == 0 || y == S-1
	case '⊠': // box + both diagonals
		return x == 0 || x == S-1 || y == 0 || y == S-1 || abs(x-y) <= 1 || abs(x+y-(S-1)) <= 1
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// candidates are the AREA glyphs the triangle renderer chooses from (no line
// glyphs — those are for the alphabet/chart, not image fill).
var candidates = []rune{
	' ', '█', '▀', '▄', '▌', '▐',
	'▘', '▝', '▖', '▗', '▞', '▚', '▛', '▜', '▙', '▟',
	'◤', '◥', '◣', '◢',
}

// RenderTriangles renders an image into a cols×rows Frame, choosing for each cell
// the glyph whose fill mask best matches the cell's bright region — yielding crisp
// diagonal edges. Foreground = average of the bright sub-pixels, background = the
// rest.
func RenderTriangles(img image.Image, cols, rows int) *terminal.Frame {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	f := terminal.NewFrame(cols, rows)
	if iw == 0 || ih == 0 {
		return f
	}
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			var R, G, B [sub][sub]float64
			var lum [sub][sub]float64
			mean := 0.0
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					px := b.Min.X + (cx*sub+sx)*iw/(cols*sub)
					py := b.Min.Y + (cy*sub+sy)*ih/(rows*sub)
					rr, gg, bb, _ := img.At(px, py).RGBA()
					r8, g8, b8 := float64(rr>>8), float64(gg>>8), float64(bb>>8)
					R[sy][sx], G[sy][sx], B[sy][sx] = r8, g8, b8
					l := 0.2126*r8 + 0.7152*g8 + 0.0722*b8
					lum[sy][sx] = l
					mean += l
				}
			}
			mean /= float64(sub * sub)
			var on [sub][sub]bool
			var fr, fg, fb, nf, br, bg, bb2, nb float64
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					if lum[sy][sx] >= mean {
						on[sy][sx] = true
						fr += R[sy][sx]
						fg += G[sy][sx]
						fb += B[sy][sx]
						nf++
					} else {
						br += R[sy][sx]
						bg += G[sy][sx]
						bb2 += B[sy][sx]
						nb++
					}
				}
			}
			fgc := avg(fr, fg, fb, nf)
			bgc := avg(br, bg, bb2, nb)
			best := ' '
			bestMiss := 1 << 30
			for _, c := range candidates {
				miss := 0
				for sy := 0; sy < sub; sy++ {
					for sx := 0; sx < sub; sx++ {
						if coverage(c, sx, sy, sub) != on[sy][sx] {
							miss++
						}
					}
				}
				if miss < bestMiss {
					bestMiss, best = miss, c
				}
			}
			f.SetBlock(cx, cy, best, fgc, bgc)
		}
	}
	return f
}

func avg(r, g, b, n float64) color.RGB {
	if n == 0 {
		return color.RGB{}
	}
	return color.RGB{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n)}
}

// Rasterize draws a Frame to an RGBA image, painting each cell's glyph MASK (not
// just its color) so triangle/diagonal/quadrant shapes are reproduced faithfully.
// cell is the pixel size of one terminal cell.
func Rasterize(f *terminal.Frame, cell int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, f.Width*cell, f.Height*cell))
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			blk := f.Blocks[y][x]
			for py := 0; py < cell; py++ {
				sy := py * sub / cell
				for px := 0; px < cell; px++ {
					sx := px * sub / cell
					c := blk.Bg
					if coverage(blk.Glyph, sx, sy, sub) {
						c = blk.Fg
					}
					img.SetRGBA(x*cell+px, y*cell+py, stdcolor.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
				}
			}
		}
	}
	return img
}

// Mark is one named glyph in the digitized alphabet.
type Mark struct {
	Name  string
	Glyph rune
}

// Marks is the catalog distilled from the hand-drawn sheet: blanks, halves,
// quadrants (incl. the diagonal pair), triangles, diagonal lines and the box.
var Marks = []Mark{
	{"blank", ' '}, {"full", '█'},
	{"top", '▀'}, {"bottom", '▄'}, {"left", '▌'}, {"right", '▐'},
	{"q-tl", '▘'}, {"q-tr", '▝'}, {"q-bl", '▖'}, {"q-br", '▗'},
	{"diag-/", '▞'}, {"diag-\\", '▚'},
	{"3-tl", '▛'}, {"3-tr", '▜'}, {"3-bl", '▙'}, {"3-br", '▟'},
	{"tri-ul", '◤'}, {"tri-ur", '◥'}, {"tri-ll", '◣'}, {"tri-lr", '◢'},
	{"line-/", '╱'}, {"line-\\", '╲'}, {"cross", '╳'},
	{"box", '□'}, {"box-x", '⊠'},
}

// Chart lays the whole alphabet out as a labeled grid (perCol marks per row),
// ready to Rasterize into the digital version of the paper sheet.
func Chart(perRow int) *terminal.Frame {
	if perRow < 1 {
		perRow = 8
	}
	rows := (len(Marks) + perRow - 1) / perRow
	gap := 2
	f := terminal.NewFrame(perRow*gap, rows*gap)
	navy := color.RGB{R: 20, G: 24, B: 56}
	white := color.RGB{R: 235, G: 238, B: 245}
	f.Fill(' ', navy, navy)
	for i, m := range Marks {
		cx := (i % perRow) * gap
		cy := (i / perRow) * gap
		f.SetBlock(cx, cy, m.Glyph, white, navy)
	}
	return f
}
