package raytrace

import (
	"testing"
)

// A negative image dimension (e.g. a terminal size minus a status line) must give
// an empty result, not a "makeslice: len out of range" / "makechan: size out of
// range" panic. Before the fix these entry points allocated w*h-sized buffers (or
// an h-sized channel) before checking the dimensions.
func TestAuditNegativeDimensionsDoNotPanic(t *testing.T) {
	scene := func() *Scene {
		return &Scene{Objects: []Object{
			Sphere{Center: Vec3{X: 0, Y: 0, Z: -3}, Radius: 1, Mat: Material{Color: Vec3{X: 1, Y: 1, Z: 1}}},
			Sphere{Center: Vec3{X: 0, Y: 4, Z: -3}, Radius: 0.5, Mat: Material{Emit: Vec3{X: 2, Y: 2, Z: 2}}},
		}}
	}
	cam := Camera{FOV: 1}
	opt := PathOptions{Samples: 1, NEE: true}
	entry := []struct {
		name string
		run  func(w, h int)
	}{
		{"pathSum", func(w, h int) { pathSum(scene(), cam, w, h, opt) }},
		{"AdaptiveRender", func(w, h int) { AdaptiveRender(scene(), cam, w, h, opt, 0) }},
		{"GBuffer", func(w, h int) { GBuffer(scene(), cam, w, h) }},
		{"Accumulator", func(w, h int) {
			a := NewAccumulator(w, h)
			a.AddSamples(scene(), cam, opt)
			a.Image()
		}},
		{"TemporalReprojector", func(w, h int) {
			tr := NewTemporalReprojector(w, h, 0.3)
			tr.Frame(scene(), cam, opt)
			tr.Frame(scene(), cam, opt)
		}},
		{"RenderVoxel", func(w, h int) { RenderVoxel(FBMTerrain(1, 0.01, 50), VoxelCamera{Height: 80}, w, h) }},
	}
	for _, e := range entry {
		for _, d := range [][2]int{{-3, 4}, {4, -3}, {-3, -4}, {0, 4}, {4, 0}} {
			if err := noPanic(func() { e.run(d[0], d[1]) }); err != nil {
				t.Errorf("%s(%d, %d): %v", e.name, d[0], d[1], err)
			}
		}
	}
}
