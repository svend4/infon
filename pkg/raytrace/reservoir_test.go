package raytrace

import "testing"

func risMeanVar(s *Scene, h Hit, alb Vec3, m, n int, seed uint64) (Vec3, float64) {
	rg := newRNG(seed)
	var sum Vec3
	var sumL, sumL2 float64
	for i := 0; i < n; i++ {
		v := s.sampleEmittersRIS(h, alb, rg, 0, m)
		sum = sum.Add(v)
		l := lum3(v)
		sumL += l
		sumL2 += l * l
	}
	mean := sum.Scale(1 / float64(n))
	meanL := sumL / float64(n)
	return mean, sumL2/float64(n) - meanL*meanL
}

func neeMean(s *Scene, h Hit, alb Vec3, n int, seed uint64) Vec3 {
	rg := newRNG(seed)
	var sum Vec3
	for i := 0; i < n; i++ {
		sum = sum.Add(s.sampleEmitters(h, alb, rg, false, 0))
	}
	return sum.Scale(1 / float64(n))
}

// RIS must be unbiased (same mean as single-sample NEE and across candidate
// counts) while cutting variance when there are several lights to choose from.
func TestRISUnbiasedAndLowerVariance(t *testing.T) {
	s := &Scene{Objects: []Object{
		Sphere{Center: Vec3{-3, 4, 1}, Radius: 0.5, Mat: Material{Emit: Vec3{20, 20, 20}}},
		Sphere{Center: Vec3{3, 5, -1}, Radius: 0.5, Mat: Material{Emit: Vec3{25, 25, 25}}},
		Sphere{Center: Vec3{0, 6, 2}, Radius: 0.5, Mat: Material{Emit: Vec3{15, 15, 15}}},
		Sphere{Center: Vec3{1, 4, -3}, Radius: 0.5, Mat: Material{Emit: Vec3{30, 30, 30}}},
	}}
	s.gatherEmitters()
	h := Hit{P: Vec3{0, 0, 0}, N: Vec3{0, 1, 0}}
	alb := Vec3{0.8, 0.8, 0.8}

	const n = 60000
	m1, v1 := risMeanVar(s, h, alb, 1, n, 1)
	m8, v8 := risMeanVar(s, h, alb, 8, n, 2)
	nee := neeMean(s, h, alb, n, 3)

	if rel := m8.Sub(nee).Len() / nee.Len(); rel > 0.03 {
		t.Errorf("RIS(8) mean should match NEE: ris=%v nee=%v rel=%.3f", m8, nee, rel)
	}
	if rel := m8.Sub(m1).Len() / m1.Len(); rel > 0.03 {
		t.Errorf("RIS should be unbiased across candidate counts: m1=%v m8=%v rel=%.3f", m1, m8, rel)
	}
	if v8 >= v1 {
		t.Errorf("RIS should reduce variance with more candidates: var(1)=%.4f var(8)=%.4f", v1, v8)
	}
}

// With m == 1 RIS is exactly the single-sample estimator, so its mean equals NEE.
func TestRISSingleCandidateEqualsNEE(t *testing.T) {
	s := &Scene{Objects: []Object{
		Sphere{Center: Vec3{2, 5, 0}, Radius: 0.6, Mat: Material{Emit: Vec3{18, 14, 10}}},
		Sphere{Center: Vec3{-2, 4, 1}, Radius: 0.4, Mat: Material{Emit: Vec3{8, 12, 18}}},
	}}
	s.gatherEmitters()
	h := Hit{P: Vec3{0, 0, 0}, N: Vec3{0, 1, 0}}
	alb := Vec3{0.7, 0.7, 0.7}
	const n = 60000
	ris1, _ := risMeanVar(s, h, alb, 1, n, 5)
	nee := neeMean(s, h, alb, n, 6)
	if rel := ris1.Sub(nee).Len() / nee.Len(); rel > 0.03 {
		t.Errorf("RIS(1) should equal NEE in the mean: ris=%v nee=%v rel=%.3f", ris1, nee, rel)
	}
}
