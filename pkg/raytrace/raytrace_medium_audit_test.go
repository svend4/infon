package raytrace

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// skyCapImage is a small equirect env map: a bright polar cap (the top rows of
// 16) over a dim sky.
func skyCapImage(rows int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			c := uint8(20)
			if y < rows {
				c = 255
			}
			img.SetRGBA(x, y, color.RGBA{R: c, G: c, B: c, A: 255})
		}
	}
	return img
}

// meanPath is the mean linear luminance per sample of a path-traced render.
func meanPath(s *Scene, cam Camera, w, h int, opt PathOptions) float64 {
	return meanLinLum(pathSum(s, cam, w, h, opt)) / float64(opt.Samples)
}

// A purely absorbing, optically thick medium blocks light like an opaque black
// object, so a surface's next-event estimate must be shadowed by it: a floor lit
// through an absorbing ball must match the same floor behind a black sphere, for
// NEE, NEE+MIS and RIS, for both an emissive sphere and the environment. Before
// the fix sampleEmitters / sampleEmittersRIS / sampleEnv tested only surfaces, so
// direct light went straight through the medium.
func TestAuditSurfaceNEEAttenuatedByMedium(t *testing.T) {
	gray := Vec3{X: 0.6, Y: 0.6, Z: 0.6}
	floor := Plane{Y: 0, Size: 1, C1: gray, C2: gray}
	cam := Camera{Pos: Vec3{X: 0, Y: 2.5, Z: 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	light := Sphere{Center: Vec3{X: 0, Y: 4, Z: 0}, Radius: 0.5, Mat: Material{Emit: Vec3{X: 8, Y: 8, Z: 8}}}
	sky := skyCapImage(2)
	env := BuildEnvSampler(sky)

	cases := []struct {
		name   string
		center Vec3    // occluder centre
		radius float64 // occluder radius
		env    bool    // lit by the environment instead of the sphere light
		ris    int
		opt    PathOptions
	}{
		{"sphere/NEE", Vec3{X: 0, Y: 2, Z: 0}, 1, false, 0, PathOptions{NEE: true}},
		{"sphere/MIS", Vec3{X: 0, Y: 2, Z: 0}, 1, false, 0, PathOptions{NEE: true, MIS: true}},
		{"sphere/RIS", Vec3{X: 0, Y: 2, Z: 0}, 1, false, 4, PathOptions{NEE: true}},
		{"env/NEE", Vec3{X: 0, Y: 4.5, Z: 0}, 3, true, 0, PathOptions{NEE: true}},
		{"env/MIS", Vec3{X: 0, Y: 4.5, Z: 0}, 3, true, 0, PathOptions{NEE: true, MIS: true}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			mk := func(occ Object, m *Medium) *Scene {
				s := &Scene{Objects: []Object{floor, occ}, Medium: m, RISCandidates: c.ris}
				if c.env {
					s.SkyTex, s.Env = ImageTex{Img: sky}, env
				} else {
					s.Objects = append(s.Objects, light)
				}
				return s
			}
			opt := c.opt
			opt.Samples, opt.MaxDepth, opt.Seed = 64, 4, 1
			const w, h = 32, 24
			none := Sphere{Center: Vec3{X: 50, Y: -50, Z: 50}, Radius: 0.1}   // out of the way
			ball := &Medium{Center: c.center, Radius: c.radius, SigmaMax: 60} // Albedo 0: pure absorber
			open := meanPath(mk(none, nil), cam, w, h, opt)
			ref := meanPath(mk(Sphere{Center: c.center, Radius: c.radius}, nil), cam, w, h, opt)
			med := meanPath(mk(none, ball), cam, w, h, opt)
			t.Logf("unoccluded %.4f  black sphere %.4f  absorbing medium %.4f", open, ref, med)
			if open-ref < 0.2*open {
				t.Fatalf("occluder barely matters (%.4f vs %.4f): test scene is not discriminating", open, ref)
			}
			if rel := math.Abs(med-ref) / ref; rel > 0.06 {
				t.Errorf("absorbing medium lets direct light through: %.4f vs black occluder %.4f (%.0f%% off)", med, ref, rel*100)
			}
		})
	}
}

// Light scattered by the medium must not depend on whether the light source is
// in the light-sampling set. A medium lit by an emissive quad (two Triangles,
// which NEE samples) must glow as much as one lit by the same quad wrapped in an
// Instance (which NEE does not sample, so phase-sampled hits count in full); and a
// medium under an env map must glow the same with and without env importance
// sampling. Before the fix mediumNEE sampled spheres only and dropped the
// phase-sampled hits, so triangle and environment in-scattering was zero.
//
// Next-event estimation reaches a light one segment earlier than a sampled hit,
// so the reference (lights found by phase sampling only) gets one more bounce.
func TestAuditMediumInScatterFromAllLights(t *testing.T) {
	cam := Camera{Pos: Vec3{}, FOV: 0.7} // looks down +Z; the medium fills the view
	med := &Medium{Center: Vec3{X: 0, Y: 0, Z: 5}, Radius: 2, SigmaMax: 1.5, Albedo: Vec3{X: 0.9, Y: 0.9, Z: 0.9}}
	emit := Material{Emit: Vec3{X: 5, Y: 5, Z: 5}}
	t1 := Triangle{A: Vec3{X: -2, Y: 4, Z: 3}, B: Vec3{X: 2, Y: 4, Z: 3}, C: Vec3{X: 2, Y: 4, Z: 7}, Mat: emit}
	t2 := Triangle{A: Vec3{X: -2, Y: 4, Z: 3}, B: Vec3{X: 2, Y: 4, Z: 7}, C: Vec3{X: -2, Y: 4, Z: 7}, Mat: emit}
	img := skyCapImage(4)
	const w, h, depth = 16, 16, 3

	cases := []struct {
		name     string
		ref, got *Scene
	}{
		{"triangles",
			&Scene{Objects: []Object{NewInstance(t1, Translate(Vec3{})), NewInstance(t2, Translate(Vec3{}))}, Medium: med},
			&Scene{Objects: []Object{t1, t2}, Medium: med}},
		{"env",
			&Scene{SkyTex: ImageTex{Img: img}, Medium: med},
			&Scene{SkyTex: ImageTex{Img: img}, Env: BuildEnvSampler(img), Medium: med}},
	}
	for _, c := range cases {
		for _, mis := range []bool{false, true} {
			opt := PathOptions{Samples: 512, MaxDepth: depth, Seed: 2, NEE: true, MIS: mis}
			refOpt := opt
			refOpt.MaxDepth++
			ref := meanPath(c.ref, cam, w, h, refOpt)
			got := meanPath(c.got, cam, w, h, opt)
			t.Logf("%s MIS=%v: phase-sampled only %.4f  light-sampled %.4f", c.name, mis, ref, got)
			if ref <= 1e-3 {
				t.Fatalf("%s: reference medium is black (%.5f)", c.name, ref)
			}
			if rel := math.Abs(got-ref) / ref; rel > 0.06 {
				t.Errorf("%s MIS=%v: medium in-scatter %.4f, want %.4f (%.0f%% off)", c.name, mis, got, ref, rel*100)
			}
		}
	}
}
