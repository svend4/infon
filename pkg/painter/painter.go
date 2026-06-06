// Package painter is a tiny, on-device "painter" — the no-network counterpart to
// a diffusion model. Where the reference brain paints from a prompt HASH, this
// one READS the prompt: a small embedded lexicon maps words (night, sunset, sea,
// forest, mountain, snow...) to a sky/ground palette and features, and it emits a
// coarse pseudo-image grid that the deterministic pseudo-diffusion renderer turns
// into a painterly picture. It implements brain.Brain, so painter.Brain{} is a
// drop-in tvcp-ai/1 backend that needs no model and no network.
package painter

import (
	"encoding/json"
	"hash/fnv"
	"math/rand"
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/pseudo"
)

type look struct {
	skyTop, skyHorizon, ground string
	water, stars, mountains    bool
	orb, orbColor              string // orb: "sun" | "moon" | ""
}

func contains(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// analyze reads a prompt into a concrete palette + features (the "model").
func analyze(prompt string) look {
	s := strings.ToLower(prompt)
	L := look{skyTop: "skyblue", skyHorizon: "blue", ground: "darkgreen", orb: "sun", orbColor: "gold"}
	switch {
	case contains(s, "night", "midnight", "starry", "starr"):
		L.skyTop, L.skyHorizon, L.stars, L.orb, L.orbColor = "navy", "navy", true, "moon", "white"
	case contains(s, "sunset", "dusk", "evening"):
		L.skyTop, L.skyHorizon, L.orbColor = "dusk", "orange", "gold"
	case contains(s, "dawn", "sunrise", "morning"):
		L.skyTop, L.skyHorizon, L.orbColor = "purple", "amber", "gold"
	case contains(s, "storm", "rain", "cloud", "overcast", "fog"):
		L.skyTop, L.skyHorizon, L.orb = "slate", "gray", ""
	}
	switch {
	case contains(s, "sea", "ocean", "harbor", "harbour", "water", "lake", "river", "bay", "coast"):
		L.ground, L.water = "teal", true
	case contains(s, "forest", "wood", "jungle", "tree"):
		L.ground = "darkgreen"
	case contains(s, "mountain", "hill", "peak", "alp", "ridge"):
		L.ground, L.mountains = "slate", true
	case contains(s, "desert", "sand", "dune"):
		L.ground = "sand"
	case contains(s, "snow", "ice", "winter", "arctic"):
		L.ground = "white"
		if L.skyTop == "skyblue" {
			L.skyTop, L.skyHorizon = "white", "skyblue"
		}
	case contains(s, "city", "town", "street", "urban"):
		L.ground = "gray"
	case contains(s, "meadow", "field", "grass", "plain", "valley"):
		L.ground = "green"
	}
	return L
}

func seedOf(prompt string) int64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(prompt))
	return int64(h.Sum32())
}

const seedW, seedH = 16, 10

// Paint turns a prompt into a pseudo-image grid spec (format "grid"), which the
// pseudo-diffusion renderer expands into a painterly image. Deterministic.
func Paint(prompt string, cols, rows int) pseudo.Spec {
	if cols <= 0 {
		cols = 64
	}
	if rows <= 0 {
		rows = 28
	}
	L := analyze(prompt)
	rng := rand.New(rand.NewSource(seedOf(prompt)))
	horizon := seedH * 6 / 10

	g := make([][]string, seedH)
	for y := 0; y < seedH; y++ {
		g[y] = make([]string, seedW)
		for x := 0; x < seedW; x++ {
			switch {
			case y < horizon-1:
				g[y][x] = L.skyTop
			case y < horizon:
				g[y][x] = L.skyHorizon
			case y == horizon:
				g[y][x] = L.skyHorizon // soft horizon
			default:
				g[y][x] = L.ground
				if L.water && (x+y)%3 == 0 {
					g[y][x] = "skyblue"
				}
			}
		}
	}

	// the orb (sun/moon): a 2x2 glow high in the sky, side chosen by the prompt.
	if L.orb != "" {
		ox := 2 + rng.Intn(seedW-5)
		oy := 1 + rng.Intn(2)
		for dy := 0; dy < 2; dy++ {
			for dx := 0; dx < 2; dx++ {
				if oy+dy < horizon-1 && ox+dx < seedW {
					g[oy+dy][ox+dx] = L.orbColor
				}
			}
		}
	}
	// stars: scattered white specks in a night sky.
	if L.stars {
		for i := 0; i < 8; i++ {
			sx, sy := rng.Intn(seedW), rng.Intn(horizon-1)
			g[sy][sx] = "white"
		}
	}
	// mountains: a few darker peaks sitting on the horizon.
	if L.mountains {
		for i := 0; i < 4; i++ {
			mx := rng.Intn(seedW)
			if horizon-1 >= 0 {
				g[horizon-1][mx] = "slate"
			}
		}
	}

	return pseudo.Spec{
		Format: pseudo.FormatGrid,
		Cols:   cols,
		Rows:   rows,
		Title:  prompt,
		Grid:   &pseudo.Grid{Rows: g},
	}
}

// Brain is the on-device painter as a tvcp-ai/1 backend. It paints kind:"image"
// requests locally and delegates every other kind to the reference policy, so it
// is a complete, network-free drop-in brain.
type Brain struct {
	Cols, Rows int
}

// Decide implements brain.Brain.
func (b Brain) Decide(req brain.Request) (brain.Response, error) {
	if req.Kind != "image" {
		return brain.Reference(req), nil
	}
	cols, rows := b.Cols, b.Rows
	if req.Canvas != nil {
		if req.Canvas.Width > 0 {
			cols = req.Canvas.Width
		}
		if req.Canvas.Height > 0 {
			rows = req.Canvas.Height
		}
	}
	spec := Paint(req.Prompt, cols, rows)
	if req.Palette != "" {
		spec.Palette = req.Palette
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return brain.Response{Protocol: brain.Protocol, Kind: "image", Error: err.Error()}, nil
	}
	return brain.Response{Protocol: brain.Protocol, Kind: "image", Image: data, Reasoning: "on-device painter"}, nil
}
