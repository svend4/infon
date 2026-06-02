// Package microfont is a tiny built-in 3x5 bitmap font so the project's image
// previews and GIFs can carry text labels with NO system font dependency — the
// "mini raster font from masks" the GLYPH_applications note asked for. Each glyph
// is five rows of three bits; Draw stamps scaled blocks onto any draw.Image.
package microfont

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// glyphs maps an (uppercased) rune to 5 rows of 3 bits (bit 2 = leftmost column).
var glyphs = map[rune][5]byte{
	'A': {7, 5, 7, 5, 5}, 'B': {6, 5, 6, 5, 6}, 'C': {3, 4, 4, 4, 3}, 'D': {6, 5, 5, 5, 6},
	'E': {7, 4, 6, 4, 7}, 'F': {7, 4, 6, 4, 4}, 'G': {3, 4, 5, 5, 3}, 'H': {5, 5, 7, 5, 5},
	'I': {7, 2, 2, 2, 7}, 'J': {1, 1, 1, 5, 2}, 'K': {5, 6, 4, 6, 5}, 'L': {4, 4, 4, 4, 7},
	'M': {5, 7, 7, 5, 5}, 'N': {5, 7, 7, 7, 5}, 'O': {2, 5, 5, 5, 2}, 'P': {6, 5, 6, 4, 4},
	'Q': {2, 5, 5, 6, 3}, 'R': {6, 5, 6, 5, 5}, 'S': {3, 4, 2, 1, 6}, 'T': {7, 2, 2, 2, 2},
	'U': {5, 5, 5, 5, 7}, 'V': {5, 5, 5, 5, 2}, 'W': {5, 5, 7, 7, 5}, 'X': {5, 5, 2, 5, 5},
	'Y': {5, 5, 2, 2, 2}, 'Z': {7, 1, 2, 4, 7},
	'0': {7, 5, 5, 5, 7}, '1': {2, 6, 2, 2, 7}, '2': {6, 1, 2, 4, 7}, '3': {6, 1, 2, 1, 6},
	'4': {5, 5, 7, 1, 1}, '5': {7, 4, 6, 1, 6}, '6': {3, 4, 6, 5, 2}, '7': {7, 1, 2, 2, 2},
	'8': {2, 5, 2, 5, 2}, '9': {2, 5, 3, 1, 6},
	' ': {0, 0, 0, 0, 0}, '.': {0, 0, 0, 0, 2}, ',': {0, 0, 0, 2, 4}, ':': {0, 2, 0, 2, 0},
	'-': {0, 0, 7, 0, 0}, '/': {1, 1, 2, 4, 4}, '#': {5, 7, 5, 7, 5}, '!': {2, 2, 2, 0, 2},
	'?': {6, 1, 2, 0, 2}, '(': {2, 4, 4, 4, 2}, ')': {2, 1, 1, 1, 2}, '+': {0, 2, 7, 2, 0},
	'=': {0, 7, 0, 7, 0}, '%': {5, 1, 2, 4, 5}, '*': {5, 2, 7, 2, 5},
}

const (
	gw, gh = 3, 5 // glyph cell in font pixels
	gap    = 1    // inter-glyph spacing in font pixels
)

// Measure returns the pixel size of s rendered at the given scale.
func Measure(s string, scale int) (w, h int) {
	if scale < 1 {
		scale = 1
	}
	n := len([]rune(s))
	if n == 0 {
		return 0, gh * scale
	}
	return (n*(gw+gap) - gap) * scale, gh * scale
}

// Draw stamps s onto dst at (x,y) in color c, scaled. It returns the x just past
// the text. Unknown runes render as a space.
func Draw(dst draw.Image, x, y, scale int, s string, c color.Color) int {
	if scale < 1 {
		scale = 1
	}
	for _, r := range strings.ToUpper(s) {
		g, ok := glyphs[r]
		if !ok {
			g = glyphs[' ']
		}
		for ry := 0; ry < gh; ry++ {
			bits := g[ry]
			for rx := 0; rx < gw; rx++ {
				if bits&(1<<uint(gw-1-rx)) == 0 {
					continue
				}
				block(dst, x+rx*scale, y+ry*scale, scale, c)
			}
		}
		x += (gw + gap) * scale
	}
	return x
}

func block(dst draw.Image, x, y, s int, c color.Color) {
	if rgba, ok := dst.(*image.RGBA); ok {
		draw.Draw(rgba, image.Rect(x, y, x+s, y+s), &image.Uniform{c}, image.Point{}, draw.Src)
		return
	}
	for dy := 0; dy < s; dy++ {
		for dx := 0; dx < s; dx++ {
			dst.Set(x+dx, y+dy, c)
		}
	}
}
