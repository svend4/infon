package raytrace

import (
	"math"
	"testing"
)

// entryHit returns the front-face hit of a ray entering an SSS sphere at the
// origin, plus a scene containing it.
func sssScene(albedo Vec3, radius float64) (*Scene, Ray, Hit) {
	s := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{SSS: 1.4, SSSColor: albedo, SSSRadius: radius}},
	}}
	r := Ray{Origin: Vec3{0, 0, -3}, Dir: Vec3{0, 0, 1}}
	h, _ := s.closest(r, geomEps, tFar)
	return s, r, h
}

// A lossless medium (albedo 1) must conserve energy: every photon that exits
// carries exactly its full throughput, and almost all photons exit.
func TestSubsurfaceLosslessConservesEnergy(t *testing.T) {
	s, r, h := sssScene(Vec3{1, 1, 1}, 0.5)
	rg := newRNG(1)
	exits, conserved := 0, 0
	const n = 300
	for i := 0; i < n; i++ {
		_, tput, exited := s.walkSubsurface(r, h, rg)
		if exited {
			exits++
			if tput == (Vec3{X: 1, Y: 1, Z: 1}) {
				conserved++
			}
		}
	}
	if exits < n*9/10 {
		t.Errorf("most lossless photons should exit, got %d/%d", exits, n)
	}
	if conserved != exits {
		t.Errorf("every exited lossless photon must carry full energy, got %d/%d", conserved, exits)
	}
}

func meanTransmit(radius float64, seed uint64) float64 {
	s, r, h := sssScene(Vec3{0.6, 0.6, 0.6}, radius)
	rg := newRNG(seed)
	const n = 4000
	var sum float64
	for i := 0; i < n; i++ {
		_, tput, exited := s.walkSubsurface(r, h, rg)
		if exited {
			sum += tput.X // Russian roulette is unbiased; absorbed photons count as 0.
		}
	}
	return sum / float64(n)
}

// A denser medium (shorter mean free path) makes more scatter events, so it
// absorbs more: a translucent object transmits more light than a dense one.
func TestSubsurfaceDensityAbsorbsMore(t *testing.T) {
	translucent := meanTransmit(1.0, 11)
	dense := meanTransmit(0.1, 11)
	if translucent <= dense {
		t.Errorf("translucent (%.3f) should transmit more than dense (%.3f)", translucent, dense)
	}
	if translucent <= 0 || translucent > 1.001 {
		t.Errorf("transmittance out of [0,1]: %.3f", translucent)
	}
	if math.Abs(translucent-dense) < 0.05 {
		t.Errorf("density should change transmittance noticeably: translucent=%.3f dense=%.3f", translucent, dense)
	}
}
