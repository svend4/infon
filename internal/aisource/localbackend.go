// LocalBackend is a fully offline, dependency-free device.NeuralBackend
// (roadmap B4). It needs no model, no GPU, and no network: it deterministically
// turns a prompt into a simple landscape sketch (sky/ground/sun/stars chosen by
// hashing the prompt and scanning for a few keywords), then rasterizes it
// through the repo's own scene renderer.
//
// It is the bottom tier of the backend policy, so "visual chat" still produces
// prompt-responsive imagery on a Raspberry Pi or an air-gapped box where no real
// model can run. It is intentionally crude — a placeholder a real model
// improves on, not a competitor to one.
package aisource

import (
	"context"
	"image"
	"strings"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
)

// LocalBackend renders prompts offline via the procedural sketch generator.
type LocalBackend struct{}

var _ device.NeuralBackend = (*LocalBackend)(nil)

// NewLocalBackend creates the offline procedural backend.
func NewLocalBackend() *LocalBackend { return &LocalBackend{} }

// Name implements device.NeuralBackend.
func (b *LocalBackend) Name() string { return "local-procedural" }

// Generate implements device.NeuralBackend: build a sketch from the prompt and
// rasterize it to width×height. ctx is honored implicitly (the work is trivial).
func (b *LocalBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	cols, rows := width/8, height/8
	if cols < 16 {
		cols = 16
	}
	if rows < 8 {
		rows = 8
	}
	sk := sketchFromPrompt(prompt, cols, rows)
	sc := sketch.ToScene(sk, cols, rows)
	return frameToImage(scene.RenderScene(&sc), width, height), nil
}

// sketchFromPrompt deterministically maps a prompt to a landscape sketch:
// keywords pick palette/elements, and a hash adds stable variety so different
// prompts look different but the same prompt is reproducible.
func sketchFromPrompt(prompt string, cols, rows int) sketch.Sketch {
	p := strings.ToLower(prompt)
	h := hashString(p)

	// Sky by time-of-day / weather keywords, else hash-chosen.
	skies := []string{"navy", "skyblue", "orange", "purple", "teal", "dusk"}
	sky := skies[h%uint32(len(skies))]
	switch {
	case strings.Contains(p, "night"), strings.Contains(p, "dark"):
		sky = "navy"
	case strings.Contains(p, "sunset"), strings.Contains(p, "dusk"), strings.Contains(p, "evening"):
		sky = "orange"
	case strings.Contains(p, "dawn"), strings.Contains(p, "morning"):
		sky = "skyblue"
	case strings.Contains(p, "storm"), strings.Contains(p, "rain"):
		sky = "gray"
	}

	grounds := []string{"darkgreen", "navy", "brown", "teal"}
	ground := grounds[(h/7)%uint32(len(grounds))]
	if strings.Contains(p, "sea") || strings.Contains(p, "ocean") || strings.Contains(p, "bay") || strings.Contains(p, "harbor") {
		ground = "navy"
	}

	shapes := []sketch.Shape{}

	// A sun or moon depending on the sky.
	if sky == "navy" {
		shapes = append(shapes, sketch.Shape{Kind: "moon", X: cols * 3 / 4, Y: rows / 4, W: 2, Color: "white"})
		// Stars at night.
		for i := 0; i < 5; i++ {
			sx := int((h>>uint(i))%uint32(cols-2)) + 1
			shapes = append(shapes, sketch.Shape{Kind: "star", X: sx, Y: 1 + i%3, Color: "white"})
		}
	} else {
		shapes = append(shapes, sketch.Shape{Kind: "sun", X: cols * 3 / 4, Y: rows / 4, W: 3, Color: "gold"})
	}

	// Mountains unless it's clearly a seascape.
	if !strings.Contains(p, "sea") && !strings.Contains(p, "ocean") {
		shapes = append(shapes,
			sketch.Shape{Kind: "mountain", X: cols / 4, H: rows / 3, Color: "darkgreen"},
			sketch.Shape{Kind: "mountain", X: cols / 2, H: rows / 4, Color: "darkgreen"},
		)
	}

	// A caption with the (trimmed) prompt.
	caption := prompt
	if len(caption) > cols-2 {
		caption = caption[:cols-2]
	}
	shapes = append(shapes, sketch.Shape{Kind: "text", X: 1, Y: 0, S: caption, Color: "white"})

	return sketch.Sketch{Sky: sky, Ground: ground, Shapes: shapes}
}

// hashString is a small stable FNV-1a hash (kept local to avoid a dependency).
func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	if h == 0 {
		h = 1
	}
	return h
}
