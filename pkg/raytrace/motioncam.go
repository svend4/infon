// motioncam.go adds camera motion blur: the camera pose is interpolated across
// the shutter window (open -> close), so a panning, dollying, zooming or
// racking-focus camera streaks the frame the way a real exposure does. It reuses
// the same Ray.Time machinery as object motion blur (MovingSphere): each path
// samples one shutter time, the camera pose is evaluated there, and that time is
// stamped on the ray so moving objects stay consistent with the moving camera.
package raytrace

import "image"

// lerpCamera linearly interpolates two camera poses at shutter fraction t in
// [0,1]. Every field (position, orientation, field of view, aperture, focus)
// interpolates, so a single open/close pair expresses pans, dollies, zooms and
// focus pulls.
func lerpCamera(a, b Camera, t float64) Camera {
	lf := func(x, y float64) float64 { return x + (y-x)*t }
	return Camera{
		Pos:      a.Pos.Add(b.Pos.Sub(a.Pos).Scale(t)),
		Yaw:      lf(a.Yaw, b.Yaw),
		Pitch:    lf(a.Pitch, b.Pitch),
		FOV:      lf(a.FOV, b.FOV),
		Aperture: lf(a.Aperture, b.Aperture),
		Focus:    lf(a.Focus, b.Focus),
	}
}

// pathSumMotion is pathSum with a camera that moves over the shutter. Per sample
// it draws a shutter time, interpolates the camera to that time, and builds the
// primary ray from the interpolated pose (basis recomputed per sample, so it is
// pricier than the static path and worth using only when the camera moves).
func pathSumMotion(s *Scene, open, close Camera, pxW, pxH int, opt PathOptions) []Vec3 {
	return pathSumRays(s, pxW, pxH, opt, func(u, v float64, rg *rng) Ray {
		t := rg.f()
		c := lerpCamera(open, close, t)
		return c.basis(pxW, pxH).lensRayAt(u, v, t, c, rg)
	})
}

// PathRenderMotion renders with camera motion blur between two poses (shutter
// open and close). With open == close it matches PathRender.
func PathRenderMotion(s *Scene, open, close Camera, pxW, pxH int, opt PathOptions) image.Image {
	if pxW <= 0 || pxH <= 0 {
		return image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	}
	return tonemapSum(pathSumMotion(s, open, close, pxW, pxH, opt), pxW, pxH, opt.Samples)
}
