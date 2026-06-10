// voxel.go is a voxel-terrain renderer in the classic "Voxel Space" (Comanche)
// style — the very thing the dream hackers describe seeing as their dreams form:
// "large rectangular pixel-blocks", a height field drawn column by column. A
// heightmap and a colormap are walked front-to-back with a per-column y-buffer, so
// near ridges occlude far ones; distance fades toward the sky (aerial haze). It is
// a separate, cheap rasteriser — no path tracing — for the blocky dream look.
package raytrace

import (
	"image"
	"image/color"
	"math"
)

// VoxelTerrain is a height field with a surface colour at each map point.
type VoxelTerrain struct {
	Height func(x, z float64) float64    // terrain height
	Shade  func(x, z, h float64) Vec3    // surface colour (channels 0..1)
}

// FBMTerrain builds procedural fractal hills with a height-banded palette (water,
// sand, grass, rock, snow) — a dreamlike landscape.
func FBMTerrain(seed, scale, amp float64) VoxelTerrain {
	height := func(x, z float64) float64 {
		n := 0.5 * (FBM(Vec3{X: (x + seed) * scale, Z: (z + seed) * scale}, 5, 2, 0.5) + 1)
		return clamp01(n) * amp
	}
	shade := func(x, z, h float64) Vec3 {
		n := h / amp
		switch {
		case n < 0.30:
			return Vec3{X: 0.15, Y: 0.32, Z: 0.52} // water
		case n < 0.38:
			return Vec3{X: 0.72, Y: 0.66, Z: 0.42} // sand
		case n < 0.68:
			return Vec3{X: 0.22, Y: 0.5, Z: 0.26} // grass
		case n < 0.85:
			return Vec3{X: 0.42, Y: 0.39, Z: 0.36} // rock
		default:
			return Vec3{X: 0.95, Y: 0.95, Z: 0.98} // snow
		}
	}
	return VoxelTerrain{Height: height, Shade: shade}
}

// VoxelCamera positions the voxel-space camera over the map.
type VoxelCamera struct {
	X, Z     float64 // position on the map
	Height   float64 // camera height above the terrain datum
	Yaw      float64 // view direction
	Horizon  float64 // screen row of the horizon (pixels)
	Scale    float64 // vertical projection scale
	Distance float64 // far view distance
}

// RenderVoxel draws the terrain in voxel-space style.
func RenderVoxel(t VoxelTerrain, cam VoxelCamera, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	skyTop := Vec3{X: 0.45, Y: 0.6, Z: 0.85}
	skyBot := Vec3{X: 0.82, Y: 0.86, Z: 0.95}
	enc := func(c Vec3) color.RGBA {
		return color.RGBA{R: uint8(clamp01(c.X) * 255), G: uint8(clamp01(c.Y) * 255), B: uint8(clamp01(c.Z) * 255), A: 255}
	}
	// sky gradient
	for y := 0; y < h; y++ {
		f := float64(y) / float64(h-1)
		row := enc(skyTop.Scale(1 - f).Add(skyBot.Scale(f)))
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, row)
		}
	}
	if cam.Distance <= 0 {
		cam.Distance = 600
	}
	if cam.Scale <= 0 {
		cam.Scale = float64(h)
	}
	ybuf := make([]int, w)
	for i := range ybuf {
		ybuf[i] = h
	}
	sinp, cosp := math.Sin(cam.Yaw), math.Cos(cam.Yaw)
	dz := 1.0
	for z := 1.0; z < cam.Distance; {
		// the sample line at depth z spans the 90-degree frustum, left to right.
		plx := (-cosp*z - sinp*z) + cam.X
		ply := (sinp*z - cosp*z) + cam.Z
		prx := (cosp*z - sinp*z) + cam.X
		pry := (-sinp*z - cosp*z) + cam.Z
		dx := (prx - plx) / float64(w)
		dy := (pry - ply) / float64(w)
		fog := clamp01(z / cam.Distance)
		skyHere := skyBot
		for i := 0; i < w; i++ {
			hgt := t.Height(plx, ply)
			screenY := int((cam.Height-hgt)/z*cam.Scale + cam.Horizon)
			if screenY < ybuf[i] {
				col := t.Shade(plx, ply, hgt).Scale(1 - 0.85*fog).Add(skyHere.Scale(0.85 * fog))
				rc := enc(col)
				top := screenY
				if top < 0 {
					top = 0
				}
				for y := top; y < ybuf[i] && y < h; y++ {
					if y >= 0 {
						img.SetRGBA(i, y, rc)
					}
				}
				ybuf[i] = screenY
			}
			plx += dx
			ply += dy
		}
		z += dz
		dz += 0.015 // level of detail: longer steps farther away
	}
	return img
}
