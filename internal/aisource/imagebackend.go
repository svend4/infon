// ImageBackend connects a real text-to-image model (a "raster" backend) to the
// device.NeuralBackend seam. Where BrainBackend asks a tvcp-ai/1 brain for a
// vector *sketch* and rasterizes it, ImageBackend asks an image-generation HTTP
// API for an actual picture (PNG/JPEG) and decodes it — the genuine "describe
// something and the model paints it" path for visual communication.
//
// It is provider-agnostic and stdlib-only. Configure it with a request/response
// shape that fits common APIs:
//   - request: JSON {"prompt":..,"width":..,"height":..} (field names overridable)
//   - response: either raw image bytes (Content-Type image/png|jpeg) OR JSON
//     carrying base64 image data (OpenAI-style {"data":[{"b64_json":".."}]},
//     or a custom field path).
package aisource

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/svend4/infon/internal/device"
)

// ImageConfig describes how to talk to a text-to-image endpoint.
type ImageConfig struct {
	URL    string            // POST endpoint
	APIKey string            // optional bearer token
	Header map[string]string // extra static headers (e.g. provider versioning)

	// PromptField/WidthField/HeightField name the JSON request fields. Empty
	// width/height fields are omitted from the request.
	PromptField string
	WidthField  string
	HeightField string
	Extra       map[string]any // extra static request fields (model, steps, ...)

	// B64Path is a dot-path into a JSON response locating base64 image data
	// (e.g. "data.0.b64_json"). Empty means: treat the response body as raw
	// image bytes. Ignored when the response Content-Type is an image type.
	B64Path string

	Timeout time.Duration
}

// ImageBackend is a device.NeuralBackend that fetches a rastered image per prompt.
type ImageBackend struct {
	cfg  ImageConfig
	http *http.Client
}

var _ device.NeuralBackend = (*ImageBackend)(nil)

// NewImageBackend creates a backend from an explicit config.
func NewImageBackend(cfg ImageConfig) *ImageBackend {
	if cfg.PromptField == "" {
		cfg.PromptField = "prompt"
	}
	if cfg.WidthField == "" {
		cfg.WidthField = "width"
	}
	if cfg.HeightField == "" {
		cfg.HeightField = "height"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &ImageBackend{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// NewImageBackendFromEnv builds a backend from environment variables, or returns
// (nil, false) if IMAGE_API_URL is unset. This is the zero-config path for the
// CLI:
//
//	IMAGE_API_URL   (required) POST endpoint
//	IMAGE_API_KEY   (optional) bearer token
//	IMAGE_B64_PATH  (optional) dot-path to base64 data, e.g. data.0.b64_json
//	IMAGE_PROMPT_FIELD / IMAGE_WIDTH_FIELD / IMAGE_HEIGHT_FIELD (optional)
func NewImageBackendFromEnv() (*ImageBackend, bool) {
	url := os.Getenv("IMAGE_API_URL")
	if url == "" {
		return nil, false
	}
	return NewImageBackend(ImageConfig{
		URL:         url,
		APIKey:      os.Getenv("IMAGE_API_KEY"),
		B64Path:     os.Getenv("IMAGE_B64_PATH"),
		PromptField: os.Getenv("IMAGE_PROMPT_FIELD"),
		WidthField:  os.Getenv("IMAGE_WIDTH_FIELD"),
		HeightField: os.Getenv("IMAGE_HEIGHT_FIELD"),
	}), true
}

// Name implements device.NeuralBackend.
func (b *ImageBackend) Name() string { return "image-api" }

// Generate implements device.NeuralBackend: POST the prompt, decode the returned
// image. The image may be any size; the BABE renderer downsamples it later.
func (b *ImageBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	reqBody := map[string]any{b.cfg.PromptField: prompt}
	if b.cfg.WidthField != "" && width > 0 {
		reqBody[b.cfg.WidthField] = width
	}
	if b.cfg.HeightField != "" && height > 0 {
		reqBody[b.cfg.HeightField] = height
	}
	for k, v := range b.cfg.Extra {
		reqBody[k] = v
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.cfg.APIKey)
	}
	for k, v := range b.cfg.Header {
		req.Header.Set(k, v)
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20)) // 64 MiB cap
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image api: status %d: %s", resp.StatusCode, snippet(body))
	}

	raw, err := b.extractImageBytes(resp.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("image api: decode: %w", err)
	}
	return img, nil
}

// extractImageBytes returns raw image bytes from the response, handling three
// shapes: an image Content-Type (raw), a configured base64 JSON path, or a raw
// body fallback.
func (b *ImageBackend) extractImageBytes(contentType string, body []byte) ([]byte, error) {
	if strings.HasPrefix(contentType, "image/") {
		return body, nil
	}
	if b.cfg.B64Path != "" {
		var doc any
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, fmt.Errorf("image api: response is not JSON: %w", err)
		}
		s, err := lookupString(doc, b.cfg.B64Path)
		if err != nil {
			return nil, fmt.Errorf("image api: %w (body: %s)", err, snippet(body))
		}
		// Tolerate data URLs ("data:image/png;base64,....").
		if i := strings.Index(s, ","); strings.HasPrefix(s, "data:") && i >= 0 {
			s = s[i+1:]
		}
		dec, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("image api: base64 decode: %w", err)
		}
		return dec, nil
	}
	// No content-type hint and no path: assume the body is the image itself.
	return body, nil
}

// lookupString walks a dot-path through decoded JSON (objects and arrays) and
// returns the string at the end. Array indices are numeric path segments.
func lookupString(doc any, path string) (string, error) {
	cur := doc
	for _, seg := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[seg]
			if !ok {
				return "", fmt.Errorf("path %q: missing key %q", path, seg)
			}
			cur = v
		case []any:
			idx := 0
			if _, err := fmt.Sscanf(seg, "%d", &idx); err != nil {
				return "", fmt.Errorf("path %q: %q is not an array index", path, seg)
			}
			if idx < 0 || idx >= len(node) {
				return "", fmt.Errorf("path %q: index %d out of range", path, idx)
			}
			cur = node[idx]
		default:
			return "", fmt.Errorf("path %q: cannot descend into %q", path, seg)
		}
	}
	s, ok := cur.(string)
	if !ok {
		return "", fmt.Errorf("path %q: value is not a string", path)
	}
	return s, nil
}

func snippet(b []byte) string {
	const max = 200
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
