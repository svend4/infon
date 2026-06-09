package raytrace

import (
	"image"
	"testing"
)

func TestMovingSphereCenterAtTime(t *testing.T) {
	m := MovingSphere{C0: Vec3{0, 0, 0}, C1: Vec3{0, 10, 0}, Radius: 1}
	cases := []struct {
		t    float64
		want Vec3
	}{
		{0, Vec3{0, 0, 0}},
		{0.5, Vec3{0, 5, 0}},
		{1, Vec3{0, 10, 0}},
	}
	for _, c := range cases {
		if got := m.center(c.t); got.Sub(c.want).Len() > 1e-9 {
			t.Errorf("center(%.1f) = %v, want %v", c.t, got, c.want)
		}
	}
}

// A moving sphere must be hit only when the ray is sampled at the time the sphere
// is actually in the ray's path.
func TestMovingSphereIntersectFollowsTime(t *testing.T) {
	m := MovingSphere{C0: Vec3{0, 0, 0}, C1: Vec3{0, 10, 0}, Radius: 1}
	lo := Ray{Origin: Vec3{0, 0, -5}, Dir: Vec3{0, 0, 1}, Time: 0}  // aimed at C0
	hi := Ray{Origin: Vec3{0, 10, -5}, Dir: Vec3{0, 0, 1}, Time: 1} // aimed at C1
	if _, ok := m.Intersect(lo, geomEps, tFar); !ok {
		t.Error("should hit at time 0 (sphere at C0)")
	}
	if _, ok := m.Intersect(Ray{Origin: lo.Origin, Dir: lo.Dir, Time: 1}, geomEps, tFar); ok {
		t.Error("same ray at time 1 should miss (sphere moved away)")
	}
	if _, ok := m.Intersect(hi, geomEps, tFar); !ok {
		t.Error("should hit at time 1 (sphere at C1)")
	}
	if _, ok := m.Intersect(Ray{Origin: hi.Origin, Dir: hi.Dir, Time: 0}, geomEps, tFar); ok {
		t.Error("same ray at time 0 should miss (sphere not there yet)")
	}
}

// The scene-BVH bounds of a moving sphere must enclose the whole sweep so the
// object is never culled at any time.
func TestMovingSphereBoundsCoverMotion(t *testing.T) {
	m := MovingSphere{C0: Vec3{0, 0, 0}, C1: Vec3{0, 10, 0}, Radius: 1}
	box, ok := objectBounds(m)
	if !ok {
		t.Fatal("moving sphere should be bounded")
	}
	if box.min.X > -1 || box.min.Y > -1 || box.min.Z > -1 {
		t.Errorf("min %v should cover C0-r", box.min)
	}
	if box.max.X < 1 || box.max.Y < 11 || box.max.Z < 1 {
		t.Errorf("max %v should cover C1+r", box.max)
	}
}

// horizontal span (in lit columns) of an emissive object on a black background.
func litSpan(img image.Image) int {
	b := img.Bounds()
	minX, maxX := b.Max.X, b.Min.X
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, _, _, _ := img.At(x, y).RGBA(); r>>8 > 10 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
			}
		}
	}
	if maxX < minX {
		return 0
	}
	return maxX - minX + 1
}

// End to end: a sphere sweeping across the frame must light a far wider band of
// pixels than the same sphere held still, because the shutter sampled it at many
// positions.
func TestMotionBlurWidensCoverage(t *testing.T) {
	cam := Camera{Pos: Vec3{0, 0, -8}}
	mat := Material{Emit: Vec3{5, 5, 5}}
	moving := &Scene{Objects: []Object{
		MovingSphere{C0: Vec3{-2, 0, 0}, C1: Vec3{2, 0, 0}, Radius: 0.5, Mat: mat},
	}}
	static := &Scene{Objects: []Object{
		Sphere{Center: Vec3{-2, 0, 0}, Radius: 0.5, Mat: mat},
	}}
	opt := PathOptions{Samples: 100, MaxDepth: 2, Seed: 1}
	mv := litSpan(PathRender(moving, cam, 64, 32, opt))
	st := litSpan(PathRender(static, cam, 64, 32, opt))
	if st == 0 {
		t.Fatal("static emissive sphere should light some pixels")
	}
	if mv <= 2*st {
		t.Errorf("motion blur should smear coverage much wider: moving=%d static=%d", mv, st)
	}
}
