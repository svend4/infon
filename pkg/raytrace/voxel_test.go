package raytrace

import (
	"image"
	"math"
	"testing"
)

func TestFBMTerrain(t *testing.T) {
	tr := FBMTerrain(10, 0.01, 80)
	h1 := tr.Height(12, 34)
	if tr.Height(12, 34) != h1 {
		t.Error("terrain height should be deterministic")
	}
	if h1 < 0 || h1 > 80 {
		t.Errorf("height should be within [0,amp], got %.2f", h1)
	}
	c := tr.Shade(12, 34, h1)
	if c.X < 0 || c.X > 1 {
		t.Errorf("shade should be a valid colour, got %+v", c)
	}
}

func TestRenderVoxelDims(t *testing.T) {
	img := RenderVoxel(FBMTerrain(1, 0.01, 70), VoxelCamera{Height: 100, Horizon: 80, Scale: 160, Distance: 300}, 200, 160)
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 160 {
		t.Fatalf("unexpected size %v", img.Bounds())
	}
}

// skyAtRow is the background gradient colour used by RenderVoxel at row y.
func skyAtRow(y, h int) Vec3 {
	f := float64(y) / float64(h-1)
	return Vec3{X: 0.45, Y: 0.6, Z: 0.85}.Scale(1 - f).Add(Vec3{X: 0.82, Y: 0.86, Z: 0.95}.Scale(f))
}

// topTerrainY is the highest screen row in a column whose colour departs from the
// sky gradient (i.e. where the terrain silhouette begins), or h if none.
func topTerrainY(img image.Image, col, h int) int {
	for y := 0; y < h; y++ {
		r, g, b, _ := img.At(col, y).RGBA()
		c := Vec3{X: float64(r) / 65535, Y: float64(g) / 65535, Z: float64(b) / 65535}
		if c.Sub(skyAtRow(y, h)).Len() > 0.12 {
			return y
		}
	}
	return h
}

func TestRenderVoxelNotBlank(t *testing.T) {
	w, h := 200, 160
	img := RenderVoxel(FBMTerrain(3, 0.012, 70), VoxelCamera{Height: 90, Horizon: 70, Scale: 160, Distance: 320}, w, h)
	if topTerrainY(img, w/2, h) >= h {
		t.Error("the voxel render should draw terrain, not only sky")
	}
}

// A hill straight ahead raises the terrain silhouette versus flat ground.
func TestVoxelHillRaisesSilhouette(t *testing.T) {
	w, h := 200, 160
	green := func(x, z, hh float64) Vec3 { return Vec3{X: 0.2, Y: 0.5, Z: 0.2} }
	flat := VoxelTerrain{Height: func(x, z float64) float64 { return 0 }, Shade: green}
	hill := VoxelTerrain{Height: func(x, z float64) float64 {
		return math.Max(0, 70-math.Hypot(x, z+100)) // a peak ahead (yaw 0 looks toward -Z)
	}, Shade: green}
	cam := VoxelCamera{X: 0, Z: 0, Height: 90, Yaw: 0, Horizon: 80, Scale: 160, Distance: 200}

	flatTop := topTerrainY(RenderVoxel(flat, cam, w, h), w/2, h)
	hillTop := topTerrainY(RenderVoxel(hill, cam, w, h), w/2, h)
	if hillTop >= flatTop {
		t.Errorf("a hill ahead should rise higher on screen (smaller y): hill %d flat %d", hillTop, flatTop)
	}
}
