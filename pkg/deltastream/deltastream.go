// Package deltastream is the synthesis primitive shared by the three pseudo-3D
// paths: a generic keyframe + byte-delta animation codec over interleaved
// Reed-Solomon. Every frame is a fixed-width byte record (a packed spec); the
// stream stores frame 0 in full and, for every later frame, only the byte
// indices that changed. fold.FoldAnim (width = pieces) and scene3d.SceneAnim
// (width = items*6) are both instances of this - one codec, three paths.
package deltastream

import (
	"errors"

	"github.com/svend4/infon/pkg/glyphqr"
)

// Anim is a sequence of equal-width byte frames.
type Anim struct {
	Width  int
	Frames [][]byte
}

func serialize(a Anim) []byte {
	out := []byte{byte(a.Width >> 8), byte(a.Width), byte(len(a.Frames) >> 8), byte(len(a.Frames))}
	if len(a.Frames) == 0 {
		return out
	}
	out = append(out, a.Frames[0]...)
	prev := a.Frames[0]
	for k := 1; k < len(a.Frames); k++ {
		cur := a.Frames[k]
		var deltas []byte
		for i := 0; i < a.Width && i < len(cur); i++ {
			if cur[i] != prev[i] {
				deltas = append(deltas, byte(i>>8), byte(i), cur[i])
			}
		}
		out = append(out, byte((len(deltas)/3)>>8), byte(len(deltas)/3))
		out = append(out, deltas...)
		prev = cur
	}
	return out
}

// maxDecodedBytes bounds the frames a decode materialises (width x frame count).
// A frame that repeats its predecessor costs two bytes on the wire, so the wire
// size alone does not bound it: a header may declare 65535 frames of 65535 bytes,
// about 4 GiB, behind a stream of a few hundred kilobytes.
const maxDecodedBytes = 64 << 20

func deserialize(b []byte, check func(width, frames int) error) (Anim, error) {
	if len(b) < 4 {
		return Anim{}, errors.New("deltastream: short header")
	}
	w := int(b[0])<<8 | int(b[1])
	nf := int(b[2])<<8 | int(b[3])
	p := 4
	a := Anim{Width: w}
	if w*nf > maxDecodedBytes {
		return a, errors.New("deltastream: animation too large")
	}
	if check != nil {
		if err := check(w, nf); err != nil {
			return a, err
		}
	}
	if nf == 0 {
		return a, nil
	}
	if p+w > len(b) {
		return a, errors.New("deltastream: short keyframe")
	}
	key := make([]byte, w)
	copy(key, b[p:p+w])
	p += w
	a.Frames = append(a.Frames, key)
	prev := key
	for k := 1; k < nf; k++ {
		if p+2 > len(b) {
			return a, errors.New("deltastream: short count")
		}
		c := int(b[p])<<8 | int(b[p+1])
		p += 2
		cur := make([]byte, w)
		copy(cur, prev)
		for d := 0; d < c; d++ {
			if p+3 > len(b) {
				return a, errors.New("deltastream: short delta")
			}
			idx := int(b[p])<<8 | int(b[p+1])
			v := b[p+2]
			p += 3
			if idx < w {
				cur[idx] = v
			}
		}
		a.Frames = append(a.Frames, cur)
		prev = cur
	}
	return a, nil
}

// Encode serialises the animation (keyframe + deltas) and protects it with
// interleaved Reed-Solomon.
func Encode(a Anim, ecc int) []byte { return glyphqr.RSEncodeBytes(serialize(a), ecc) }

// Decode inverts Encode, correcting up to ecc/2 byte errors per block. A stream
// declaring more than 64 MiB of frames is rejected before any frame is built.
func Decode(wire []byte, ecc int) (Anim, error) { return DecodeChecked(wire, ecc, nil) }

// DecodeChecked is Decode with a caller check on the declared frame width and
// frame count, run before any frame is built; an error from check aborts the decode.
func DecodeChecked(wire []byte, ecc int, check func(width, frames int) error) (Anim, error) {
	data, err := glyphqr.RSDecodeBytes(wire, ecc)
	if err != nil {
		return Anim{}, err
	}
	return deserialize(data, check)
}

// RawSize is the serialised (pre-RS) size in bytes.
func RawSize(a Anim) int { return len(serialize(a)) }
