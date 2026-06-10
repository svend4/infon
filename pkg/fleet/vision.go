package fleet

import "image"

// vision.go is the camera front-end: it reads a frame and extracts robot-health
// signatures, the way info150's triage4-drive reads eye-state / gaze / posture
// from a cab-mounted camera. The pipeline is then identical to a sensor reading —
// SignalsFromImage feeds straight into MonitoringEngine.Assess — so a camera and a
// telemetry stream are the same to the engine (the variant-E "one engine, any
// input"). Pure and deterministic.

// SignalsFromImage extracts four signatures from a camera frame, each 0..1 where
// higher is more concerning:
//
//	thermal      redness of the frame (a hot spot / overheating motor)
//	darkness     fraction of near-black pixels (an obscured / occluded view)
//	overexposure fraction of near-white pixels (glare blinding the camera)
//	blur         1 - edge energy (out-of-focus or motion-blurred / vibrating)
//
// The frame is downsampled for speed, so cost is bounded regardless of resolution.
func SignalsFromImage(img image.Image) []Signal {
	b := img.Bounds()
	w, hgt := b.Dx(), b.Dy()
	if w == 0 || hgt == 0 {
		return nil
	}
	step := 1
	if m := max(w, hgt); m > 160 {
		step = m / 160
	}
	var nTot, nDark, nWhite, nHot, edgeN int
	var redSum, edgeSum float64
	for y := b.Min.Y; y < b.Max.Y; y += step {
		prev := -1.0
		for x := b.Min.X; x < b.Max.X; x += step {
			r16, g16, b16, _ := img.At(x, y).RGBA()
			r, g, bl := float64(r16>>8), float64(g16>>8), float64(b16>>8)
			luma := (0.299*r + 0.587*g + 0.114*bl) / 255
			nTot++
			if luma < 0.12 {
				nDark++
			}
			if luma > 0.95 {
				nWhite++
			}
			red := (r - max(g, bl)) / 255
			if red < 0 {
				red = 0
			}
			redSum += red
			if red > 0.22 { // a warm/orange pixel counts as a hot spot
				nHot++
			}
			if prev >= 0 {
				d := luma - prev
				if d < 0 {
					d = -d
				}
				edgeSum += d
				edgeN++
			}
			prev = luma
		}
	}
	meanRed := redSum / float64(nTot)
	hotFrac := float64(nHot) / float64(nTot)
	edge := 0.0
	if edgeN > 0 {
		edge = edgeSum / float64(edgeN)
	}
	// thermal: subtract a small ambient-warmth floor, then amplify, so a genuinely
	// hot region stands out from the scene's background colour temperature.
	thermal := clamp01((meanRed*2.5 + hotFrac*3 - 0.15) * 3)
	return []Signal{
		{Name: "thermal", Value: thermal, Weight: 1.0},
		{Name: "darkness", Value: float64(nDark) / float64(nTot), Weight: 0.8},
		{Name: "overexposure", Value: float64(nWhite) / float64(nTot), Weight: 0.6},
		{Name: "blur", Value: clamp01(1 - edge/0.03), Weight: 0.4},
	}
}

// SeverityColor maps a severity 0..1 to green (calm) -> amber -> red (critical) —
// the exported form of the colour used for status lights and frame borders.
func SeverityColor(s float64) [3]float64 { return severityColor(s) }
