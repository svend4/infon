package fleet

import (
	"image"
	"image/color"
	"testing"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func sigVal(sigs []Signal, name string) float64 {
	for _, s := range sigs {
		if s.Name == name {
			return s.Value
		}
	}
	return -1
}

func TestVisionRedIsHot(t *testing.T) {
	s := SignalsFromImage(solid(64, 64, color.RGBA{R: 230, G: 20, B: 20, A: 255}))
	if v := sigVal(s, "thermal"); v < 0.7 {
		t.Errorf("a red frame should read hot, thermal=%.2f", v)
	}
	if v := sigVal(s, "darkness"); v > 0.1 {
		t.Errorf("a red frame is not dark, darkness=%.2f", v)
	}
}

func TestVisionBlackIsDark(t *testing.T) {
	s := SignalsFromImage(solid(64, 64, color.RGBA{A: 255}))
	if v := sigVal(s, "darkness"); v < 0.9 {
		t.Errorf("a black frame should read dark, darkness=%.2f", v)
	}
	if v := sigVal(s, "thermal"); v > 0.05 {
		t.Errorf("a black frame is not hot, thermal=%.2f", v)
	}
}

func TestVisionWhiteIsOverexposed(t *testing.T) {
	s := SignalsFromImage(solid(64, 64, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
	if v := sigVal(s, "overexposure"); v < 0.9 {
		t.Errorf("a white frame should read over-exposed, overexposure=%.2f", v)
	}
}

func TestVisionFlatIsBlurryStripesAreSharp(t *testing.T) {
	flat := SignalsFromImage(solid(64, 64, color.RGBA{R: 120, G: 120, B: 120, A: 255}))
	if v := sigVal(flat, "blur"); v < 0.8 {
		t.Errorf("a flat frame has no edges (blurry), blur=%.2f", v)
	}
	// vertical stripes: strong horizontal edges -> sharp (low blur)
	stripes := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			c := color.RGBA{A: 255}
			if x%2 == 0 {
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			}
			stripes.SetRGBA(x, y, c)
		}
	}
	if v := sigVal(SignalsFromImage(stripes), "blur"); v > 0.2 {
		t.Errorf("a striped frame is sharp (low blur), blur=%.2f", v)
	}
}

func TestVisionFeedsEngine(t *testing.T) {
	// a hot frame, assessed, should not be nominal.
	sigs := SignalsFromImage(solid(48, 48, color.RGBA{R: 240, G: 10, B: 10, A: 255}))
	a := NewMonitoringEngine().Assess(Reading{Unit: "cam", Signals: sigs})
	if a.Level == LevelOK {
		t.Errorf("a hot camera frame should raise the level, got OK (sev %.2f)", a.Severity)
	}
	if a.Worst != "thermal" {
		t.Errorf("the dominant concern of a red frame should be thermal, got %q", a.Worst)
	}
}
