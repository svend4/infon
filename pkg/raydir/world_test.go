package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func startSpheres() []raytrace.Sphere {
	return []raytrace.Sphere{
		{Center: raytrace.Vec3{X: -1.5, Y: 1, Z: 0}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.8, Y: 0.3, Z: 0.3}}},
		{Center: raytrace.Vec3{X: 1.5, Y: 1, Z: 0}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.3, Y: 0.5, Z: 0.9}}},
	}
}

func TestRefWorldDirectorEvolvesAndStaysValid(t *testing.T) {
	w := &RefWorldDirector{}
	sph := startSpheres()
	first := sph[0].Center
	for i := 0; i < 50; i++ {
		sph = w.NextWorld(sph)
		for _, s := range sph {
			if s.Radius < 0.29 || s.Radius > 2.01 || s.Center.Y < s.Radius-1e-9 {
				t.Fatalf("sphere left valid range: r=%v y=%v", s.Radius, s.Center.Y)
			}
		}
	}
	if sph[0].Center.Sub(first).Len() < 1e-3 {
		t.Error("the world should have moved")
	}
}

func TestBrainWorldDirectorWithLocal(t *testing.T) {
	w := &BrainWorldDirector{B: brain.Local{}}
	sph := startSpheres()
	first := sph[0].Center
	for i := 0; i < 10; i++ {
		sph = w.NextWorld(sph)
	}
	if sph[0].Center.Sub(first).Len() < 1e-3 {
		t.Error("brain-authored world should have moved (reference returns spin=1)")
	}
}

func TestChoreographProducesAnim(t *testing.T) {
	anim := Choreograph(&BrainDirector{B: brain.Local{}}, &RefWorldDirector{},
		initState(), startSpheres(), raytrace.Vec3{X: 5, Y: 8, Z: -3}, 12)
	if anim.N != 2 || len(anim.Frames) != 12 {
		t.Fatalf("anim = N%d/%d frames, want N2/12", anim.N, len(anim.Frames))
	}
	// camera and world both advanced between the first and last frame.
	if anim.Frames[0].Cam.Pos.Sub(anim.Frames[11].Cam.Pos).Len() < 1e-3 {
		t.Error("camera should have moved across the animation")
	}
	if anim.Frames[0].Spheres[0].Center.Sub(anim.Frames[11].Spheres[0].Center).Len() < 1e-3 {
		t.Error("spheres should have moved across the animation")
	}
}

func TestRaySphereDirectorGathers(t *testing.T) {
	d := RaySphereDirector{B: brain.Local{}}
	sph := startSpheres() // two spheres at x = -1.5 and x = +1.5
	d0 := sph[0].Center.Sub(sph[1].Center).Len()
	for i := 0; i < 10; i++ {
		sph = d.NextWorld(sph)
	}
	d1 := sph[0].Center.Sub(sph[1].Center).Len()
	if d1 >= d0 {
		t.Errorf("game:ray reference should gather the spheres: %.2f -> %.2f", d0, d1)
	}
}
