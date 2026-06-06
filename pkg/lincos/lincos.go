// Package lincos builds a self-describing "Lingua Cosmica" frame for a tangram
// figure: a run of bytes that teaches a receiver how to read it from first
// principles, in the spirit of Freudenthal's Lincos and CosmicOS. No values are
// shared in advance — not the radix, not the grid size, not the length. The
// frame opens by teaching counting (unary tallies 1, 2, 3), then states its own
// radix, its own height and width, then the payload, then a checksum computed by
// a rule it has effectively just demonstrated. A decoder that knows only the
// universal convention "n ones then a zero means the number n" recovers the
// figure and verifies it — with no external key.
//
// Layout (a bitstream, MSB first):
//
//	tally(1) tally(2) tally(3)   preamble: "this is how we count" (and a magic)
//	tally(radix)                 the encoding base (e.g. 16)
//	tally(H) tally(W)            the grid dimensions
//	H*W fields of ceil(log2 radix) bits   the cells, row-major (the figure's digits)
//	tally(checksum)              checksum = (sum of cells) mod radix
package lincos

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/svend4/infon/pkg/tangram"
)

// sanity bounds so a garbage stream that happens to start plausibly cannot make
// the decoder allocate wildly.
const (
	maxRadix = 1 << 12
	maxDim   = 1 << 12
)

// Lesson is the trace of what a decoder bootstrapped from the bytes alone.
type Lesson struct {
	Radix   int   // encoding base, learned from the stream
	Bits    int   // bits per cell = ceil(log2 radix)
	H, W    int   // grid dimensions, learned from the stream
	Cells   []int // the payload digits, row-major
	Sum     int   // sum of cells
	Check   int   // checksum the frame stated
	OK      bool  // recomputed (Sum mod Radix) == Check
	BitLen  int   // total bits consumed
}

func bitsFor(radix int) int {
	b := 0
	for (1 << b) < radix {
		b++
	}
	return b
}

// digitsOf expands a figure's Number into exactly H*W base-AlphabetSize digits,
// row-major (leading blank cells preserved) — the figure read cell by cell.
func digitsOf(f tangram.Figure) []int {
	total := f.H * f.W
	if total <= 0 {
		return nil
	}
	base := big.NewInt(int64(tangram.AlphabetSize))
	q := new(big.Int).Set(f.Number())
	r := new(big.Int)
	ds := make([]int, total)
	for i := total - 1; i >= 0; i-- {
		q.QuoRem(q, base, r)
		ds[i] = int(r.Int64())
	}
	return ds
}

// Encode produces the self-describing frame for f.
func Encode(f tangram.Figure) []byte {
	radix := tangram.AlphabetSize
	bits := bitsFor(radix)
	cells := digitsOf(f)
	sum := 0
	for _, d := range cells {
		sum += d
	}
	w := &bitWriter{}
	// preamble: teach counting (also a 1-2-3 magic).
	w.tally(1)
	w.tally(2)
	w.tally(3)
	// self-stated parameters.
	w.tally(radix)
	w.tally(f.H)
	w.tally(f.W)
	// payload: the cells.
	for _, d := range cells {
		w.field(d, bits)
	}
	// self-check.
	w.tally(sum % radix)
	return w.bytes()
}

// Decode reads a frame using only the bytes and the universal tally convention.
// It returns the reconstructed figure and the Lesson it bootstrapped.
func Decode(frame []byte) (tangram.Figure, Lesson, error) {
	r := &bitReader{buf: frame}
	var L Lesson
	// preamble must read as 1, 2, 3 — otherwise this is not a Lingua frame.
	for want := 1; want <= 3; want++ {
		got, err := r.tally()
		if err != nil {
			return tangram.Figure{}, L, fmt.Errorf("preamble: %w", err)
		}
		if got != want {
			return tangram.Figure{}, L, fmt.Errorf("preamble mismatch: read %d, want %d", got, want)
		}
	}
	radix, err := r.tally()
	if err != nil {
		return tangram.Figure{}, L, fmt.Errorf("radix: %w", err)
	}
	if radix < 2 || radix > maxRadix {
		return tangram.Figure{}, L, fmt.Errorf("implausible radix %d", radix)
	}
	L.Radix = radix
	L.Bits = bitsFor(radix)
	if L.H, err = r.tally(); err != nil {
		return tangram.Figure{}, L, fmt.Errorf("height: %w", err)
	}
	if L.W, err = r.tally(); err != nil {
		return tangram.Figure{}, L, fmt.Errorf("width: %w", err)
	}
	if L.H < 1 || L.W < 1 || L.H > maxDim || L.W > maxDim {
		return tangram.Figure{}, L, fmt.Errorf("implausible dims %dx%d", L.H, L.W)
	}
	total := L.H * L.W
	L.Cells = make([]int, total)
	for i := 0; i < total; i++ {
		v, err := r.field(L.Bits)
		if err != nil {
			return tangram.Figure{}, L, fmt.Errorf("cell %d: %w", i, err)
		}
		if v >= radix {
			return tangram.Figure{}, L, fmt.Errorf("cell %d digit %d >= radix %d", i, v, radix)
		}
		L.Cells[i] = v
		L.Sum += v
	}
	if L.Check, err = r.tally(); err != nil {
		return tangram.Figure{}, L, fmt.Errorf("checksum: %w", err)
	}
	L.OK = (L.Sum%radix == L.Check)
	L.BitLen = r.pos
	// fold the cells back into a Number and rebuild the figure's shape.
	n := new(big.Int)
	base := big.NewInt(int64(radix))
	for _, d := range L.Cells {
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(d)))
	}
	fig := tangram.FromNumber(n, L.H, L.W)
	if !L.OK {
		return fig, L, errors.New("checksum mismatch: frame was misread or corrupted")
	}
	return fig, L, nil
}

// Explain narrates, line by line, how the frame teaches its own decoding.
func Explain(frame []byte) string {
	_, L, err := Decode(frame)
	var b strings.Builder
	b.WriteString("a frame that needs no key — it teaches its own language:\n")
	fmt.Fprintf(&b, "  1. preamble 1,2,3            learn to count (unary tallies)\n")
	fmt.Fprintf(&b, "  2. radix       = %d           learn the encoding base -> %d bits/cell\n", L.Radix, L.Bits)
	fmt.Fprintf(&b, "  3. grid        = %d x %d        learn the shape's dimensions\n", L.H, L.W)
	fmt.Fprintf(&b, "  4. payload     = %d cells     read the figure, cell by cell\n", len(L.Cells))
	fmt.Fprintf(&b, "  5. checksum    = %d           sum(cells) mod radix = %d  ->  %s\n",
		L.Check, L.Sum%maxOr(L.Radix, 1), okword(L.OK))
	fmt.Fprintf(&b, "  total: %d bits in %d bytes\n", L.BitLen, (L.BitLen+7)/8)
	if err != nil && !L.OK {
		fmt.Fprintf(&b, "  ! %v\n", err)
	}
	return b.String()
}

func okword(ok bool) string {
	if ok {
		return "understood (no external key)"
	}
	return "MISREAD"
}
func maxOr(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
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

// tally writes n ones then a terminating zero (the number n in unary).
func (w *bitWriter) tally(n int) {
	for i := 0; i < n; i++ {
		w.bit(1)
	}
	w.bit(0)
}

// field writes v as a fixed-width big-endian field of the given bit count.
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
		return 0, errors.New("unexpected end of frame")
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
		if n > maxRadix*maxDim {
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
		v = (v << 1) | b
	}
	return v, nil
}
