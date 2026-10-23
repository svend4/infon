package raytrace

import (
	"math"
	"sync"
	"testing"
)

// Rendering only reads the scene, so one Scene may be rendered from several
// goroutines at once (two eyes, two integrators, a preview and a refine pass).
// Every render rebuilds the emitter tables; that must not race with another
// render's workers reading them (run with -race), and both renders must see every
// light. Before the fix gatherEmitters truncated and refilled the shared slices in
// place, so a concurrent render could read a half-built table.
func TestAuditConcurrentRendersShareScene(t *testing.T) {
	emit := Material{Emit: Vec3{X: 4, Y: 4, Z: 4}}
	gray := Vec3{X: 0.6, Y: 0.6, Z: 0.6}
	s := &Scene{Objects: []Object{
		Plane{Y: 0, Size: 1, C1: gray, C2: gray},
		Sphere{Center: Vec3{X: -1.5, Y: 3, Z: 0}, Radius: 0.5, Mat: emit},
		Sphere{Center: Vec3{X: 1.5, Y: 3, Z: 0}, Radius: 0.5, Mat: emit},
		Triangle{A: Vec3{X: -1, Y: 3.5, Z: -1}, B: Vec3{X: 1, Y: 3.5, Z: -1}, C: Vec3{X: 0, Y: 3.5, Z: 1}, Mat: emit},
	}}
	cam := Camera{Pos: Vec3{X: 0, Y: 2.5, Z: 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	opt := PathOptions{Samples: 2, MaxDepth: 3, Seed: 1, NEE: true}
	const w, h = 16, 12
	want := pathSum(s, cam, w, h, opt)

	const n = 4
	got := make([][]Vec3, n)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // other integrators gather the same tables meanwhile
		defer wg.Done()
		AdaptiveRender(s, cam, w, h, PathOptions{Samples: 8, MaxDepth: 3, Seed: 2, NEE: true, MIS: true}, 0)
	}()
	go func() {
		defer wg.Done()
		BuildCaustics(s, 2000, 0.2, 1)
	}()
	for g := 0; g < n; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 3; i++ {
				got[g] = pathSum(s, cam, w, h, opt)
			}
		}(g)
	}
	wg.Wait()
	for g := range got {
		for i := range want {
			if got[g][i] != want[i] {
				t.Fatalf("render %d pixel %d = %+v, want %+v (a concurrent render saw a different light table)", g, i, got[g][i], want[i])
			}
		}
	}
}
