package raytrace

import (
	"math"
	"testing"
)

// Power-weighted light selection stays unbiased with several lights of very
// different brightness: NEE+MIS must converge to the same mean as pure BSDF
// sampling. (If the per-light selection probability and the MIS pdf disagreed, the
// means would diverge.)
func TestManyLightUnbiased(t *testing.T) {
	bright := Material{Emit: Vec3{X: 6, Y: 6, Z: 6}}
	dim := Material{Emit: Vec3{X: 0.4, Y: 0.4, Z: 0.5}}
	scene := func() *Scene {
		return &Scene{Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{X: 0.6, Y: 0.6, Z: 0.6}, C2: Vec3{X: 0.6, Y: 0.6, Z: 0.6}},
			// two overhead quads of very different power
			Triangle{A: Vec3{-4, 3, -4}, B: Vec3{0, 3, -4}, C: Vec3{0, 3, 4}, Mat: bright},
			Triangle{A: Vec3{-4, 3, -4}, B: Vec3{0, 3, 4}, C: Vec3{-4, 3, 4}, Mat: bright},
			Triangle{A: Vec3{0, 3, -4}, B: Vec3{4, 3, -4}, C: Vec3{4, 3, 4}, Mat: dim},
			Triangle{A: Vec3{0, 3, -4}, B: Vec3{4, 3, 4}, C: Vec3{0, 3, 4}, Mat: dim},
		}}
	}
	cam := Camera{Pos: Vec3{X: 0, Y: 2.5, Z: 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	const w, h = 48, 36
	bsdf := meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: 500, MaxDepth: 4, Seed: 9})) / 500
	nee := meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: 80, MaxDepth: 4, Seed: 1, NEE: true, MIS: true})) / 80
	if bsdf <= 1e-4 {
		t.Fatalf("reference is black (%.5f)", bsdf)
	}
	if rel := math.Abs(nee-bsdf) / bsdf; rel > 0.06 {
		t.Errorf("power-weighted many-light NEE biased: BSDF %.4f vs NEE+MIS %.4f (%.1f%%)", bsdf, nee, rel*100)
	}
}

// An emissive mesh inside an instance is now NEE-sampled (transformed to world):
// NEE+MIS must match pure BSDF sampling. Before emissive-in-mesh gathering, the
// floor's indirect light from the mesh was undercounted in NEE mode (biased).
func TestEmissiveMeshNEEUnbiased(t *testing.T) {
	emit := Material{Emit: Vec3{X: 4, Y: 4, Z: 4}}
	mesh := NewMesh([]Triangle{
		{A: Vec3{-2, 0, -2}, B: Vec3{2, 0, -2}, C: Vec3{2, 0, 2}, Mat: emit},
		{A: Vec3{-2, 0, -2}, B: Vec3{2, 0, 2}, C: Vec3{-2, 0, 2}, Mat: emit},
	})
	scene := func() *Scene {
		return &Scene{Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{X: 0.6, Y: 0.6, Z: 0.6}, C2: Vec3{X: 0.6, Y: 0.6, Z: 0.6}},
			NewInstance(mesh, Translate(Vec3{X: 0, Y: 3, Z: 0})), // an emissive panel overhead
		}}
	}
	cam := Camera{Pos: Vec3{X: 0, Y: 2.5, Z: 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	const w, h = 48, 36
	bsdf := meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: 600, MaxDepth: 4, Seed: 9})) / 600
	nee := meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: 80, MaxDepth: 4, Seed: 1, NEE: true, MIS: true})) / 80
	if bsdf <= 1e-4 {
		t.Fatalf("reference is black (%.5f); emissive mesh not lighting", bsdf)
	}
	if rel := math.Abs(nee-bsdf) / bsdf; rel > 0.07 {
		t.Errorf("emissive-mesh NEE biased: BSDF %.4f vs NEE+MIS %.4f (%.1f%%)", bsdf, nee, rel*100)
	}
}
