package raydir

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func solidImg(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func hasColor(img image.Image, want color.RGBA, tol uint32) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			dr := absU(r>>8, uint32(want.R))
			dg := absU(g>>8, uint32(want.G))
			db := absU(bl>>8, uint32(want.B))
			if dr <= tol && dg <= tol && db <= tol {
				return true
			}
		}
	}
	return false
}

func absU(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestThumbnail(t *testing.T) {
	src := solidImg(100, 80, color.RGBA{R: 220, G: 20, B: 20, A: 255})
	th := Thumbnail(src, 10, 8)
	if th.Bounds().Dx() != 10 || th.Bounds().Dy() != 8 {
		t.Fatalf("thumbnail should be 10x8, got %v", th.Bounds())
	}
	if !hasColor(th, color.RGBA{R: 220, G: 20, B: 20}, 8) {
		t.Error("a red source should make a red thumbnail")
	}
}

func TestTravelogueCaptureCap(t *testing.T) {
	tl := NewTravelogue("Trip")
	for i := 0; i < 30; i++ {
		tl.Capture(fmt.Sprintf("%d", i), 0.5, solidImg(8, 8, color.RGBA{A: 255}))
	}
	if tl.Len() != 24 {
		t.Errorf("travelogue should cap at 24, got %d", tl.Len())
	}
	if tl.Moments[0].Place != "6" || tl.Moments[len(tl.Moments)-1].Place != "29" {
		t.Errorf("should keep the most recent 24 (6..29), got %s..%s", tl.Moments[0].Place, tl.Moments[len(tl.Moments)-1].Place)
	}
}

func TestClockStr(t *testing.T) {
	if got := clockStr(0.0); got != "00:00" {
		t.Errorf("clockStr(0) = %q", got)
	}
	if got := clockStr(0.5); got != "12:00" {
		t.Errorf("clockStr(0.5) = %q", got)
	}
}

func TestTravelogueRender(t *testing.T) {
	tl := NewTravelogue("My Walk")
	tl.Capture("Forest", 0.3, solidImg(40, 30, color.RGBA{R: 30, G: 200, B: 60, A: 255}))
	tl.Capture("Shore", 0.6, solidImg(40, 30, color.RGBA{R: 40, G: 90, B: 220, A: 255}))
	page := tl.Render(2)
	if page.Bounds().Dx() < 80 || page.Bounds().Dy() < 30 {
		t.Fatalf("the page should be sized for two cells, got %v", page.Bounds())
	}
	if !hasColor(page, color.RGBA{R: 30, G: 200, B: 60}, 12) {
		t.Error("the page should contain the forest thumbnail's green")
	}
	if !hasColor(page, color.RGBA{R: 40, G: 90, B: 220}, 12) {
		t.Error("the page should contain the shore thumbnail's blue")
	}
}
