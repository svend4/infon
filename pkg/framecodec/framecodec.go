// Package framecodec is a per-frame adaptive codec for byte buffers (e.g. encoded
// terminal frames): for each frame it emits the SMALLEST of three encodings —
// RAW (the bytes as-is), ZLIB (deflate of the bytes), or DELTA (only the bytes that
// changed since the previous frame) — tagged by a one-byte mode so the decoder knows
// which it is. Static or low-motion content goes out as a tiny delta, busy or
// incompressible content falls back to raw, and anything in between to zlib; the
// stream always pays the minimum. It is the "delta encoding and compression" the
// frame packet path left for later (see internal/network/frame_packet.go), kept as a
// small reusable primitive that measures the wire.
package framecodec

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
)

// Mode tags the encoding of a frame (the first byte of an encoded frame).
const (
	ModeRaw   byte = 0 // the bytes verbatim
	ModeZlib  byte = 1 // zlib/deflate of the bytes
	ModeDelta byte = 2 // only the bytes that changed since prev (same length required)
)

// Encode returns the smallest of RAW / ZLIB / DELTA encodings of cur, given the
// previous frame prev (pass nil on the first frame). The first byte is the Mode.
func Encode(prev, cur []byte) []byte {
	best := make([]byte, 0, len(cur)+1)
	best = append(best, ModeRaw)
	best = append(best, cur...)

	if z := zlibCompress(cur); len(z)+1 < len(best) {
		b := make([]byte, 0, len(z)+1)
		b = append(b, ModeZlib)
		b = append(b, z...)
		best = b
	}
	if len(prev) == len(cur) { // DELTA only makes sense frame-to-frame at equal size
		if d := deltaEncode(prev, cur); len(d)+1 < len(best) {
			b := make([]byte, 0, len(d)+1)
			b = append(b, ModeDelta)
			b = append(b, d...)
			best = b
		}
	}
	return best
}

// Decode reconstructs a frame from its encoding and the previous frame (needed only
// for DELTA; pass nil otherwise).
func Decode(prev, enc []byte) ([]byte, error) {
	if len(enc) == 0 {
		return nil, errors.New("framecodec: empty frame")
	}
	switch enc[0] {
	case ModeRaw:
		return append([]byte(nil), enc[1:]...), nil
	case ModeZlib:
		return zlibDecompress(enc[1:])
	case ModeDelta:
		return deltaDecode(prev, enc[1:])
	default:
		return nil, errors.New("framecodec: unknown mode")
	}
}

// deltaEncode lists the changed bytes as uvarint(count) then count×(uvarint gap,
// value), where gap is the distance from the previous changed index (>=1) — compact
// when changes are sparse.
func deltaEncode(prev, cur []byte) []byte {
	var b []byte
	var tmp [binary.MaxVarintLen64]byte
	changes := 0
	for i := range cur {
		if cur[i] != prev[i] {
			changes++
		}
	}
	n := binary.PutUvarint(tmp[:], uint64(changes))
	b = append(b, tmp[:n]...)
	last := -1
	for i := range cur {
		if cur[i] != prev[i] {
			m := binary.PutUvarint(tmp[:], uint64(i-last))
			b = append(b, tmp[:m]...)
			b = append(b, cur[i])
			last = i
		}
	}
	return b
}

func deltaDecode(prev, d []byte) ([]byte, error) {
	out := append([]byte(nil), prev...)
	r := bytes.NewReader(d)
	count, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	pos := -1
	for k := uint64(0); k < count; k++ {
		gap, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, err
		}
		val, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		pos += int(gap)
		if pos < 0 || pos >= len(out) {
			return nil, errors.New("framecodec: delta index out of range")
		}
		out[pos] = val
	}
	return out, nil
}

func zlibCompress(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

func zlibDecompress(b []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

// Stream is the sender side: it keeps the previous frame and tallies the wire cost,
// so a caller can stream frames and see how little crosses the wire.
type Stream struct {
	prev                    []byte
	Frames, Bytes, RawBytes int // frames sent, coded bytes, would-be raw bytes
	Raw, Zlib, Delta        int // how many frames used each mode
}

// Push encodes the next frame, updates the tallies, and returns the bytes to send.
func (s *Stream) Push(frame []byte) []byte {
	enc := Encode(s.prev, frame)
	s.prev = append(s.prev[:0:0], frame...)
	s.Frames++
	s.Bytes += len(enc)
	s.RawBytes += len(frame)
	switch enc[0] {
	case ModeRaw:
		s.Raw++
	case ModeZlib:
		s.Zlib++
	case ModeDelta:
		s.Delta++
	}
	return enc
}

// Savings is the fraction of raw bytes saved on the wire (0..1).
func (s *Stream) Savings() float64 {
	if s.RawBytes == 0 {
		return 0
	}
	return 1 - float64(s.Bytes)/float64(s.RawBytes)
}

// Decoder is the receiver side: it keeps the previous frame so DELTA frames resolve.
type Decoder struct{ prev []byte }

// Push decodes the next frame and remembers it for the following DELTA.
func (d *Decoder) Push(enc []byte) ([]byte, error) {
	out, err := Decode(d.prev, enc)
	if err != nil {
		return nil, err
	}
	d.prev = append(d.prev[:0:0], out...)
	return out, nil
}
