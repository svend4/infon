// wire.go ships a ray-traced scene as a few dozen bytes over the same Reed-Solomon
// channel the rest of the project uses (pkg/glyphqr), in the spirit of
// pkg/scene3d and the semantic-video codec: transmit the *description* of the 3-D
// world (a camera, a light and a handful of spheres) — not the rendered pixels —
// and let the receiver ray-trace it locally. Floats are quantised into a fixed
// world box; a 12-sphere scene plus camera is on the order of 100 bytes and
// survives bursts up to the ECC budget. This is "meaning, not pixels" applied to
// real 3-D.
package raytrace

import (
	"errors"

	"github.com/svend4/infon/pkg/glyphqr"
)

const wireMagic = 0x52 // 'R'

// SceneWire is the transmittable description of a simple ray-traced scene: a
// camera, a point light and up to 255 spheres.
type SceneWire struct {
	Cam     Camera
	Light   Vec3
	Spheres []Sphere
}

// fixed quantisation scales.
const (
	posScale = 256.0   // position: int16 / 256  -> ±128 with 1/256 precision
	angScale = 10000.0 // angle:    int16 / 10000 -> ±3.2767 rad
	radScale = 1024.0  // radius:   uint16 / 1024 -> 0..64
	fovSpan  = 6.4     // fov byte: b/255 * 6.4 rad
)

func putI16(b *[]byte, v int16)  { *b = append(*b, byte(uint16(v)>>8), byte(v)) }
func putU16(b *[]byte, v uint16) { *b = append(*b, byte(v>>8), byte(v)) }

func qI16(f, scale float64) int16 {
	v := f * scale
	switch {
	case v > 32767:
		return 32767
	case v < -32768:
		return -32768
	default:
		return int16(v)
	}
}

func qU16(f, scale float64) uint16 {
	v := f * scale
	switch {
	case v > 65535:
		return 65535
	case v < 0:
		return 0
	default:
		return uint16(v)
	}
}

func qByte(f float64) byte { return byte(clamp01(f)*255 + 0.5) }

// Encode serialises the scene and protects it with interleaved Reed-Solomon.
func Encode(w SceneWire, ecc int) []byte {
	b := []byte{wireMagic, 1}
	putVec := func(v Vec3) {
		putI16(&b, qI16(v.X, posScale))
		putI16(&b, qI16(v.Y, posScale))
		putI16(&b, qI16(v.Z, posScale))
	}
	putVec(w.Cam.Pos)
	putI16(&b, qI16(w.Cam.Yaw, angScale))
	putI16(&b, qI16(w.Cam.Pitch, angScale))
	b = append(b, qByte(w.Cam.FOV/fovSpan))
	putVec(w.Light)

	n := len(w.Spheres)
	if n > 255 {
		n = 255
	}
	b = append(b, byte(n))
	for i := 0; i < n; i++ {
		s := w.Spheres[i]
		putVec(s.Center)
		putU16(&b, qU16(s.Radius, radScale))
		b = append(b, qByte(s.Mat.Color.X), qByte(s.Mat.Color.Y), qByte(s.Mat.Color.Z), qByte(s.Mat.Reflect))
	}
	return glyphqr.RSEncodeBytes(b, ecc)
}

// rdr is a bounds-checked cursor; once a read runs past the end, err latches and
// subsequent reads return zero, so a corrupt stream yields an error, never a panic.
type rdr struct {
	b   []byte
	p   int
	err bool
}

func (r *rdr) need(n int) bool {
	if r.err || r.p+n > len(r.b) {
		r.err = true
		return false
	}
	return true
}

func (r *rdr) u8() byte {
	if !r.need(1) {
		return 0
	}
	v := r.b[r.p]
	r.p++
	return v
}

func (r *rdr) i16() int16 {
	if !r.need(2) {
		return 0
	}
	v := int16(uint16(r.b[r.p])<<8 | uint16(r.b[r.p+1]))
	r.p += 2
	return v
}

func (r *rdr) u16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := uint16(r.b[r.p])<<8 | uint16(r.b[r.p+1])
	r.p += 2
	return v
}

func (r *rdr) vec(scale float64) Vec3 {
	return Vec3{
		X: float64(r.i16()) / scale,
		Y: float64(r.i16()) / scale,
		Z: float64(r.i16()) / scale,
	}
}

// Decode inverts Encode, correcting bursts up to the ECC budget first.
func Decode(wire []byte, ecc int) (SceneWire, error) {
	data, err := glyphqr.RSDecodeBytes(wire, ecc)
	if err != nil {
		return SceneWire{}, err
	}
	r := &rdr{b: data}
	if r.u8() != wireMagic || r.u8() != 1 {
		return SceneWire{}, errors.New("raytrace: bad scene header")
	}
	var w SceneWire
	w.Cam.Pos = r.vec(posScale)
	w.Cam.Yaw = float64(r.i16()) / angScale
	w.Cam.Pitch = float64(r.i16()) / angScale
	w.Cam.FOV = float64(r.u8()) / 255 * fovSpan
	w.Light = r.vec(posScale)
	n := int(r.u8())
	for i := 0; i < n; i++ {
		var s Sphere
		s.Center = r.vec(posScale)
		s.Radius = float64(r.u16()) / radScale
		s.Mat.Color = Vec3{X: float64(r.u8()) / 255, Y: float64(r.u8()) / 255, Z: float64(r.u8()) / 255}
		s.Mat.Reflect = float64(r.u8()) / 255
		w.Spheres = append(w.Spheres, s)
	}
	if r.err {
		return SceneWire{}, errors.New("raytrace: truncated scene")
	}
	return w, nil
}

// Build turns a decoded wire into a renderable Scene, adding a checkerboard floor
// just under the lowest sphere, a sky gradient and the given ambient term.
func (w SceneWire) Build(ambient float64) *Scene {
	floorY := 0.0
	first := true
	for _, s := range w.Spheres {
		if b := s.Center.Y - s.Radius; first || b < floorY {
			floorY, first = b, false
		}
	}
	objs := []Object{Plane{Y: floorY, Size: 1, C1: Vec3{0.20, 0.20, 0.22}, C2: Vec3{0.80, 0.80, 0.82}}}
	for _, s := range w.Spheres {
		objs = append(objs, s)
	}
	return &Scene{
		Objects:   objs,
		Light:     w.Light,
		LightInt:  1.4,
		Ambient:   ambient,
		SkyTop:    Vec3{0.45, 0.62, 0.95},
		SkyBottom: Vec3{0.92, 0.95, 1.0},
		MaxBounce: 2,
		AttenK:    0.03,
	}
}
