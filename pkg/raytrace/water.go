// water.go adds an animated water surface: a horizontal, reflective plane whose
// shading normal ripples as a sum of moving directional waves (a cheap Gerstner-
// style field). The geometry stays flat, but the rippling normal makes the sky and
// scene shimmer in the reflection, and the surface advances with a shared clock so
// every peer sees the same swell — motion as a formula, not frames. Clean-room.
package raytrace

import "math"

// Water is an infinite reflective water plane at height Y. Time advances the waves;
// Scale is their spatial size and Amp their slope strength.
type Water struct {
	Y       float64
	Color   Vec3    // deep-water tint
	Reflect float64 // mirror fraction (default 0.5)
	Time    float64 // wave phase (seconds)
	Amp     float64 // wave slope amplitude (default 0.3)
	Scale   float64 // wave spatial scale (default 2)
}

// waterWaves are the directional wave components (k direction, speed, weight).
var waterWaves = [...]struct{ kx, kz, sp, w float64 }{
	{1.0, 0.25, 1.0, 1.0}, {-0.4, 1.0, 1.3, 0.6}, {0.8, -0.7, 0.8, 0.45}, {-0.9, -0.5, 1.6, 0.3},
}

// normal returns the rippled surface normal at world point p.
func (w Water) normal(p Vec3) Vec3 {
	s := w.Scale
	if s <= 0 {
		s = 2
	}
	a := w.Amp
	if a <= 0 {
		a = 0.3
	}
	var dx, dz float64
	for _, wv := range waterWaves {
		ph := (wv.kx*p.X+wv.kz*p.Z)/s - w.Time*wv.sp
		c := math.Cos(ph) * a * wv.w
		dx += wv.kx / s * c
		dz += wv.kz / s * c
	}
	return Vec3{X: -dx, Y: 1, Z: -dz}.Norm()
}

// Intersect implements Object: the ray meets the y=Y plane, with a rippling normal
// and a reflective, tinted material.
func (w Water) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	if math.Abs(r.Dir.Y) < geomEps {
		return Hit{}, false
	}
	t := (w.Y - r.Origin.Y) / r.Dir.Y
	if t <= tMin || t > tMax {
		return Hit{}, false
	}
	p := r.At(t)
	n, front := orient(w.normal(p), r.Dir)
	refl := w.Reflect
	if refl == 0 {
		refl = 0.5
	}
	col := w.Color
	if col.LenSq() == 0 {
		col = Vec3{X: 0.1, Y: 0.3, Z: 0.45}
	}
	return Hit{T: t, P: p, N: n, Front: front, Mat: Material{Color: col, Reflect: refl}}, true
}
