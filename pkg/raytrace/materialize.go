// materialize.go reproduces the moment a dream forms, as the hackers describe it:
// first the world is coarse, "large rectangular pixel-blocks", then "everything
// adjusts itself" and resolves into a normal image. Pixelate gives the blocky
// look; Materialize sweeps from heavily blocked (t=0) to sharp (t=1), so a frame
// can be made to "render in" from voxels — the dream materialising.
package raytrace

import "image"

// Pixelate averages the image into block×block tiles (a voxel/mosaic look).
func Pixelate(img image.Image, block int) image.Image {
	src, w, h := imgToBuf(img)
	if block < 1 {
		block = 1
	}
	out := make([]Vec3, w*h)
	for by := 0; by < h; by += block {
		for bx := 0; bx < w; bx += block {
			var sum Vec3
			n := 0.0
			for y := by; y < by+block && y < h; y++ {
				for x := bx; x < bx+block && x < w; x++ {
					sum = sum.Add(src[y*w+x])
					n++
				}
			}
			avg := sum.Scale(1 / n)
			for y := by; y < by+block && y < h; y++ {
				for x := bx; x < bx+block && x < w; x++ {
					out[y*w+x] = avg
				}
			}
		}
	}
	return bufToImg(out, w, h)
}

// maxMaterializeBlock is how coarse the world is at the very start of forming.
const maxMaterializeBlock = 24

// Materialize sweeps a frame from coarse voxel blocks (t=0) to sharp (t=1) — the
// dream rendering itself in.
func Materialize(img image.Image, t float64) image.Image {
	t = clamp01(t)
	block := int(float64(maxMaterializeBlock)*(1-t) + 0.5)
	if block <= 1 {
		return img
	}
	return Pixelate(img, block)
}
