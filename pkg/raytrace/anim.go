// anim.go streams a *moving* ray-traced scene over the project's shared
// keyframe+byte-delta Reed-Solomon codec (pkg/deltastream, the same one behind
// fold.FoldAnim and scene3d.SceneAnim). Each frame is a fixed-width record
// (camera + light + N spheres); only the bytes that change between frames are
// transmitted, so an orbiting or animated scene costs a handful of bytes per
// tick and tolerates packet loss.
package raytrace

import (
	"errors"

	"github.com/svend4/infon/pkg/deltastream"
)

// SceneAnim is a sequence of scenes that all share the same sphere count N.
type SceneAnim struct {
	N      int
	Frames []SceneWire
}

// per-frame byte layout: cam pos(6)+yaw/pitch(4)+fov(1)=11, light(6),
// then N * (center(6)+radius(2)+color(3)+reflect(1)=12).
func frameWidth(n int) int { return 11 + 6 + 12*n }

func packFrame(w SceneWire, n int) []byte {
	b := make([]byte, 0, frameWidth(n))
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
	for i := 0; i < n; i++ {
		var s Sphere
		if i < len(w.Spheres) {
			s = w.Spheres[i]
		}
		putVec(s.Center)
		putU16(&b, qU16(s.Radius, radScale))
		b = append(b, qByte(s.Mat.Color.X), qByte(s.Mat.Color.Y), qByte(s.Mat.Color.Z), qByte(s.Mat.Reflect))
	}
	return b
}

func unpackFrame(b []byte, n int) SceneWire {
	r := &rdr{b: b}
	var w SceneWire
	w.Cam.Pos = r.vec(posScale)
	w.Cam.Yaw = float64(r.i16()) / angScale
	w.Cam.Pitch = float64(r.i16()) / angScale
	w.Cam.FOV = float64(r.u8()) / 255 * fovSpan
	w.Light = r.vec(posScale)
	for i := 0; i < n; i++ {
		var s Sphere
		s.Center = r.vec(posScale)
		s.Radius = float64(r.u16()) / radScale
		s.Mat.Color = Vec3{X: float64(r.u8()) / 255, Y: float64(r.u8()) / 255, Z: float64(r.u8()) / 255}
		s.Mat.Reflect = float64(r.u8()) / 255
		w.Spheres = append(w.Spheres, s)
	}
	return w
}

// EncodeAnim serialises the animation (keyframe + per-byte deltas) over RS.
func EncodeAnim(a SceneAnim, ecc int) []byte {
	da := deltastream.Anim{Width: frameWidth(a.N)}
	for _, f := range a.Frames {
		da.Frames = append(da.Frames, packFrame(f, a.N))
	}
	return deltastream.Encode(da, ecc)
}

// DecodeAnim inverts EncodeAnim, correcting bursts up to the ECC budget.
func DecodeAnim(wire []byte, ecc int) (SceneAnim, error) {
	da, err := deltastream.Decode(wire, ecc)
	if err != nil {
		return SceneAnim{}, err
	}
	n := (da.Width - 17) / 12
	if n < 0 || frameWidth(n) != da.Width {
		return SceneAnim{}, errors.New("raytrace: bad animation frame width")
	}
	a := SceneAnim{N: n}
	for _, fb := range da.Frames {
		a.Frames = append(a.Frames, unpackFrame(fb, n))
	}
	return a, nil
}

// AnimRawSize is the serialised (pre-RS) byte size, handy for "bytes per frame".
func AnimRawSize(a SceneAnim) int {
	da := deltastream.Anim{Width: frameWidth(a.N)}
	for _, f := range a.Frames {
		da.Frames = append(da.Frames, packFrame(f, a.N))
	}
	return deltastream.RawSize(da)
}
