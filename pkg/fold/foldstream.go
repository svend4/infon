package fold

import (
	"errors"
	"math"

	"github.com/svend4/infon/pkg/glyphqr"
	t7 "github.com/svend4/infon/pkg/tangram7"
)

// foldstream.go is path 1 of the pseudo-3D program: tangram -> origami body.
// Instead of extruding a piece to a prism (see Frame), each flat piece is FOLDED
// up about a hinge by a DIHEDRAL ANGLE, turning the flat dissection (the net)
// into a standing relief (the body). The whole animation is a stream of one
// angle-byte per piece per frame, sent as keyframe + deltas over the same
// Reed-Solomon channel as a semantic-video call, so an "origami unfold" travels
// in a few dozen bytes and survives a burst of lost/corrupted bytes.

// FoldMaxAngle maps an angle byte 0..255 onto 0..90 degrees.
const FoldMaxAngle = math.Pi / 2

// FacesFold folds each piece up about a horizontal hinge through its own
// centroid by angles[i] (radians), anchoring the result to the ground plane.
// At angle 0 the piece is flat (the original 2D figure seen top-down in iso);
// as the angle grows it tilts up into the third dimension.
func FacesFold(pieces []t7.Piece, angles []float64) []Face {
	var faces []Face
	for i, p := range pieces {
		a := 0.0
		if i < len(angles) {
			a = angles[i]
		}
		n := len(p.Shape)
		if n < 3 {
			continue
		}
		var cx, cy float64
		for _, q := range p.Shape {
			cx += q.X
			cy += q.Y
		}
		cx /= float64(n)
		cy /= float64(n)
		ca, sa := math.Cos(a), math.Sin(a)
		pts := make([]Pt3, n)
		minz := math.Inf(1)
		for j, q := range p.Shape {
			ny := cy + (q.Y-cy)*ca
			nz := (q.Y - cy) * sa
			pts[j] = Pt3{q.X, ny, nz}
			if nz < minz {
				minz = nz
			}
		}
		for j := range pts {
			pts[j].Z -= minz
		}
		faces = append(faces, Face{Pts: pts, Color: p.Color, Shade: 0.55 + 0.45*ca})
	}
	return faces
}

// FoldAnim is an origami animation as pure data: one angle byte per piece per
// frame. Pixels are derived; this is all that travels.
type FoldAnim struct {
	NPieces int
	Frames  [][]byte
}

// AnglesAt converts frame f to radians for FacesFold.
func (fa FoldAnim) AnglesAt(f int) []float64 {
	out := make([]float64, fa.NPieces)
	if f < 0 || f >= len(fa.Frames) {
		return out
	}
	for i := 0; i < fa.NPieces && i < len(fa.Frames[f]); i++ {
		out[i] = float64(fa.Frames[f][i]) / 255 * FoldMaxAngle
	}
	return out
}

// serialize encodes the animation as keyframe + per-frame deltas (only the
// angle bytes that changed), the compact P-frame form.
func serializeFold(fa FoldAnim) []byte {
	out := []byte{byte(fa.NPieces), byte(len(fa.Frames) >> 8), byte(len(fa.Frames))}
	if len(fa.Frames) == 0 {
		return out
	}
	out = append(out, fa.Frames[0]...)
	prev := fa.Frames[0]
	for k := 1; k < len(fa.Frames); k++ {
		cur := fa.Frames[k]
		var deltas []byte
		for i := 0; i < fa.NPieces && i < len(cur); i++ {
			if cur[i] != prev[i] {
				deltas = append(deltas, byte(i), cur[i])
			}
		}
		out = append(out, byte(len(deltas)/2))
		out = append(out, deltas...)
		prev = cur
	}
	return out
}

func deserializeFold(b []byte) (FoldAnim, error) {
	if len(b) < 3 {
		return FoldAnim{}, errors.New("fold: short header")
	}
	np := int(b[0])
	nf := int(b[1])<<8 | int(b[2])
	p := 3
	fa := FoldAnim{NPieces: np}
	if nf == 0 {
		return fa, nil
	}
	if p+np > len(b) {
		return fa, errors.New("fold: short keyframe")
	}
	key := make([]byte, np)
	copy(key, b[p:p+np])
	p += np
	fa.Frames = append(fa.Frames, key)
	prev := key
	for k := 1; k < nf; k++ {
		if p >= len(b) {
			return fa, errors.New("fold: short delta count")
		}
		c := int(b[p])
		p++
		cur := make([]byte, np)
		copy(cur, prev)
		for d := 0; d < c; d++ {
			if p+1 >= len(b) {
				return fa, errors.New("fold: short delta")
			}
			idx := int(b[p])
			v := b[p+1]
			p += 2
			if idx < np {
				cur[idx] = v
			}
		}
		fa.Frames = append(fa.Frames, cur)
		prev = cur
	}
	return fa, nil
}

// EncodeFoldStream serialises the animation and protects it with interleaved
// Reed-Solomon (the same byte-stream FEC a semantic video call uses).
func EncodeFoldStream(fa FoldAnim, ecc int) []byte {
	return glyphqr.RSEncodeBytes(serializeFold(fa), ecc)
}

// DecodeFoldStream inverts EncodeFoldStream, correcting up to ecc/2 byte errors
// per block before deserialising.
func DecodeFoldStream(wire []byte, ecc int) (FoldAnim, error) {
	data, err := glyphqr.RSDecodeBytes(wire, ecc)
	if err != nil {
		return FoldAnim{}, err
	}
	return deserializeFold(data)
}

// RawSize is the serialised (pre-RS) byte size of the animation.
func RawSize(fa FoldAnim) int { return len(serializeFold(fa)) }
