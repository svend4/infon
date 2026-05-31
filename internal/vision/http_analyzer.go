package vision

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
	"time"
)

// HTTPAnalyzer is a FrameAnalyzer backed by an external vision model over HTTP
// (local sidecar #1 or cloud #2). It speaks a tiny contract:
//
//	POST {url}  JSON {"image": "<base64 png>"}
//	-> JSON {"detections": [{"label","confidence","x","y","w","h","points"}]}
//
// coords are normalized 0..1 so the response is resolution-independent.
type HTTPAnalyzer struct {
	url    string
	apiKey string
	http   *http.Client
}

var _ FrameAnalyzer = (*HTTPAnalyzer)(nil)

// NewHTTPAnalyzer creates an analyzer pointed at url (optional bearer apiKey).
func NewHTTPAnalyzer(url, apiKey string) *HTTPAnalyzer {
	return &HTTPAnalyzer{url: url, apiKey: apiKey, http: &http.Client{Timeout: 30 * time.Second}}
}

// NewHTTPAnalyzerFromEnv builds an analyzer from VISION_API_URL (+ optional
// VISION_API_KEY), or returns (nil,false) when unset.
func NewHTTPAnalyzerFromEnv() (*HTTPAnalyzer, bool) {
	url := os.Getenv("VISION_API_URL")
	if url == "" {
		return nil, false
	}
	return NewHTTPAnalyzer(url, os.Getenv("VISION_API_KEY")), true
}

// Name implements FrameAnalyzer.
func (a *HTTPAnalyzer) Name() string { return "http-vision" }

// Analyze implements FrameAnalyzer: PNG-encode, POST, decode detections.
func (a *HTTPAnalyzer) Analyze(ctx context.Context, img image.Image) ([]Detection, error) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, err
	}
	reqBody, err := json.Marshal(map[string]any{
		"image": base64.StdEncoding.EncodeToString(pngBuf.Bytes()),
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vision api: status %d", resp.StatusCode)
	}

	var out struct {
		Detections []Detection `json:"detections"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Detections, nil
}
