// Restoration backend (roadmap C1).
//
// A NeuralBackend decorator that sends each generated frame to an image
// RESTORATION/super-resolution model (Real-ESRGAN / SwinIR class) and uses the
// improved result. The model can be a local HTTP sidecar (#1) or a cloud API
// (#2); both speak the same tiny contract as ImageBackend's raster path:
//
//	POST {url}  JSON {"image":"<base64 png>", "scale":N}
//	-> raw image bytes, OR JSON with a base64 field at RESTORE_B64_PATH.
//
// Because the terminal's output canvas is fixed and tiny, restoration only needs
// to clean up a small image (denoise/sharpen) before block encoding — the exact
// niche C1 describes. It wraps any inner backend (or runs on a passthrough), so
// it composes with the offline brain, the raster API, and StreamingBackend.
package aisource

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
	"os"
	"strconv"
	"time"

	"github.com/svend4/infon/internal/device"
)

// RestoreConfig configures the restoration HTTP call.
type RestoreConfig struct {
	URL     string // POST endpoint of the restoration model
	APIKey  string // optional bearer token (cloud)
	Scale   int    // upscale factor hint (1 = restore only)
	B64Path string // dot-path to base64 in a JSON response; "" = raw bytes
	Timeout time.Duration
}

// RestoreBackend wraps an inner NeuralBackend and post-processes its frames
// through a restoration model.
type RestoreBackend struct {
	inner device.NeuralBackend
	cfg   RestoreConfig
	http  *http.Client
}

var _ device.NeuralBackend = (*RestoreBackend)(nil)

// NewRestoreBackend wraps inner with a restoration pass described by cfg.
func NewRestoreBackend(inner device.NeuralBackend, cfg RestoreConfig) *RestoreBackend {
	if cfg.Scale <= 0 {
		cfg.Scale = 1
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &RestoreBackend{inner: inner, cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// NewRestoreBackendFromEnv wraps inner when RESTORE_API_URL is set, else returns
// inner unchanged so callers can opt in transparently:
//
//	RESTORE_API_URL   POST endpoint (required to enable)
//	RESTORE_API_KEY   optional bearer token
//	RESTORE_SCALE     optional integer upscale factor (default 1)
//	RESTORE_B64_PATH  optional base64 dot-path (e.g. data.0.b64_json)
func NewRestoreBackendFromEnv(inner device.NeuralBackend) device.NeuralBackend {
	url := os.Getenv("RESTORE_API_URL")
	if url == "" || inner == nil {
		return inner
	}
	scale, _ := strconv.Atoi(os.Getenv("RESTORE_SCALE"))
	return NewRestoreBackend(inner, RestoreConfig{
		URL:     url,
		APIKey:  os.Getenv("RESTORE_API_KEY"),
		Scale:   scale,
		B64Path: os.Getenv("RESTORE_B64_PATH"),
	})
}

// Name implements device.NeuralBackend.
func (b *RestoreBackend) Name() string {
	if b.inner == nil {
		return "restore"
	}
	return "restore/" + b.inner.Name()
}

// Generate produces the inner frame, then restores it. On any restoration error
// it falls back to the unrestored frame, so the stream never stops.
func (b *RestoreBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	base, err := b.inner.Generate(ctx, prompt, width, height)
	if err != nil {
		return nil, err
	}
	restored, rerr := b.restore(ctx, base)
	if rerr != nil {
		return base, nil // graceful fallback
	}
	return restored, nil
}

func (b *RestoreBackend) restore(ctx context.Context, img image.Image) (image.Image, error) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, err
	}
	reqBody, err := json.Marshal(map[string]any{
		"image": base64.StdEncoding.EncodeToString(pngBuf.Bytes()),
		"scale": b.cfg.Scale,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.cfg.APIKey)
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("restore api: status %d", resp.StatusCode)
	}

	raw := body
	if b.cfg.B64Path != "" {
		var doc any
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, err
		}
		s, err := lookupString(doc, b.cfg.B64Path)
		if err != nil {
			return nil, err
		}
		if raw, err = base64.StdEncoding.DecodeString(s); err != nil {
			return nil, err
		}
	}
	out, _, err := image.Decode(bytes.NewReader(raw))
	return out, err
}
