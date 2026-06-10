package raydir

import (
	"image"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func refScene() *raytrace.Scene {
	return &raytrace.Scene{
		SkyTop: raytrace.Vec3{X: 0.4, Y: 0.5, Z: 0.8}, SkyBottom: raytrace.Vec3{X: 0.9, Y: 0.9, Z: 0.95},
		Objects: []raytrace.Object{
			raytrace.Plane{Y: 0, Size: 1, C1: raytrace.Vec3{X: 0.6, Y: 0.6, Z: 0.6}, C2: raytrace.Vec3{X: 0.4, Y: 0.4, Z: 0.4}},
			raytrace.Sphere{Center: raytrace.Vec3{X: 0, Y: 1, Z: 4}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.8, Y: 0.3, Z: 0.3}}},
			raytrace.Sphere{Center: raytrace.Vec3{X: 0, Y: 6, Z: 3}, Radius: 1, Mat: raytrace.Material{Emit: raytrace.Vec3{X: 18, Y: 18, Z: 18}}},
		},
	}
}

// Holding the camera still accumulates samples (the view refines); moving the
// camera restarts the accumulation.
func TestRefinerAccumulatesAndResets(t *testing.T) {
	scene := refScene()
	r := NewRefiner(24, 18, 4, 64, raytrace.PathOptions{MaxDepth: 3, Seed: 1, NEE: true, MIS: true})
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 1.5, Z: 0}, Pitch: -0.05, FOV: 1.0}
	r.Frame(scene, cam)
	if r.Samples() != 4 {
		t.Fatalf("after one frame, samples = %d, want 4", r.Samples())
	}
	r.Frame(scene, cam) // same camera -> accumulate
	if r.Samples() != 8 {
		t.Fatalf("holding still should accumulate to 8, got %d", r.Samples())
	}
	moved := cam
	moved.Pos.X += 1
	r.Frame(scene, moved) // camera changed -> reset to one batch
	if r.Samples() != 4 {
		t.Errorf("moving should restart accumulation, got %d", r.Samples())
	}
}

// Refinement reduces noise: an 8-sample refined view is closer to a converged
// reference than a 1-batch view.
func TestRefinerReducesNoise(t *testing.T) {
	scene := refScene()
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 1.5, Z: 0}, Pitch: -0.05, FOV: 1.0}
	const w, h = 24, 18
	ref := raytrace.PathRender(scene, cam, w, h, raytrace.PathOptions{Samples: 400, MaxDepth: 3, Seed: 7, NEE: true, MIS: true})
	errTo := func(img image.Image) float64 {
		var s float64
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				ar, ag, ab, _ := img.At(x, y).RGBA()
				br, bg, bb, _ := ref.At(x, y).RGBA()
				dr := float64(int(ar>>8) - int(br>>8))
				dg := float64(int(ag>>8) - int(bg>>8))
				db := float64(int(ab>>8) - int(bb>>8))
				s += dr*dr + dg*dg + db*db
			}
		}
		return s
	}
	r := NewRefiner(w, h, 2, 64, raytrace.PathOptions{MaxDepth: 3, Seed: 1, NEE: true, MIS: true})
	e1 := errTo(r.Frame(scene, cam)) // 2 spp
	var refined image.Image
	for i := 0; i < 15; i++ { // hold still -> ~30 spp
		refined = r.Frame(scene, cam)
	}
	if e2 := errTo(refined); e2 >= e1 {
		t.Errorf("refining should approach the reference: start err %.0f, refined err %.0f", e1, e2)
	}
}
