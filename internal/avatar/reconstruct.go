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

// Reconstructor turns received keypoints (+ a one-time reference face) back into
// an image, via an external generative model over HTTP (the receiver side of
// C5). Contract:
//
//	POST {url}  JSON {"reference":"<base64 png>", "points":[[x,y]...],
//	                  "yaw":..,"pitch":..,"roll":..}
//	-> raw image bytes (the reconstructed face)
//
// The reference image is sent once and cached server-side by the model, but we
// include it for statelessness; a real deployment would send it only on the
// first frame.
type Reconstructor struct {
	url    string
	refB64 string
	http   *http.Client
}

// NewReconstructor creates a receiver-side face reconstructor. ref is the
// one-time reference face image; may be nil (the model may have its own).
func NewReconstructor(url string, ref image.Image) (*Reconstructor, error) {
	r := &Reconstructor{url: url, http: &http.Client{Timeout: 30 * time.Second}}
	if ref != nil {
		var buf bytes.Buffer
		if err := png.Encode(&buf, ref); err != nil {
			return nil, err
		}
		r.refB64 = base64.StdEncoding.EncodeToString(buf.Bytes())
	}
	return r, nil
}

// Reconstruct asks the model to render the face described by kp.
func (r *Reconstructor) Reconstruct(ctx context.Context, kp Keypoints) (image.Image, error) {
	pts := make([][2]float32, len(kp.Points))
	copy(pts, kp.Points)
	reqBody, err := json.Marshal(map[string]any{
		"reference": r.refB64,
		"points":    pts,
		"yaw":       kp.Yaw,
		"pitch":     kp.Pitch,
		"roll":      kp.Roll,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("avatar reconstruct: status %d", resp.StatusCode)
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	return img, err
}
