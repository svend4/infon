package glyphqr

import (
	"errors"
	"image"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/terminal"
)

// A glyph-QR is a grid of glyph-symbols (16-glyph alphabet, one nibble per cell)
// carrying a small replicated header and Reed-Solomon parity, so it decodes even
// when several cells are wrong. Layout: [header: 3 copies of (ecc,N) = 12 cells]
// then [codeword: 2 cells per byte]. The header is majority-voted; the codeword
// is RS-corrected (up to ecc/2 byte errors).

const magic = 0xA5

var qrAlphabet = []string{
	"full", "tri-ul", "tri-ur", "tri-ll", "tri-lr", "top", "bottom", "left",
	"right", "q-tl", "q-tr", "q-bl", "q-br", "diag-/", "diag-\\", "box",
}

var qrDigit = func() map[string]int {
	m := make(map[string]int, len(qrAlphabet))
	for i, name := range qrAlphabet {
		m[string(glyphset.Glyph(name))] = i
	}
	return m
}()

func cellToken(d int) string { return string(glyphset.Glyph(qrAlphabet[d&0xf])) }

func bytesToNibbles(b []byte) []int {
	out := make([]int, 0, 2*len(b))
	for _, x := range b {
		out = append(out, int(x>>4), int(x&0xf))
	}
	return out
}

func nibblesToBytes(n []int) []byte {
	out := make([]byte, len(n)/2)
	for i := range out {
		out[i] = byte(n[2*i]<<4 | (n[2*i+1] & 0xf))
	}
	return out
}

func maj3(a, b, c byte) byte {
	if a == b || a == c {
		return a
	}
	if b == c {
		return b
	}
	return a
}

func isqrtCeil(n int) int {
	w := 1
	for w*w < n {
		w++
	}
	return w
}

// Encode packs data into a glyph-QR grid with ecc parity bytes (ecc clamped to
// 2..30; corrects up to ecc/2 wrong cells-per-byte). data is at most 253 bytes.
func Encode(data []byte, ecc int) ([][]string, error) {
	if ecc < 2 {
		ecc = 2
	}
	if ecc > 30 {
		ecc = 30
	}
	if len(data) > 253 {
		return nil, errors.New("glyphqr: data too long (max 253 bytes)")
	}
	msg := append([]byte{magic, byte(len(data))}, data...)
	cw := rsEncode(msg, ecc)
	header := []byte{byte(ecc), byte(len(cw))}

	var nib []int
	for rep := 0; rep < 3; rep++ {
		nib = append(nib, bytesToNibbles(header)...)
	}
	nib = append(nib, bytesToNibbles(cw)...)

	w := isqrtCeil(len(nib))
	if w < 6 {
		w = 6
	}
	h := (len(nib) + w - 1) / w
	grid := make([][]string, h)
	k := 0
	for y := 0; y < h; y++ {
		grid[y] = make([]string, w)
		for x := 0; x < w; x++ {
			if k < len(nib) {
				grid[y][x] = cellToken(nib[k])
				k++
			} else {
				grid[y][x] = cellToken(0)
			}
		}
	}
	return grid, nil
}

// EncodeText is Encode for a string.
func EncodeText(s string, ecc int) ([][]string, error) { return Encode([]byte(s), ecc) }

// Decode reads a glyph-QR grid back to bytes, majority-voting the header and
// Reed-Solomon-correcting the codeword.
func Decode(grid [][]string) ([]byte, error) {
	var nib []int
	for _, row := range grid {
		for _, c := range row {
			nib = append(nib, qrDigit[c]) // unknown glyph -> 0
		}
	}
	if len(nib) < 12 {
		return nil, errors.New("glyphqr: grid too small")
	}
	hb := nibblesToBytes(nib[:12]) // 3 copies of [ecc, N]
	ecc := int(maj3(hb[0], hb[2], hb[4]))
	n := int(maj3(hb[1], hb[3], hb[5]))
	if ecc < 2 || ecc > 30 || n < ecc+2 {
		return nil, errors.New("glyphqr: bad header")
	}
	need := 12 + 2*n
	if len(nib) < need {
		return nil, errors.New("glyphqr: truncated grid")
	}
	cw := nibblesToBytes(nib[12:need])
	msg, err := rsDecode(cw, ecc)
	if err != nil {
		return nil, err
	}
	if len(msg) < 2 || msg[0] != magic {
		return nil, errors.New("glyphqr: undecodable (bad magic)")
	}
	dlen := int(msg[1])
	if 2+dlen > len(msg) {
		return nil, errors.New("glyphqr: bad length field")
	}
	return msg[2 : 2+dlen], nil
}

// DecodeText is Decode returning a string.
func DecodeText(grid [][]string) (string, error) {
	b, err := Decode(grid)
	return string(b), err
}

// Render rasterises the grid to an image (cell px per cell), dark glyphs on
// light — a scannable picture of the code.
func Render(grid [][]string, cell int) image.Image {
	if len(grid) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	h, w := len(grid), len(grid[0])
	f := terminal.NewFrame(w, h)
	bg := color.RGB{R: 245, G: 245, B: 248}
	ink := color.RGB{R: 20, G: 24, B: 40}
	f.Fill(' ', bg, bg)
	for y := 0; y < h; y++ {
		for x := 0; x < w && x < len(grid[y]); x++ {
			g := ' '
			for _, r := range grid[y][x] {
				g = r
				break
			}
			f.SetBlock(x, y, g, ink, bg)
		}
	}
	return glyphset.RasterizeSize(f, w*cell, h*cell)
}

// Rows returns the grid as printable strings (one glyph rune per cell).
func Rows(grid [][]string) []string {
	out := make([]string, len(grid))
	for y, row := range grid {
		s := ""
		for _, c := range row {
			s += c
		}
		out[y] = s
	}
	return out
}
