package brain

import (
	"encoding/json"
	"net/http"
	"time"
)

// capabilities.go adds capability NEGOTIATION to tvcp-ai/1: a brain advertises
// which kinds, games and image formats it supports at GET /v1/capabilities, so a
// client can check support before sending a request (and a registry/discovery
// layer can route by capability). It complements POST /v1/decide.

// Capabilities is what a brain advertises it can do.
type Capabilities struct {
	Protocol string   `json:"protocol"`
	Kinds    []string `json:"kinds"`
	Games    []string `json:"games,omitempty"`
	Formats  []string `json:"formats,omitempty"`
}

func capHas(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ReferenceCapabilities is what the built-in Reference brain supports.
func ReferenceCapabilities() Capabilities {
	return Capabilities{
		Protocol: Protocol,
		Kinds:    []string{"move", "draw", "sketch", "image", "react"},
		Games:    []string{"tictactoe", "wordle", "uno", "tangram", "world", "rpg", "ray", "rayscene"},
		Formats:  []string{"grid", "pixels", "glyphs", "sigils", "vector", "sketch", "mixed", "marks"},
	}
}

// Supports reports whether the advertised capabilities cover a kind (and game, if
// given). An empty game means "kind only".
func (c Capabilities) Supports(kind, game string) bool {
	if !capHas(c.Kinds, kind) {
		return false
	}
	if game != "" && !capHas(c.Games, game) {
		return false
	}
	return true
}

// CapabilitiesHandler serves the given capabilities as JSON (GET /v1/capabilities).
func CapabilitiesHandler(c Capabilities) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.Protocol == "" {
			c.Protocol = Protocol
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(c)
	}
}

// Mux returns an http.Handler serving the reference brain at /v1/decide and the
// given capabilities at /v1/capabilities — a complete tvcp-ai/1 endpoint.
func Mux(c Capabilities) http.Handler {
	m := http.NewServeMux()
	m.Handle("/v1/decide", Handler())
	m.Handle("/v1/capabilities", CapabilitiesHandler(c))
	return m
}

// FetchCapabilities GETs and decodes a brain's advertised capabilities.
func FetchCapabilities(url string) (Capabilities, error) {
	cl := &http.Client{Timeout: 10 * time.Second}
	resp, err := cl.Get(url)
	if err != nil {
		return Capabilities{}, err
	}
	defer resp.Body.Close()
	var c Capabilities
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return Capabilities{}, err
	}
	return c, nil
}
