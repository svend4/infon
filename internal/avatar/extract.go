package avatar

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"time"
)

// Extractor is the SENDER side of neural avatars: it turns a camera frame into
// compact face keypoints via an external landmark model over HTTP (the
// avatar_landmark sidecar, or a cloud face-mesh API). Contract:
//
//	POST {url}  JSON {"image":"<base64 png>"}
//	-> JSON {"points":[[x,y]...], "yaw":F,"pitch":F,"roll":F}
//
// coords are normalized 0..1. The result is packed via Keypoints.Encode (~294
// bytes) for transmission instead of video.
type Extractor struct {
	url  string
	http *http.Client
}

// NewExtractor creates a landmark extractor pointed at url.
func NewExtractor(url string) *Extractor {
	return &Extractor{url: url, http: &http.Client{Timeout: 15 * time.Second}}
}

// Extract returns the face keypoints for a frame, tagged with frameNum.
func (e *Extractor) Extract(ctx context.Context, img image.Image, frameNum uint32) (Keypoints, error) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return Keypoints{}, err
	}
	reqBody, err := json.Marshal(map[string]any{
		"image": base64.StdEncoding.EncodeToString(pngBuf.Bytes()),
	})
	if err != nil {
		return Keypoints{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(reqBody))
	if err != nil {
		return Keypoints{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return Keypoints{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return Keypoints{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Keypoints{}, fmt.Errorf("avatar extract: status %d", resp.StatusCode)
	}

	var out struct {
		Points           [][2]float32 `json:"points"`
		Yaw, Pitch, Roll float32
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return Keypoints{}, err
	}
	return Keypoints{
		Frame:  frameNum,
		Points: out.Points,
		Yaw:    out.Yaw,
		Pitch:  out.Pitch,
		Roll:   out.Roll,
	}, nil
}
