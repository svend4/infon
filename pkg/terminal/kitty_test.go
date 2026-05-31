package terminal

import (
	"encoding/base64"
	"image"
	gocolor "image/color"
	"strings"
	"testing"
)

func TestKittyEmptyImage(t *testing.T) {
	if EncodeKitty(image.NewRGBA(image.Rect(0, 0, 0, 0))) != "" {
		t.Error("empty image should encode to empty string")
	}
}

func TestKittyWrapsInAPC(t *testing.T) {
	out := EncodeKitty(solidImg(4, 4, gocolor.RGBA{R: 255, A: 255}))
	if !strings.HasPrefix(out, "\x1b_G") {
		t.Error("kitty output should start with APC graphics intro ESC _ G")
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("kitty output should end with ST (ESC \\)")
	}
}

func TestKittyHeaderKeys(t *testing.T) {
	out := EncodeKitty(solidImg(8, 5, gocolor.RGBA{G: 255, A: 255}))
	// First chunk must declare action, format and dimensions.
	if !strings.Contains(out, "a=T") {
		t.Error("missing action a=T")
	}
	if !strings.Contains(out, "f=24") {
		t.Error("missing format f=24 (RGB)")
	}
	if !strings.Contains(out, "s=8") || !strings.Contains(out, "v=5") {
		t.Errorf("missing/incorrect dimensions s=8,v=5 in: %s", headPrefix(out))
	}
}

func TestKittyPayloadDecodesToRGB(t *testing.T) {
	// A 2x1 image of red,green should encode to base64 of exactly 6 RGB bytes.
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, gocolor.RGBA{R: 255, A: 255})
	img.SetRGBA(1, 0, gocolor.RGBA{G: 255, A: 255})
	out := EncodeKitty(img)

	// Extract the payload after the first ';' and before the trailing ST.
	semi := strings.Index(out, ";")
	st := strings.Index(out, "\x1b\\")
	if semi < 0 || st < 0 {
		t.Fatal("malformed kitty sequence")
	}
	payload := out[semi+1 : st]
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload not valid base64: %v", err)
	}
	if len(raw) != 6 {
		t.Fatalf("decoded %d bytes, want 6 (2 px * RGB)", len(raw))
	}
	// First pixel red, second green.
	if raw[0] != 255 || raw[1] != 0 || raw[2] != 0 {
		t.Errorf("pixel 0 = %v, want red", raw[0:3])
	}
	if raw[3] != 0 || raw[4] != 255 || raw[5] != 0 {
		t.Errorf("pixel 1 = %v, want green", raw[3:6])
	}
}

func TestKittyChunksLargeImage(t *testing.T) {
	// A big image must split into multiple APC chunks, all but the last with m=1.
	out := EncodeKitty(solidImg(64, 64, gocolor.RGBA{B: 255, A: 255}))
	chunks := strings.Count(out, "\x1b_G")
	if chunks < 2 {
		t.Errorf("expected multiple chunks for a 64x64 image, got %d", chunks)
	}
	if !strings.Contains(out, "m=1") {
		t.Error("multi-chunk output should mark continuation with m=1")
	}
	// The final chunk must close the sequence (m=0 somewhere after an m=1).
	if !strings.Contains(out, "m=0") {
		t.Error("final chunk should carry m=0")
	}
}

func headPrefix(s string) string {
	if len(s) > 60 {
		return s[:60]
	}
	return s
}
