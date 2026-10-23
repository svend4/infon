package brain

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/svend4/infon/pkg/scene"
)

// promptText returns the prompt caption (the text op at 2,1) of a draw scene.
func promptText(t *testing.T, data json.RawMessage) string {
	t.Helper()
	var sc scene.Scene
	if err := json.Unmarshal(data, &sc); err != nil {
		t.Fatalf("scene: %v", err)
	}
	for _, op := range sc.Ops {
		if op.Op == "text" && op.X == 2 && op.Y == 1 {
			return op.S
		}
	}
	t.Fatal("no prompt text op in scene")
	return ""
}

// refDraw truncated the prompt to w-4 bytes, so a canvas 1..3 cells wide sliced
// with a negative bound and panicked (Reference is served over HTTP).
func TestAuditRefDrawNarrowCanvas(t *testing.T) {
	for w := 1; w <= 6; w++ {
		for _, prompt := range []string{"", "a calm harbor at dawn"} {
			for _, req := range []Request{
				{Kind: "draw", Prompt: prompt, Canvas: &Canvas{Width: w, Height: 10}},
				{Kind: "image", Format: "vector", Prompt: prompt, Canvas: &Canvas{Width: w, Height: 10}},
			} {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("kind %s width %d prompt %q: panic: %v", req.Kind, w, prompt, r)
						}
					}()
					resp := Reference(req)
					if resp.Error != "" {
						t.Fatalf("kind %s width %d: error %q", req.Kind, w, resp.Error)
					}
					if req.Kind == "draw" {
						if s := promptText(t, resp.Scene); len(s) > max(w-4, 0) {
							t.Fatalf("width %d: caption %q longer than %d bytes", w, s, max(w-4, 0))
						}
					}
				}()
			}
		}
	}
}

// The caption is cut on a rune boundary, so a non-ASCII prompt never turns into
// a broken glyph at the cut.
func TestAuditRefDrawCutsOnRuneBoundary(t *testing.T) {
	prompt := "привет мир"
	for w := 5; w < 4+len(prompt); w++ {
		resp := Reference(Request{Kind: "draw", Prompt: prompt, Canvas: &Canvas{Width: w, Height: 10}})
		s := promptText(t, resp.Scene)
		if !utf8.ValidString(s) || strings.ContainsRune(s, utf8.RuneError) {
			t.Fatalf("width %d: caption %q cut inside a rune", w, s)
		}
		if !strings.HasPrefix(prompt, s) {
			t.Fatalf("width %d: caption %q is not a prefix of the prompt", w, s)
		}
	}
}
