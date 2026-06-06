// Package arecibo builds the FULL Lingua-Cosmica frame: a single message that
// carries not only a figure and a self-check, but the ALPHABET ITSELF — an
// in-frame legend giving each symbol's pixel shape (sub-cell coverage bitmap).
// A receiver that shares no culture learns to count from the preamble, learns
// what every symbol looks like from the legend, reads the figure, and verifies
// it — then can even RENDER the figure using only what the frame taught. This is
// pkg/lincos taken to its Arecibo/CosmicOS conclusion.
package arecibo

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/tangram"
)

// Message is everything a decoder bootstrapped from the bytes alone.
type Message struct {
	Radix  int      // encoding base (learned)
	Bits   int      // bits per cell
	S      int      // sub-cell resolution of the legend bitmaps
	H, W   int      // figure grid
	Legend [][]bool // Legend[d] is the S*S coverage bitmap for symbol d (row-major)
	Cells  []int    // the figure, row-major
	Sum    int
	Check  int
	OK     bool
	BitLen int
}

func bitsFor(radix int) int {
	b := 0
	for (1 << b) < radix {
		b++
	}
	return b
}

func digitsOf(f tangram.Figure) []int {
	total := f.H * f.W
	ds := make([]int, total)
	base := big.NewInt(int64(tangram.AlphabetSize))
	q := new(big.Int).Set(f.Number())
	r := new(big.Int)
	for i := total - 1; i >= 0; i-- {
		q.QuoRem(q, base, r)
		ds[i] = int(r.Int64())
	}
	return ds
}

// Encode emits the self-describing frame for f, including the alphabet legend.
func Encode(f tangram.Figure) []byte {
	radix := tangram.AlphabetSize
	S := glyphset.Sub
	bits := bitsFor(radix)
	cells := digitsOf(f)
	sum := 0
	for _, d := range cells {
		sum += d
	}
	w := &bitWriter{}
	w.tally(1)
	w.tally(2)
	w.tally(3) // learn to count
	w.tally(radix)
	w.tally(S)
	// legend: every symbol's shape, so the alphabet need not be shared.
	for d := 0; d < radix; d++ {
		name := tangram.GlyphForDigit(d)
		for sy := 0; sy < S; sy++ {
			for sx := 0; sx < S; sx++ {
				if glyphset.Coverage(name, sx, sy, S) {
					w.bit(1)
				} else {
					w.bit(0)
				}
			}
		}
	}
	w.tally(f.H)
	w.tally(f.W)
	for _, d := range cells {
		w.field(d, bits)
	}
	w.tally(sum % radix)
	return w.bytes()
}

// Decode reconstructs the message (alphabet legend included) and the figure using
// only the bytes — no shared alphabet, no external key.
func Decode(frame []byte) (Message, tangram.Figure, error) {
	r := &bitReader{buf: frame}
	var m Message
	for want := 1; want <= 3; want++ {
		g, err := r.tally()
		if err != nil || g != want {
			return m, tangram.Figure{}, fmt.Errorf("arecibo: preamble (read %d, want %d)", g, want)
		}
	}
	radix, err := r.tally()
	if err != nil || radix < 2 || radix > 4096 {
		return m, tangram.Figure{}, errors.New("arecibo: implausible radix")
	}
	m.Radix, m.Bits = radix, bitsFor(radix)
	S, err := r.tally()
	if err != nil || S < 1 || S > 64 {
		return m, tangram.Figure{}, errors.New("arecibo: implausible sub-cell size")
	}
	m.S = S
	m.Legend = make([][]bool, radix)
	for d := 0; d < radix; d++ {
		bm := make([]bool, S*S)
		for i := range bm {
			b, err := r.bit()
			if err != nil {
				return m, tangram.Figure{}, errors.New("arecibo: truncated legend")
			}
			bm[i] = b == 1
		}
		m.Legend[d] = bm
	}
	if m.H, err = r.tally(); err != nil {
		return m, tangram.Figure{}, errors.New("arecibo: height")
	}
	if m.W, err = r.tally(); err != nil {
		return m, tangram.Figure{}, errors.New("arecibo: width")
	}
	if m.H < 1 || m.W < 1 || m.H > 4096 || m.W > 4096 {
		return m, tangram.Figure{}, errors.New("arecibo: implausible dims")
	}
	total := m.H * m.W
	m.Cells = make([]int, total)
	for i := 0; i < total; i++ {
		v, err := r.field(m.Bits)
		if err != nil || v >= radix {
			return m, tangram.Figure{}, fmt.Errorf("arecibo: cell %d", i)
		}
		m.Cells[i] = v
		m.Sum += v
	}
	if m.Check, err = r.tally(); err != nil {
		return m, tangram.Figure{}, errors.New("arecibo: checksum")
	}
	m.OK = m.Sum%radix == m.Check
	m.BitLen = r.pos
	n := new(big.Int)
	base := big.NewInt(int64(radix))
	for _, d := range m.Cells {
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(d)))
	}
	fig := tangram.FromNumber(n, m.H, m.W)
	if !m.OK {
		return m, fig, errors.New("arecibo: checksum mismatch (corrupted)")
	}
	return m, fig, nil
}

// LegendBit reports symbol d's coverage at (sx,sy) from a decoded message — the
// alphabet as LEARNED from the frame, with no appeal to glyphset.
func (m Message) LegendBit(d, sx, sy int) bool {
	if d < 0 || d >= len(m.Legend) {
		return false
	}
	i := sy*m.S + sx
	return i >= 0 && i < len(m.Legend[d]) && m.Legend[d][i]
}

// ---- bit IO ----------------------------------------------------------------

type bitWriter struct {
	buf   []byte
	nbits int
}

func (w *bitWriter) bit(b int) {
	idx := w.nbits / 8
	for idx >= len(w.buf) {
		w.buf = append(w.buf, 0)
	}
	if b&1 != 0 {
		w.buf[idx] |= 1 << uint(7-w.nbits%8)
	}
	w.nbits++
}
func (w *bitWriter) tally(n int) {
	for i := 0; i < n; i++ {
		w.bit(1)
	}
	w.bit(0)
}
func (w *bitWriter) field(v, bits int) {
	for i := bits - 1; i >= 0; i-- {
		w.bit((v >> uint(i)) & 1)
	}
}
func (w *bitWriter) bytes() []byte { return w.buf }

type bitReader struct {
	buf []byte
	pos int
}

func (r *bitReader) bit() (int, error) {
	if r.pos >= len(r.buf)*8 {
		return 0, errors.New("eof")
	}
	b := int(r.buf[r.pos/8]>>uint(7-r.pos%8)) & 1
	r.pos++
	return b, nil
}
func (r *bitReader) tally() (int, error) {
	n := 0
	for {
		b, err := r.bit()
		if err != nil {
			return 0, err
		}
		if b == 0 {
			return n, nil
		}
		n++
		if n > 1<<16 {
			return 0, errors.New("tally overflow")
		}
	}
}
func (r *bitReader) field(bits int) (int, error) {
	v := 0
	for i := 0; i < bits; i++ {
		b, err := r.bit()
		if err != nil {
			return 0, err
		}
		v = v<<1 | b
	}
	return v, nil
}
