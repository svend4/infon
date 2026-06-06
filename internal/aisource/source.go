package aisource

import (
	"os"
	"strconv"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/pseudo"
)

// BuildSource returns a device.Camera for a named synthesized source, so the
// SAME generative pipeline that powers `tvcp synth` / `tvcp ai` can be streamed
// over the network (send / call) and recorded — a model-painted picture flows to
// a peer exactly like webcam video.
//
// Recognized names:
//
//	plasma, ripple        procedural generators (Layer 1)
//	ai                    procedural plasma AICamera
//	neural                async neural source; backend chosen from the
//	                      environment (IMAGE_API_URL > BRAIN_URL > placeholder)
//
// Any other name returns (nil, false) so the caller can fall back to its own
// handling (e.g. the existing test-pattern camera).
func BuildSource(name string, width, height int, fps float64) (device.Camera, bool) {
	switch name {
	case "plasma", "ripple":
		gen, err := device.NewProceduralGenerator(name)
		if err != nil {
			return nil, false
		}
		return device.NewGenerativeSource(width, height, fps, gen), true
	case "ai":
		return NewAICamera(width, height, fps), true
	case "neural":
		gen := device.NewNeuralGenerator(NeuralBackendFromEnv(), "a calm landscape")
		return device.NewGenerativeSource(width, height, fps, gen), true
	default:
		return nil, false
	}
}

// NeuralBackendFromEnv picks a neural backend by a tiered policy (roadmap B4),
// best-available first:
//
//  1. IMAGE_API_URL  — a real raster text-to-image model (highest fidelity);
//  2. BRAIN_URL      — a tvcp-ai/1 sketch brain (local Ollama or cloud);
//  3. TVCP_LOCAL_BRAIN=1 — the offline, dependency-free procedural LocalBackend
//     (prompt-responsive imagery with no model/GPU/network);
//  4. otherwise nil  — NeuralGenerator shows its animated placeholder.
//
// The local tier keeps "visual chat" working on a Raspberry Pi or air-gapped
// host, honoring TVCP's offline-first identity.
func NeuralBackendFromEnv() device.NeuralBackend {
	backend := pickBackend()

	// Optional director+painter composition (roadmap B5): DIRECTOR_URL plans a
	// sketch with an LLM brain, then the chosen backend (if any) paints it; with
	// no painter it renders the plan procedurally. This wraps the base backend.
	if dir := os.Getenv("DIRECTOR_URL"); dir != "" {
		backend = NewDirectorPainter(dir, backend)
	}
	if backend == nil {
		return nil
	}
	// Optional restoration/super-resolution pass (roadmap C1): RESTORE_API_URL
	// post-processes each frame through a Real-ESRGAN-class model. No-op unset.
	backend = NewRestoreBackendFromEnv(backend)
	// Optional temporal-coherence wrapper (roadmap B1): TVCP_STREAM_COHERENCE in
	// (0,1] cross-fades frames for smooth, live evolution instead of hard cuts.
	if v := os.Getenv("TVCP_STREAM_COHERENCE"); v != "" {
		if c, err := strconv.ParseFloat(v, 64); err == nil && c > 0 {
			return NewStreamingBackend(backend, c)
		}
	}
	return backend
}

// pickBackend implements the base tier policy (highest fidelity first).
func pickBackend() device.NeuralBackend {
	if ib, ok := NewImageBackendFromEnv(); ok {
		return ib
	}
	if url := os.Getenv("BRAIN_URL"); url != "" {
		// TVCP_PSEUDO routes the brain through pkg/pseudo (kind:"image"): a plain
		// text model paints via a coarse color grid / glyphs / sigils / vectors —
		// no diffusion model. The value names the format ("1"/"true" = grid).
		if pf, ok := os.LookupEnv("TVCP_PSEUDO"); ok {
			switch pf {
			case "1", "true", "yes", "on":
				pf = ""
			}
			return NewPseudoBackend(url, pseudo.Format(pf))
		}
		return NewBrainBackend(url)
	}
	if os.Getenv("TVCP_LOCAL_BRAIN") == "1" {
		return NewLocalBackend()
	}
	return nil
}

// IsGenerativeName reports whether name selects a synthesized source (as opposed
// to a built-in test pattern like "bounce" or "gradient").
func IsGenerativeName(name string) bool {
	switch name {
	case "plasma", "ripple", "ai", "neural":
		return true
	default:
		return false
	}
}
