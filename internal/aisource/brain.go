package aisource

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
	"github.com/svend4/infon/pkg/terminal"
)

// BrainFrame asks a tvcp-ai/1 brain for a sketch and renders it to a Frame, so a
// model can act as a (slow, painterly) video source. Falls back to a notice.
func BrainFrame(url, prompt string, w, h int) *terminal.Frame {
	hb := brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 120 * time.Second}}
	resp, err := hb.Decide(brain.Request{Kind: "sketch", Prompt: prompt, Canvas: &brain.Canvas{Width: w, Height: h}})
	var sk sketch.Sketch
	if err == nil && resp.Error == "" && json.Unmarshal(resp.Sketch, &sk) == nil && len(sk.Shapes) > 0 {
		sc := sketch.ToScene(sk, w, h)
		return scene.RenderScene(&sc)
	}
	f := terminal.NewFrame(w, h)
	f.Fill(' ', color.NewRGB(200, 200, 210), color.NewRGB(10, 10, 16))
	f.DrawText(1, 1, "(brain frame unavailable)", color.NewRGB(235, 120, 120), color.NewRGB(10, 10, 16))
	return f
}
