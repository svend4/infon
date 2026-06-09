package raydir

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// daynight.go gives the shared world a time of day. SkyForTime maps a single
// number — the fraction of a day in [0,1) — to a sky gradient and a sun (its
// direction, colour and whether it is up), so dawn, noon, dusk and night all fall
// out of one value. The host advances it and broadcasts EncodeEnv (8 bytes), so
// the whole group sees the same light evolve: a living world for almost nothing on
// the wire — meaning, not pixels.

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func lerpV(a, b raytrace.Vec3, t float64) raytrace.Vec3 { return a.Add(b.Sub(a).Scale(t)) }

// SkyForTime maps time-of-day t in [0,1) (0 = midnight, 0.25 = sunrise, 0.5 =
// noon, 0.75 = sunset) to a sky gradient (top, bottom), the sun direction, the
// sun's emission colour, and whether the sun is above the horizon.
func SkyForTime(t float64) (top, bottom, sunDir, sunColor raytrace.Vec3, sunUp bool) {
	t = t - math.Floor(t)
	a := 2 * math.Pi * (t - 0.25) // 0 at sunrise
	h := math.Sin(a)              // sun altitude, -1..1
	sunDir = raytrace.Vec3{X: math.Cos(a), Y: h, Z: 0.25}.Norm()
	sunUp = h > 0.03

	day := clamp01((h + 0.15) / 0.3)         // 0 night .. 1 full day
	horizon := clamp01(1 - math.Abs(h)/0.22) // 1 at dawn/dusk, 0 otherwise
	night := raytrace.Vec3{X: 0.02, Y: 0.03, Z: 0.08}
	dayTop := raytrace.Vec3{X: 0.35, Y: 0.55, Z: 0.92}
	dayBot := raytrace.Vec3{X: 0.80, Y: 0.86, Z: 0.96}
	warm := raytrace.Vec3{X: 0.98, Y: 0.45, Z: 0.18}

	top = lerpV(night, dayTop, day)
	bottom = lerpV(night, dayBot, day)
	bottom = lerpV(bottom, warm, horizon*0.75) // warm glow at the horizon
	top = lerpV(top, warm.Scale(0.5), horizon*0.35)
	sunColor = lerpV(warm.Scale(1.3), raytrace.Vec3{X: 1, Y: 0.97, Z: 0.9}, day).Scale(14 + 40*day)
	return top, bottom, sunDir, sunColor, sunUp
}

// SetTime sets the world's time of day (wrapped to [0,1)) and switches the world
// to time-driven sky and sun.
func (w *World) SetTime(t float64) {
	w.Time = t - math.Floor(t)
	w.timeSet = true
}

// EncodeEnv packs the time of day into 8 bytes for broadcast.
func EncodeEnv(t float64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(t))
	return b
}

// DecodeEnv parses a time of day produced by EncodeEnv.
func DecodeEnv(b []byte) (float64, error) {
	if len(b) < 8 {
		return 0, errors.New("raydir: env payload too short")
	}
	t := math.Float64frombits(binary.BigEndian.Uint64(b))
	if math.IsNaN(t) || math.IsInf(t, 0) {
		return 0, errors.New("raydir: bad env time")
	}
	return t, nil
}
