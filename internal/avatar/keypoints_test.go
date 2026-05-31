package avatar

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	gocolor "image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func sampleKeypoints(n int) Keypoints {
	pts := make([][2]float32, n)
	for i := 0; i < n; i++ {
		pts[i] = [2]float32{float32(i) / float32(n), float32(i%7) / 7.0}
	}
	return Keypoints{Frame: 42, Points: pts, Yaw: 0.1, Pitch: -0.2, Roll: 0.05}
}

func TestKeypointsRoundTrip(t *testing.T) {
	k := sampleKeypoints(68)
	data := k.Encode()
	got, err := DecodeKeypoints(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Frame != k.Frame || len(got.Points) != len(k.Points) {
		t.Fatalf("header mismatch: %+v", got)
	}
	// Quantization is 16-bit, so coords should match within ~1/65535.
	for i := range k.Points {
		if math.Abs(float64(got.Points[i][0]-k.Points[i][0])) > 1e-4 ||
			math.Abs(float64(got.Points[i][1]-k.Points[i][1])) > 1e-4 {
			t.Fatalf("point %d drift: got %v want %v", i, got.Points[i], k.Points[i])
		}
	}
	// Pose floats survive exactly (stored as float32 bits).
	if got.Yaw != k.Yaw || got.Pitch != k.Pitch || got.Roll != k.Roll {
		t.Errorf("pose mismatch: %v/%v/%v", got.Yaw, got.Pitch, got.Roll)
	}
}

func TestKeypointsEncodedSize(t *testing.T) {
	k := sampleKeypoints(68)
	if len(k.Encode()) != EncodedSize(68) {
		t.Errorf("size mismatch: %d vs %d", len(k.Encode()), EncodedSize(68))
	}
	// The headline C5 claim: a 68-point frame is tiny (< 300 bytes).
	if EncodedSize(68) > 300 {
		t.Errorf("68-point frame = %d bytes, expected < 300", EncodedSize(68))
	}
}

func TestKeypointsVsVideoBandwidth(t *testing.T) {
	// Compare keypoints to a modest pixel frame (80x24 cells * 10 bytes).
	kpBytes := EncodedSize(68)
	videoBytes := 80 * 24 * 10
	ratio := float64(videoBytes) / float64(kpBytes)
	t.Logf("keypoints=%dB video=%dB ratio=%.0fx", kpBytes, videoBytes, ratio)
	if ratio < 50 {
		t.Errorf("keypoints should be >=50x smaller than block video, got %.0fx", ratio)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, err := DecodeKeypoints([]byte{1, 2, 3}); err != ErrBadKeypoints {
		t.Errorf("short data: got %v, want ErrBadKeypoints", err)
	}
	// Right length header but wrong magic.
	bad := make([]byte, 22)
	if _, err := DecodeKeypoints(bad); err != ErrBadKeypoints {
		t.Errorf("bad magic: got %v, want ErrBadKeypoints", err)
	}
	// Valid magic but truncated point data.
	k := sampleKeypoints(10).Encode()
	if _, err := DecodeKeypoints(k[:len(k)-3]); err != ErrBadKeypoints {
		t.Errorf("truncated: got %v, want ErrBadKeypoints", err)
	}
}

func TestQuantizeClamps(t *testing.T) {
	if dequantize(quantize(-5)) != 0 {
		t.Error("negative should quantize to 0")
	}
	if dequantize(quantize(5)) != 1 {
		t.Error("over-1 should quantize to 1")
	}
}

func facePNG(t *testing.T) image.Image {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{R: 200, G: 180, B: 160, A: 255})
		}
	}
	return img
}

func TestReconstructorCallsModel(t *testing.T) {
	// Sidecar returns a solid image; verify the request carried points + pose.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = jsonDecode(r, &body)
		if _, ok := body["points"]; !ok {
			t.Error("request missing points")
		}
		if _, ok := body["reference"]; !ok {
			t.Error("request missing reference")
		}
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		var buf bytes.Buffer
		_ = png.Encode(&buf, img)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	rc, err := NewReconstructor(srv.URL, facePNG(t))
	if err != nil {
		t.Fatalf("new reconstructor: %v", err)
	}
	img, err := rc.Reconstruct(context.Background(), sampleKeypoints(68))
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}
	if img.Bounds().Dx() != 16 {
		t.Errorf("reconstructed image = %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestReconstructorHandlesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	rc, _ := NewReconstructor(srv.URL, nil)
	if _, err := rc.Reconstruct(context.Background(), sampleKeypoints(5)); err == nil {
		t.Error("expected error on HTTP 500")
	}
}

// jsonDecode reads a JSON body from a request.
func jsonDecode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
