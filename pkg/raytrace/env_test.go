package raytrace

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestEnvSamplerIntegratesToSphere(t *testing.T) {
	// A proper solid-angle pdf satisfies E[1/pdf] = total sphere area = 4*pi.
	img := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 16; x++ {
			c := uint8(40 + (x*9)%200)
			img.SetRGBA(x, y, color.RGBA{R: c, G: c, B: c, A: 255})
		}
	}
	e := BuildEnvSampler(img)
	rg := newRNG(1)
	var s float64
	const N = 200000
	for i := 0; i < N; i++ {
		if _, _, pdf := e.Sample(rg); pdf > 0 {
			s += 1 / pdf
		}
	}
	avg := s / float64(N)
	if math.Abs(avg-4*math.Pi) > 1.0 {
		t.Errorf("E[1/pdf] = %.3f, want ~4*pi = %.3f (Jacobian wrong?)", avg, 4*math.Pi)
	}
}

func TestEnvSamplerFavorsBrightRegions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 2, G: 2, B: 2, A: 255})
		}
	}
	img.SetRGBA(12, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // one bright texel
	e := BuildEnvSampler(img)
	rg := newRNG(2)
	near, total := 0, 20000
	for i := 0; i < total; i++ {
		d, _, _ := e.Sample(rg)
		u := 0.5 + math.Atan2(d.Z, d.X)/(2*math.Pi)
		v := 0.5 - math.Asin(math.Max(-1, math.Min(1, d.Y)))/math.Pi
		if int(u*16) == 12 && int(v*8) == 2 {
			near++
		}
	}
	if frac := float64(near) / float64(total); frac < 0.5 {
		t.Errorf("the bright texel should dominate sampling, got %.2f", frac)
	}
}

func TestEnvNEEIsUnbiased(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			c := uint8(10)
			if y < 3 {
				c = 255 // a bright band near the top of the sky
			}
			img.SetRGBA(x, y, color.RGBA{R: c, G: c, B: c, A: 255})
		}
	}
	tex := ImageTex{Img: img}
	build := func(env bool) *Scene {
		s := &Scene{
			Objects: []Object{Plane{Y: 0, Size: 1, C1: Vec3{0.7, 0.7, 0.7}, C2: Vec3{0.7, 0.7, 0.7}}},
			SkyTex:  tex,
		}
		if env {
			s.Env = BuildEnvSampler(img)
		}
		return s
	}
	cam := Camera{Pos: Vec3{0, 2, 5}, Yaw: math.Pi, Pitch: -0.4, FOV: 1.0}
	avg := func(s *Scene, opt PathOptions) float64 {
		im := PathRender(s, cam, 28, 28, opt)
		var sum float64
		n := 0
		for y := 16; y < 26; y++ {
			for x := 6; x < 22; x++ {
				r, g, b, _ := im.At(x, y).RGBA()
				sum += lum(r, g, b)
				n++
			}
		}
		return sum / float64(n)
	}
	ref := avg(build(false), PathOptions{Samples: 400, MaxDepth: 3, Seed: 1})
	en := avg(build(true), PathOptions{Samples: 48, MaxDepth: 3, Seed: 2, NEE: true})
	if tol := ref*0.2 + 400; math.Abs(en-ref) > tol {
		t.Errorf("env-NEE mean %.0f vs naive %.0f (tol %.0f) - biased?", en, ref, tol)
	}
}

// An env sampler built from the Preetham sky favours the sun direction (its pdf is
// higher toward the sun than away), so env-NEE focuses where the light is.
func TestEnvSamplerFromSky(t *testing.T) {
	sun := Vec3{X: 0.3, Y: 0.5, Z: 0.8}.Norm()
	sky := NewPreethamSky(sun, 2.5)
	e := BuildEnvSamplerFromSky(sky.At, 64, 32)
	if e == nil {
		t.Fatal("sampler should build")
	}
	toward := e.Pdf(sun)
	away := e.Pdf(Vec3{X: -0.3, Y: 0.5, Z: -0.8}.Norm())
	if toward <= away {
		t.Errorf("sky sampler should favour the sun: toward %.4f away %.4f", toward, away)
	}
}
