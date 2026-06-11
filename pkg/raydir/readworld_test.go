package raydir

import (
	"image"
	"image/color"
	"testing"

	"github.com/svend4/infon/internal/vision"
)

func solid(w, h int, r, g, b uint8) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

// halfBright is half highlight (near white) and half mid grey, for the glow test.
func halfBright(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{R: 120, G: 120, B: 120, A: 255}
			if y < h/2 {
				c = color.RGBA{R: 252, G: 252, B: 252, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestReadImageWarmth(t *testing.T) {
	warm := ReadImage(solid(40, 40, 220, 120, 60), nil)
	cold := ReadImage(solid(40, 40, 60, 120, 220), nil)
	if warm[AxWarm] < 0.6 {
		t.Errorf("a red image should read warm, got %.2f", warm[AxWarm])
	}
	if cold[AxWarm] > 0.4 {
		t.Errorf("a blue image should read cool, got %.2f", cold[AxWarm])
	}
	if !(warm[AxWarm] > cold[AxWarm]) {
		t.Error("warm should exceed cool")
	}
}

func TestReadImageBrightness(t *testing.T) {
	bright := ReadImage(solid(40, 40, 240, 240, 240), nil)
	dark := ReadImage(solid(40, 40, 18, 18, 18), nil)
	if bright[AxSun] < 0.8 {
		t.Errorf("a near-white image should read sunny, got %.2f", bright[AxSun])
	}
	if dark[AxSun] > 0.2 {
		t.Errorf("a near-black image should read dark, got %.2f", dark[AxSun])
	}
}

func TestReadImageFog(t *testing.T) {
	grey := ReadImage(solid(40, 40, 128, 128, 128), nil) // no saturation -> fog
	vivid := ReadImage(solid(40, 40, 220, 30, 30), nil)  // saturated -> clear
	if grey[AxFog] < 0.5 {
		t.Errorf("a flat grey image should read foggy, got %.2f", grey[AxFog])
	}
	if !(grey[AxFog] > vivid[AxFog]) {
		t.Errorf("grey should read foggier than vivid: %.2f vs %.2f", grey[AxFog], vivid[AxFog])
	}
}

func TestReadImageGlow(t *testing.T) {
	glowy := ReadImage(halfBright(40, 40), nil)
	flat := ReadImage(solid(40, 40, 100, 100, 100), nil)
	if glowy[AxGlow] <= flat[AxGlow] {
		t.Errorf("highlights should read as glow: %.2f vs %.2f", glowy[AxGlow], flat[AxGlow])
	}
}

func TestReadImageDensityScale(t *testing.T) {
	img := solid(16, 16, 128, 128, 128)
	none := ReadImage(img, nil)
	if none[AxDensity] != 0.5 || none[AxScale] != 0.5 {
		t.Errorf("with no detections density/scale stay neutral, got %.2f/%.2f", none[AxDensity], none[AxScale])
	}
	few := []vision.Detection{{W: 0.1, H: 0.1}, {W: 0.1, H: 0.1}}
	many := make([]vision.Detection, 8)
	for i := range many {
		many[i] = vision.Detection{W: 0.4, H: 0.4}
	}
	if r := ReadImage(img, few); r[AxDensity] >= ReadImage(img, many)[AxDensity] {
		t.Error("more detections should read as higher density")
	}
	if r := ReadImage(img, few); ReadImage(img, many)[AxScale] <= r[AxScale] {
		t.Error("larger detections should read as larger scale")
	}
}

func TestReadImageMood(t *testing.T) {
	warmBright := ReadImage(solid(40, 40, 240, 190, 110), nil).Mood()
	coldDark := ReadImage(solid(40, 40, 30, 40, 80), nil).Mood()
	positive := map[string]bool{"joyful": true, "serene": true}
	negative := map[string]bool{"somber": true, "ominous": true}
	if !positive[warmBright] {
		t.Errorf("a warm bright image should read positive, got %q", warmBright)
	}
	if !negative[coldDark] {
		t.Errorf("a cold dark image should read negative, got %q", coldDark)
	}
}
