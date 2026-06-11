package raydir

import "math"

// face.go provides a reusable synthetic face — normalized keypoints (oval outline,
// two eyes, nose, mouth) in 0..1 — for demos and tests of AvatarFace, varied a
// little per seed. Real calls feed live landmarks from internal/avatar instead.

// DemoFace returns synthetic face keypoints (x,y in 0..1, y down), seed-varied.
func DemoFace(seed int) [][2]float32 {
	var p [][2]float32
	add := func(x, y float64) { p = append(p, [2]float32{float32(x), float32(y)}) }
	for i := 0; i < 18; i++ { // oval outline
		a := float64(i) / 18 * 2 * math.Pi
		add(0.5+0.42*math.Cos(a), 0.5+0.47*math.Sin(a))
	}
	eyeY := 0.40 + 0.02*float64(seed%2)
	for _, ex := range []float64{0.35, 0.65} { // two eyes
		for i := 0; i < 6; i++ {
			a := float64(i) / 6 * 2 * math.Pi
			add(ex+0.06*math.Cos(a), eyeY+0.04*math.Sin(a))
		}
	}
	add(0.5, 0.50) // nose
	add(0.5, 0.56)
	smile := 0.03 + 0.03*float64(seed%3) // a varied mouth curve
	for i := 0; i <= 8; i++ {
		t := float64(i) / 8
		add(0.36+0.28*t, 0.66+smile*math.Sin(t*math.Pi))
	}
	return p
}
