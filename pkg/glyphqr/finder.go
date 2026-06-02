package glyphqr

import (
	"errors"

	"github.com/svend4/infon/pkg/glyphset"
)

// finder.go gives a glyph-QR an OPTICAL skeleton: a one-cell quiet border with
// finder marks in three corners (top-left, top-right, bottom-left) and a blank
// bottom-right. That 3-of-4 asymmetry is the project's "canonical id as a finder
// pattern" — a scanner that has rectified the code to a cell matrix can recover
// its rotation with no prior key, then strip the frame and RS-decode the payload.
// (Rotation here is of the cell matrix; an optical front-end first rectifies the
// captured quad to this grid.)

// finderTok is a solid, rotation-symmetric mark, so it stays recognizable under
// any quarter-turn of the matrix.
var finderTok = string(glyphset.Glyph("full"))

// EncodeFramed wraps Encode's grid in the finder frame described above.
func EncodeFramed(data []byte, ecc int) ([][]string, error) {
	inner, err := Encode(data, ecc)
	if err != nil {
		return nil, err
	}
	return frame(inner), nil
}

// EncodeTextFramed is EncodeFramed for a string.
func EncodeTextFramed(s string, ecc int) ([][]string, error) { return EncodeFramed([]byte(s), ecc) }

func frame(inner [][]string) [][]string {
	h := len(inner)
	w := 0
	if h > 0 {
		w = len(inner[0])
	}
	H, W := h+2, w+2
	g := make([][]string, H)
	for y := 0; y < H; y++ {
		g[y] = make([]string, W)
		for x := 0; x < W; x++ {
			g[y][x] = "" // quiet border + (later) interior
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g[y+1][x+1] = inner[y][x]
		}
	}
	g[0][0] = finderTok     // TL
	g[0][W-1] = finderTok   // TR
	g[H-1][0] = finderTok   // BL
	g[H-1][W-1] = ""        // BR: deliberately empty (the orientation key)
	return g
}

func rotateCW(g [][]string) [][]string {
	h := len(g)
	if h == 0 {
		return g
	}
	w := len(g[0])
	out := make([][]string, w)
	for i := 0; i < w; i++ {
		out[i] = make([]string, h)
		for j := 0; j < h; j++ {
			out[i][j] = g[h-1-j][i]
		}
	}
	return out
}

// RotateCW returns the grid turned k quarter-turns clockwise (k may be any int).
func RotateCW(g [][]string, k int) [][]string {
	k = ((k % 4) + 4) % 4
	out := g
	for i := 0; i < k; i++ {
		out = rotateCW(out)
	}
	return out
}

func cornersCanonical(g [][]string) bool {
	h := len(g)
	if h < 2 {
		return false
	}
	w := len(g[0])
	if w < 2 {
		return false
	}
	return g[0][0] == finderTok && g[0][w-1] == finderTok &&
		g[h-1][0] == finderTok && g[h-1][w-1] == ""
}

// Orient returns how many clockwise quarter-turns canonicalize the grid, and
// whether a unique finder orientation was found.
func Orient(g [][]string) (int, bool) {
	cur := g
	for k := 0; k < 4; k++ {
		if cornersCanonical(cur) {
			return k, true
		}
		cur = rotateCW(cur)
	}
	return 0, false
}

// DecodeFramed auto-rotates a framed glyph-QR using its finder corners, strips
// the border, and decodes the (RS-corrected) payload.
func DecodeFramed(g [][]string) ([]byte, error) {
	k, ok := Orient(g)
	if !ok {
		return nil, errors.New("glyphqr: no finder pattern (cannot orient)")
	}
	return Decode(strip(RotateCW(g, k)))
}

// DecodeTextFramed is DecodeFramed returning a string.
func DecodeTextFramed(g [][]string) (string, error) {
	b, err := DecodeFramed(g)
	return string(b), err
}

func strip(g [][]string) [][]string {
	h := len(g)
	w := 0
	if h > 0 {
		w = len(g[0])
	}
	if h < 3 || w < 3 {
		return g
	}
	out := make([][]string, h-2)
	for y := 1; y < h-1; y++ {
		out[y-1] = append([]string(nil), g[y][1:w-1]...)
	}
	return out
}
