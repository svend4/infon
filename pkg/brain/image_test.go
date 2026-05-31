package brain

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
)

// The reference brain must be able to "paint" every pseudo-image format with no
// model at all — that is the whole point of pseudo-images.
func TestReferenceImageAllFormats(t *testing.T) {
	formats := []string{"", "grid", "pixels", "glyphs", "sigils", "vector", "sketch"}
	for _, f := range formats {
		resp := Reference(Request{
			Protocol: Protocol,
			Kind:     "image",
			Prompt:   "a calm harbor at dawn",
			Format:   f,
			Canvas:   &Canvas{Width: 60, Height: 26},
		})
		if resp.Error != "" {
			t.Fatalf("format %q: error %s", f, resp.Error)
		}
		if len(resp.Image) == 0 {
			t.Fatalf("format %q: empty image spec", f)
		}
		var spec pseudo.Spec
		if err := json.Unmarshal(resp.Image, &spec); err != nil {
			t.Fatalf("format %q: spec parse: %v", f, err)
		}
		frame, err := spec.Frame()
		if err != nil {
			t.Fatalf("format %q: render frame: %v", f, err)
		}
		if frame == nil || frame.Width == 0 || frame.Height == 0 {
			t.Fatalf("format %q: empty frame", f)
		}
		if _, err := spec.Image(120, 52); err != nil {
			t.Fatalf("format %q: render image: %v", f, err)
		}
	}
}
