package raytrace

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// modeMeans renders scene with pure BSDF sampling (mode 0, the unbiased
// reference), NEE (mode 1) and NEE+MIS (mode 2) and returns the three mean
// linear luminances.
func modeMeans(scene func() *Scene, refSpp, spp int) (ref, nee, mis float64) {
	cam := Camera{Pos: Vec3{X: 0, Y: 2.5, Z: 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	const w, h = 32, 24
	ref = meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: refSpp, MaxDepth: 4, Seed: 9})) / float64(refSpp)
	nee = meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: spp, MaxDepth: 4, Seed: 1, NEE: true})) / float64(spp)
	mis = meanLinLum(pathSum(scene(), cam, w, h, PathOptions{Samples: spp, MaxDepth: 4, Seed: 1, NEE: true, MIS: true})) / float64(spp)
	return ref, nee, mis
}

// checkModesAgree requires NEE and NEE+MIS to match the mode-0 reference within
// tol. The tolerance also absorbs the few-percent shadow-ray self-occlusion of
// light-sampled emitters at grazing angles (sampleEmitters' dist-2*shadowEps),
// which the Lambertian triangle-light tests already tolerate.
func checkModesAgree(t *testing.T, scene func() *Scene, refSpp, spp int, tol float64) {
	t.Helper()
	ref, nee, mis := modeMeans(scene, refSpp, spp)
	t.Logf("BSDF %.4f  NEE %.4f  NEE+MIS %.4f", ref, nee, mis)
	if ref <= 1e-4 {
		t.Fatalf("reference is black (%.5f)", ref)
	}
	if rel := math.Abs(nee-ref) / ref; rel > tol {
		t.Errorf("NEE biased: BSDF %.4f vs NEE %.4f (%.1f%%)", ref, nee, rel*100)
	}
	if rel := math.Abs(mis-ref) / ref; rel > tol {
		t.Errorf("NEE+MIS biased: BSDF %.4f vs NEE+MIS %.4f (%.1f%%)", ref, mis, rel*100)
	}
}

// A Principled (Disney-style) floor lit by an emissive quad: its diffuse lobe
// must receive the same direct light under NEE and NEE+MIS as under pure BSDF
// sampling. Before the fix the diffuse lobe never did next-event estimation and
// left prevPdfB stale, so modes 1/2 lost the floor's direct light.
func TestAuditPrincipledDiffuseNEEUnbiased(t *testing.T) {
	emit := Material{Emit: Vec3{X: 3, Y: 3, Z: 3}}
	for _, metal := range []float64{0, 0.5} {
		floor := Material{Principled: true, Rough: 0.5, Metal: metal}
		scene := func() *Scene {
			return &Scene{Objects: []Object{
				Plane{Y: 0, Size: 1, C1: Vec3{X: 0.6, Y: 0.6, Z: 0.6}, C2: Vec3{X: 0.6, Y: 0.6, Z: 0.6}, Mat: floor},
				Triangle{A: Vec3{-4, 3, -4}, B: Vec3{4, 3, -4}, C: Vec3{4, 3, 4}, Mat: emit},
				Triangle{A: Vec3{-4, 3, -4}, B: Vec3{4, 3, 4}, C: Vec3{-4, 3, 4}, Mat: emit},
			}}
		}
		checkModesAgree(t, scene, 400, 100, 0.06)
	}
}

// A Principled floor under an importance-sampled environment: the diffuse lobe
// must do env-NEE and record its BSDF pdf so the sky hit is MIS-weighted
// correctly; otherwise the env light on the diffuse lobe is dropped.
func TestAuditPrincipledDiffuseEnvNEEUnbiased(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			c := uint8(20)
			if y < 4 {
				c = 255
			}
			img.SetRGBA(x, y, color.RGBA{R: c, G: c, B: c, A: 255})
		}
	}
	env := BuildEnvSampler(img)
	scene := func() *Scene {
		return &Scene{
			Objects: []Object{Plane{Y: 0, Size: 1, C1: Vec3{X: 0.6, Y: 0.6, Z: 0.6}, C2: Vec3{X: 0.6, Y: 0.6, Z: 0.6},
				Mat: Material{Principled: true, Rough: 0.5}}},
			SkyTex: ImageTex{Img: img},
			Env:    env,
		}
	}
	// keep the camera off the sky so the mean is the floor's lighting only
	checkModesAgree(t, scene, 400, 100, 0.06)
}

// Emissive objects that gatherEmitters does not collect (a MovingSphere, a
// sphere inside an Instance, an emissive Plane) are never light-sampled, so a
// BSDF-sampled hit on them after a diffuse bounce is their only estimator: NEE
// and NEE+MIS must still match pure BSDF sampling.
func TestAuditUnsampledEmitterUnbiased(t *testing.T) {
	emit := Material{Emit: Vec3{X: 3, Y: 3, Z: 3}}
	gray := Vec3{X: 0.6, Y: 0.6, Z: 0.6}
	floor := Plane{Y: 0, Size: 1, C1: gray, C2: gray}
	cases := []struct {
		name string
		obj  Object
	}{
		{"MovingSphere", MovingSphere{C0: Vec3{X: 0, Y: 3.5, Z: 0}, C1: Vec3{X: 0, Y: 3.5, Z: 0}, Radius: 1.5, Mat: emit}},
		{"InstanceSphere", NewInstance(Sphere{Radius: 1.5, Mat: emit}, Translate(Vec3{X: 0, Y: 3.5, Z: 0}))},
		{"Plane", Plane{Y: 3, Size: 1, Mat: emit}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			scene := func() *Scene { return &Scene{Objects: []Object{floor, c.obj}} }
			checkModesAgree(t, scene, 300, 100, 0.06)
		})
	}
	// Mixed: a light-sampled quad plus an unsampled MovingSphere. The sampled
	// light must stay NEE/MIS-weighted (not double counted) while the other is
	// added through BSDF sampling.
	t.Run("Mixed", func(t *testing.T) {
		scene := func() *Scene {
			return &Scene{Objects: []Object{
				floor,
				Triangle{A: Vec3{-4, 3, -4}, B: Vec3{0, 3, -4}, C: Vec3{0, 3, 4}, Mat: emit},
				Triangle{A: Vec3{-4, 3, -4}, B: Vec3{0, 3, 4}, C: Vec3{-4, 3, 4}, Mat: emit},
				MovingSphere{C0: Vec3{X: 2, Y: 3.5, Z: 0}, C1: Vec3{X: 2, Y: 3.5, Z: 0}, Radius: 1, Mat: emit},
			}}
		}
		checkModesAgree(t, scene, 400, 100, 0.06)
	})
}
