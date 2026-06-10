// stereo.go renders the world in depth: a left/right eye pair and a red-cyan
// anaglyph that fuses them. The eyes are offset along the camera's own horizontal
// axis (a parallel rig), so near things shift more between the eyes than far ones
// — the parallax your brain reads as depth through red/cyan glasses.
package raytrace

import (
	"image"
	"image/color"
	"math"
)

// cameraRight is the camera's horizontal (screen-right) axis, computed exactly as
// the renderer builds its ray basis, so an eye offset is purely horizontal.
func cameraRight(c Camera) Vec3 {
	cp := math.Cos(c.Pitch)
	forward := Vec3{X: cp * math.Sin(c.Yaw), Y: math.Sin(c.Pitch), Z: cp * math.Cos(c.Yaw)}.Norm()
	right := forward.Cross(Vec3{Y: 1})
	if right.LenSq() < geomEps {
		right = Vec3{X: 1}
	}
	return right.Norm()
}

// StereoCameras returns the left and right eye cameras, offset from cam by `sep`
// world units along its horizontal axis (a parallel rig). Their midpoint is cam.
func StereoCameras(cam Camera, sep float64) (left, right Camera) {
	half := cameraRight(cam).Scale(sep / 2)
	left, right = cam, cam
	left.Pos = cam.Pos.Sub(half)
	right.Pos = cam.Pos.Add(half)
	return left, right
}

// Anaglyph fuses a left and right eye into a red-cyan anaglyph: the red channel is
// taken from the left eye, green and blue from the right. Viewed through
// red(left)/cyan(right) glasses, the parallax reads as depth.
func Anaglyph(left, right image.Image) image.Image {
	lb, rb := left.Bounds(), right.Bounds()
	w, h := lb.Dx(), lb.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			lr, _, _, _ := left.At(lb.Min.X+x, lb.Min.Y+y).RGBA()
			_, rg, rbl, _ := right.At(rb.Min.X+x, rb.Min.Y+y).RGBA()
			out.SetRGBA(x, y, color.RGBA{R: uint8(lr >> 8), G: uint8(rg >> 8), B: uint8(rbl >> 8), A: 255})
		}
	}
	return out
}
