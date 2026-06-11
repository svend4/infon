package raydir

import (
	"image"
	"math"

	"github.com/svend4/infon/internal/vision"
)

// readworld.go is the reader pointed at the REAL world. Block C's ReadScene recovers
// the six Q6 coordinates from a scene the director authored; ReadImage recovers them
// from raw pixels — a photo or a camera frame — plus, when a vision model supplies
// them, detected objects. Colour and light come straight from the image (warmth from
// the red/blue balance, sun from luminance, fog from low saturation and contrast,
// glow from highlights); density and scale come from how many things are present and
// how large. From the vector follow the hexagram and the mood (SceneVector.Mood), so
// you can ask "what hexagram is this real scene, and how does it feel?".

// imageStats samples an image and returns its mean luminance, mean R/G/B, mean HSV
// saturation, luminance standard deviation, and the fraction of bright highlights.
func imageStats(img image.Image) (meanL, meanR, meanG, meanB, meanSat, stdL, brightFrac float64) {
	b := img.Bounds()
	if b.Empty() {
		return
	}
	// subsample to at most ~160×160 points so large images read fast.
	stepX := max1(b.Dx() / 160)
	stepY := max1(b.Dy() / 160)
	var n float64
	var sumL2 float64
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			r16, g16, bl16, _ := img.At(x, y).RGBA()
			r := float64(r16) / 65535
			g := float64(g16) / 65535
			bb := float64(bl16) / 65535
			l := 0.299*r + 0.587*g + 0.114*bb
			meanR += r
			meanG += g
			meanB += bb
			meanL += l
			sumL2 += l * l
			mx := math.Max(r, math.Max(g, bb))
			mn := math.Min(r, math.Min(g, bb))
			if mx > 0 {
				meanSat += (mx - mn) / mx
			}
			if l > 0.85 {
				brightFrac++
			}
			n++
		}
	}
	if n == 0 {
		return
	}
	meanR, meanG, meanB = meanR/n, meanG/n, meanB/n
	meanL, meanSat, brightFrac = meanL/n, meanSat/n, brightFrac/n
	stdL = math.Sqrt(math.Max(0, sumL2/n-meanL*meanL))
	return
}

// ReadImage infers the six Q6 coordinates from a real image. Pass detections from a
// vision model for density/scale (nil leaves them neutral at 0.5, since raw pixels
// alone do not count discrete objects).
func ReadImage(img image.Image, dets []vision.Detection) SceneVector {
	meanL, meanR, _, meanB, meanSat, _, brightFrac := imageStats(img)
	var v SceneVector
	v[AxSun] = clampf((meanL-0.12)/0.6, 0, 1)       // brightness ~ day
	v[AxWarm] = clampf(0.5+(meanR-meanB)*2.2, 0, 1) // red over blue = warm
	satTerm := clampf(1-meanSat/0.35, 0, 1)         // washed-out colour reads as fog
	bright := clampf((meanL-0.55)/0.4, 0, 1)        // ...but a bright scene is not fog
	v[AxFog] = clampf(satTerm*(1-0.7*bright), 0, 1)
	v[AxGlow] = clampf(brightFrac/0.06, 0, 1) // highlights glow
	if len(dets) > 0 {
		v[AxDensity] = clampf(float64(len(dets))/8, 0, 1)
		area := 0.0
		for _, d := range dets {
			area += d.W * d.H
		}
		v[AxScale] = clampf(math.Sqrt(area/float64(len(dets)))*2.2, 0, 1)
	} else {
		v[AxDensity], v[AxScale] = 0.5, 0.5
	}
	return v
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}
