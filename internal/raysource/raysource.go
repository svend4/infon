// Package raysource exposes the pkg/raytrace CPU ray tracer as a
// device.FrameGenerator, so the whole terminal-video pipeline (preview, send,
// call, group calls, recording) can use a live ray-traced 3-D scene as its
// "camera": synthesized pixels, no capture, no GPU. Because GenerativeSource
// already adapts a FrameGenerator to device.Camera, ray3d becomes a drop-in
// camera anywhere TVCP reads frames.
package raysource

import (
	"image"
	"math"
	"time"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/raytrace"
)

// RayGenerator renders an orbiting ray-traced scene, one frame at a time. It
// animates by wall-clock time so it looks alive in a live call/preview.
type RayGenerator struct {
	scene  *raytrace.Scene
	period float64 // seconds per full turntable revolution
	radius float64
	height float64
	target raytrace.Vec3
	spp    int
}

var _ device.FrameGenerator = (*RayGenerator)(nil)

// New returns a RayGenerator that orbits the given scene (a default demo scene if
// scene is nil). spp is the anti-aliasing sample count per axis.
func New(scene *raytrace.Scene, spp int) *RayGenerator {
	if scene == nil {
		scene = DemoScene()
	}
	if spp < 1 {
		spp = 1
	}
	return &RayGenerator{
		scene:  scene,
		period: 8,
		radius: 6,
		height: 2.4,
		target: raytrace.Vec3{X: 0, Y: 1, Z: 0},
		spp:    spp,
	}
}

// Name implements device.FrameGenerator.
func (g *RayGenerator) Name() string { return "raytrace" }

// Generate renders one orbiting frame at the requested resolution.
func (g *RayGenerator) Generate(width, height, frameNum int, elapsed time.Duration) (image.Image, error) {
	ang := 2 * math.Pi * (elapsed.Seconds() / g.period)
	pos := raytrace.Vec3{X: math.Sin(ang) * g.radius, Y: g.height, Z: math.Cos(ang) * g.radius}
	d := g.target.Sub(pos)
	cam := raytrace.Camera{
		Pos:   pos,
		Yaw:   math.Atan2(d.X, d.Z),
		Pitch: math.Atan2(d.Y, math.Hypot(d.X, d.Z)),
		FOV:   math.Pi / 3,
	}
	return raytrace.Render(g.scene, cam, width, height, raytrace.Options{Samples: g.spp}), nil
}

// DemoScene builds the default scene: a red glossy sphere, a mirror sphere and a
// glass sphere on a checkerboard floor under a soft sky.
func DemoScene() *raytrace.Scene {
	w := raytrace.SceneWire{
		Light: raytrace.Vec3{X: 6, Y: 9, Z: -4},
		Spheres: []raytrace.Sphere{
			{Center: raytrace.Vec3{X: -1.3, Y: 1, Z: 0}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.85, Y: 0.25, Z: 0.25}, Spec: 0.4, Shine: 48}},
			{Center: raytrace.Vec3{X: 1.3, Y: 1, Z: 0.4}, Radius: 1, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.9, Y: 0.9, Z: 0.95}, Reflect: 0.7}},
			{Center: raytrace.Vec3{X: 0, Y: 0.55, Z: 2}, Radius: 0.55, Mat: raytrace.Material{Glass: 1.5}},
		},
	}
	return w.Build(0.14)
}
