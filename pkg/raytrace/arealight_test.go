package raytrace

import (
	"math"
	"testing"
)

// An emissive triangle is a NEE-sampled area light: the path tracer with NEE+MIS
// must converge to the SAME mean floor radiance as pure BSDF sampling (mode 0).
// That is the unbiasedness proof for the triangle-light estimator and its MIS
// weight — if the area, count or pdf were wrong, the means would diverge.
func triLitScene() *Scene {
	emit := Material{Emit: Vec3{3, 3, 3}}
	return &Scene{Objects: []Object{
		Plane{Y: 0, Size: 1, C1: Vec3{0.6, 0.6, 0.6}, C2: Vec3{0.6, 0.6, 0.6}},
		// a large quad (two triangles) overhead, facing down, out of frame
		Triangle{A: Vec3{-4, 3, -4}, B: Vec3{4, 3, -4}, C: Vec3{4, 3, 4}, Mat: emit},
		Triangle{A: Vec3{-4, 3, -4}, B: Vec3{4, 3, 4}, C: Vec3{-4, 3, 4}, Mat: emit},
	}}
}

func TestTriangleAreaLightUnbiased(t *testing.T) {
	cam := Camera{Pos: Vec3{0, 2.5, 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	const w, h = 48, 36

	const refSpp = 500
	bsdf := meanLinLum(pathSum(triLitScene(), cam, w, h, PathOptions{Samples: refSpp, MaxDepth: 4, Seed: 9})) / refSpp
	const neeSpp = 80
	nee := meanLinLum(pathSum(triLitScene(), cam, w, h, PathOptions{Samples: neeSpp, MaxDepth: 4, Seed: 1, NEE: true, MIS: true})) / neeSpp

	if bsdf <= 1e-4 {
		t.Fatalf("reference is black (%.5f); scene not lit by the triangle", bsdf)
	}
	if rel := math.Abs(nee-bsdf) / bsdf; rel > 0.05 {
		t.Errorf("triangle-light NEE biased: BSDF mean %.4f vs NEE+MIS %.4f (%.1f%% off)", bsdf, nee, rel*100)
	}
}

// NEE+MIS makes the triangle area light far less noisy than pure BSDF sampling at
// the same sample count: at low spp it should sit closer to a converged reference.
func TestTriangleAreaLightLessNoisy(t *testing.T) {
	cam := Camera{Pos: Vec3{0, 2.5, 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	const w, h = 40, 30
	ref := pathSum(triLitScene(), cam, w, h, PathOptions{Samples: 1500, MaxDepth: 4, Seed: 7, NEE: true, MIS: true})
	for i := range ref {
		ref[i] = ref[i].Scale(1.0 / 1500)
	}
	rms := func(buf []Vec3, spp int) float64 {
		var s float64
		for i, v := range buf {
			d := v.Scale(1.0 / float64(spp)).Sub(ref[i])
			s += d.Dot(d)
		}
		return math.Sqrt(s / float64(len(buf)))
	}
	const spp = 16
	noisy := rms(pathSum(triLitScene(), cam, w, h, PathOptions{Samples: spp, MaxDepth: 4, Seed: 3}), spp)
	clean := rms(pathSum(triLitScene(), cam, w, h, PathOptions{Samples: spp, MaxDepth: 4, Seed: 3, NEE: true, MIS: true}), spp)
	if clean >= noisy {
		t.Errorf("NEE should be less noisy: BSDF rms %.4f vs NEE rms %.4f", noisy, clean)
	}
}
