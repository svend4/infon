// Package scene3d is path 3 of the pseudo-3D program: a tiny 3-D scene graph.
// A Scene is a list of placed tangram7 pieces - each Item is {piece, x, y, z,
// yaw (45-degree steps), colour}, six bytes. Items are extruded to slabs and
// drawn with the path-1 isometric renderer (fold.Render - the second cross-path
// overlap). A SceneAnim is a sequence of scenes serialised as keyframe + deltas
// and shipped over the same Reed-Solomon byte-stream as a semantic video call,
// so an animated 3-D scene - "a real game" - travels in a few dozen bytes per
// frame and survives loss.
package scene3d

import (
	"errors"
	"image"
	"math"

	"github.com/svend4/infon/pkg/fold"
	"github.com/svend4/infon/pkg/glyphqr"
	t7 "github.com/svend4/infon/pkg/tangram7"
)

// World scales: a position byte 0..255 maps onto these spans.
const (
	Span      = 14.0 // x,y extent
	ZSpan     = 6.0  // z (stack) extent
	Thickness = 0.7  // slab thickness
)

var paletteNames = []string{
	"blue", "teal", "gold", "red", "orange", "green", "pink",
	"amber", "navy", "coral", "mint", "purple",
}

// ColorName maps a colour byte to a palette name.
func ColorName(b uint8) string { return paletteNames[int(b)%len(paletteNames)] }

// Item is one placed piece: which tangram7 piece, its x/y/z position bytes, a
// yaw in 45-degree steps and a colour byte. Six bytes on the wire.
type Item struct{ Piece, X, Y, Z, Yaw, Color uint8 }

func (it Item) pack() []byte { return []byte{it.Piece, it.X, it.Y, it.Z, it.Yaw, it.Color} }
func unpack(b []byte) Item {
	return Item{b[0], b[1], b[2], b[3], b[4], b[5]}
}

// pieceModel returns piece id's outline centred on its centroid (local frame).
func pieceModel(id int) [][2]float64 {
	pcs := t7.Square()
	p := pcs[id%len(pcs)]
	var cx, cy float64
	for _, q := range p.Shape {
		cx += q.X
		cy += q.Y
	}
	n := float64(len(p.Shape))
	cx /= n
	cy /= n
	out := make([][2]float64, len(p.Shape))
	for i, q := range p.Shape {
		out[i] = [2]float64{q.X - cx, q.Y - cy}
	}
	return out
}

// ItemFaces extrudes one item into an isometric slab (top + shaded walls).
func ItemFaces(it Item) []fold.Face {
	poly := pieceModel(int(it.Piece))
	a := float64(int(it.Yaw)%8) * math.Pi / 4
	ca, sa := math.Cos(a), math.Sin(a)
	wx := float64(it.X) / 255 * Span
	wy := float64(it.Y) / 255 * Span
	wz := float64(it.Z) / 255 * ZSpan
	n := len(poly)
	ground := make([][2]float64, n)
	top := make([]fold.Pt3, n)
	var cx, cy float64
	for i, p := range poly {
		rx := p[0]*ca - p[1]*sa
		ry := p[0]*sa + p[1]*ca
		gx, gy := wx+rx, wy+ry
		ground[i] = [2]float64{gx, gy}
		top[i] = fold.Pt3{X: gx, Y: gy, Z: wz + Thickness}
		cx += gx
		cy += gy
	}
	cx /= float64(n)
	cy /= float64(n)
	col := ColorName(it.Color)
	faces := []fold.Face{{Pts: top, Color: col, Shade: 1.0}}
	for i := 0; i < n; i++ {
		g0, g1 := ground[i], ground[(i+1)%n]
		quad := []fold.Pt3{
			{X: g0[0], Y: g0[1], Z: wz}, {X: g1[0], Y: g1[1], Z: wz},
			{X: g1[0], Y: g1[1], Z: wz + Thickness}, {X: g0[0], Y: g0[1], Z: wz + Thickness},
		}
		ex, ey := g1[0]-g0[0], g1[1]-g0[1]
		nx, ny := ey, -ex
		mx, my := (g0[0]+g1[0])/2-cx, (g0[1]+g1[1])/2-cy
		if nx*mx+ny*my < 0 {
			nx, ny = -nx, -ny
		}
		ln := math.Hypot(nx, ny)
		if ln == 0 {
			ln = 1
		}
		d := (nx*(-1) + ny*(-0.4)) / ln
		faces = append(faces, fold.Face{Pts: quad, Color: col, Shade: 0.5 + 0.32*((d+1)/2)})
	}
	return faces
}

// SceneFaces builds the faces for a whole scene.
func SceneFaces(items []Item) []fold.Face {
	var faces []fold.Face
	for _, it := range items {
		faces = append(faces, ItemFaces(it)...)
	}
	return faces
}

// Render draws a scene in isometric projection.
func Render(items []Item, pxW, pxH int) image.Image { return fold.Render(SceneFaces(items), pxW, pxH) }

// SceneAnim is an animated scene as pure data: items per frame.
type SceneAnim struct {
	NItems int
	Frames [][]Item
}

func serialize(a SceneAnim) []byte {
	out := []byte{byte(a.NItems), byte(len(a.Frames) >> 8), byte(len(a.Frames))}
	if len(a.Frames) == 0 {
		return out
	}
	for _, it := range a.Frames[0] {
		out = append(out, it.pack()...)
	}
	prev := a.Frames[0]
	for k := 1; k < len(a.Frames); k++ {
		cur := a.Frames[k]
		var deltas []byte
		for i := 0; i < a.NItems && i < len(cur); i++ {
			if cur[i] != prev[i] {
				deltas = append(deltas, byte(i))
				deltas = append(deltas, cur[i].pack()...)
			}
		}
		out = append(out, byte(len(deltas)/7))
		out = append(out, deltas...)
		prev = cur
	}
	return out
}

func deserialize(b []byte) (SceneAnim, error) {
	if len(b) < 3 {
		return SceneAnim{}, errors.New("scene3d: short header")
	}
	ni := int(b[0])
	nf := int(b[1])<<8 | int(b[2])
	p := 3
	a := SceneAnim{NItems: ni}
	if nf == 0 {
		return a, nil
	}
	if p+ni*6 > len(b) {
		return a, errors.New("scene3d: short keyframe")
	}
	key := make([]Item, ni)
	for i := 0; i < ni; i++ {
		key[i] = unpack(b[p : p+6])
		p += 6
	}
	a.Frames = append(a.Frames, key)
	prev := key
	for k := 1; k < nf; k++ {
		if p >= len(b) {
			return a, errors.New("scene3d: short delta count")
		}
		c := int(b[p])
		p++
		cur := make([]Item, ni)
		copy(cur, prev)
		for d := 0; d < c; d++ {
			if p+7 > len(b) {
				return a, errors.New("scene3d: short delta")
			}
			idx := int(b[p])
			it := unpack(b[p+1 : p+7])
			p += 7
			if idx < ni {
				cur[idx] = it
			}
		}
		a.Frames = append(a.Frames, cur)
		prev = cur
	}
	return a, nil
}

// EncodeSceneStream serialises the animation and protects it with interleaved
// Reed-Solomon (the semantic-video byte channel).
func EncodeSceneStream(a SceneAnim, ecc int) []byte { return glyphqr.RSEncodeBytes(serialize(a), ecc) }

// DecodeSceneStream inverts EncodeSceneStream, correcting bursts up to ecc/2 per
// block before deserialising.
func DecodeSceneStream(wire []byte, ecc int) (SceneAnim, error) {
	data, err := glyphqr.RSDecodeBytes(wire, ecc)
	if err != nil {
		return SceneAnim{}, err
	}
	return deserialize(data)
}

// RawSize is the serialised (pre-RS) byte size.
func RawSize(a SceneAnim) int { return len(serialize(a)) }
