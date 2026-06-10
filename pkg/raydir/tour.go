package raydir

import (
	"math"
	"sort"

	"github.com/svend4/infon/pkg/raytrace"
)

// tour.go flies the camera on rails. Given a set of waypoints — the world's
// landmarks, say — it threads a smooth Catmull-Rom spline through them and walks a
// camera along it at a steady pace (reparameterised by arc length), always looking
// the way it's travelling. That turns a grown world into a cinematic fly-through: a
// film of the place, not a jittery hand-walk. Pure geometry, so it's deterministic
// and testable.

// CatmullRom interpolates the centre segment p1→p2 (with neighbours p0,p3 setting
// the tangents) at t in [0,1]. The curve passes through p1 at t=0 and p2 at t=1.
func CatmullRom(p0, p1, p2, p3 raytrace.Vec3, t float64) raytrace.Vec3 {
	t2 := t * t
	t3 := t2 * t
	a := p1.Scale(2)
	b := p2.Sub(p0).Scale(t)
	c := p0.Scale(2).Sub(p1.Scale(5)).Add(p2.Scale(4)).Sub(p3).Scale(t2)
	d := p1.Scale(3).Sub(p0).Sub(p2.Scale(3)).Add(p3).Scale(t3)
	return a.Add(b).Add(c).Add(d).Scale(0.5)
}

// Tour is a camera fly-through along a spline through waypoints.
type Tour struct {
	Points []raytrace.Vec3
	FOV    float64
	cumLen []float64 // arc-length LUT (cumulative distance at each sample)
	param  []float64 // the raw spline parameter at each sample
	total  float64
}

const tourSamples = 256

// NewTour builds a tour through the given waypoints (>=1).
func NewTour(points []raytrace.Vec3, fov float64) *Tour {
	if fov <= 0 {
		fov = math.Pi / 3
	}
	t := &Tour{Points: points, FOV: fov}
	t.buildArcTable()
	return t
}

// TourFromLandmarks builds a tour through the world's landmarks (in index order),
// at camera height `height`.
func TourFromLandmarks(marks []Landmark, height, fov float64) *Tour {
	ms := append([]Landmark(nil), marks...)
	sort.Slice(ms, func(i, j int) bool { return ms[i].Index < ms[j].Index })
	pts := make([]raytrace.Vec3, 0, len(ms))
	for _, m := range ms {
		pts = append(pts, raytrace.Vec3{X: m.At.X, Y: height, Z: m.At.Z})
	}
	return NewTour(pts, fov)
}

// rawPosAt evaluates the spline at raw parameter s in [0,1] (uniform over segments).
func (t *Tour) rawPosAt(s float64) raytrace.Vec3 {
	n := len(t.Points)
	if n == 0 {
		return raytrace.Vec3{}
	}
	if n == 1 {
		return t.Points[0]
	}
	segs := n - 1
	fs := clampf(s, 0, 1) * float64(segs)
	i := int(fs)
	if i >= segs {
		i = segs - 1
	}
	lt := fs - float64(i)
	idx := func(k int) int {
		if k < 0 {
			return 0
		}
		if k > n-1 {
			return n - 1
		}
		return k
	}
	return CatmullRom(t.Points[idx(i-1)], t.Points[idx(i)], t.Points[idx(i+1)], t.Points[idx(i+2)], lt)
}

// buildArcTable samples the spline and records cumulative arc length, so a tour
// parameter can be mapped to a constant-speed position.
func (t *Tour) buildArcTable() {
	t.cumLen = make([]float64, tourSamples+1)
	t.param = make([]float64, tourSamples+1)
	prev := t.rawPosAt(0)
	for k := 0; k <= tourSamples; k++ {
		s := float64(k) / tourSamples
		p := t.rawPosAt(s)
		if k > 0 {
			t.total += p.Sub(prev).Len()
		}
		t.cumLen[k] = t.total
		t.param[k] = s
		prev = p
	}
}

// paramForArc maps u in [0,1] (fraction of the total length) to the raw parameter.
func (t *Tour) paramForArc(u float64) float64 {
	if t.total == 0 {
		return clampf(u, 0, 1)
	}
	target := clampf(u, 0, 1) * t.total
	k := sort.SearchFloat64s(t.cumLen, target)
	if k <= 0 {
		return t.param[0]
	}
	if k >= len(t.cumLen) {
		return t.param[len(t.param)-1]
	}
	d0, d1 := t.cumLen[k-1], t.cumLen[k]
	frac := 0.0
	if d1 > d0 {
		frac = (target - d0) / (d1 - d0)
	}
	return t.param[k-1] + (t.param[k]-t.param[k-1])*frac
}

// PosAt is the constant-speed position at u in [0,1] along the whole tour.
func (t *Tour) PosAt(u float64) raytrace.Vec3 { return t.rawPosAt(t.paramForArc(u)) }

// CameraAt is the camera at u in [0,1]: positioned on the spline, looking the way
// it's moving.
func (t *Tour) CameraAt(u float64) raytrace.Camera {
	pos := t.PosAt(u)
	const du = 0.01
	var dir raytrace.Vec3
	if u+du <= 1 {
		dir = t.PosAt(u + du).Sub(pos)
	} else {
		dir = pos.Sub(t.PosAt(u - du))
	}
	if dir.LenSq() < 1e-9 {
		dir = raytrace.Vec3{Z: 1}
	}
	dir = dir.Norm()
	yaw := math.Atan2(dir.X, dir.Z)
	pitch := math.Asin(clampf(dir.Y, -1, 1))
	return raytrace.Camera{Pos: pos, Yaw: yaw, Pitch: pitch, FOV: t.FOV}
}
