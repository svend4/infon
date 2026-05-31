// Package visualchat defines the runtime control surface for "communicating in
// pictures" (roadmap B2): a small, transport-agnostic Steer message that changes
// what a neural image source generates mid-stream, and a Controller that applies
// steers to the async neural generator.
//
// The idea: instead of (or alongside) text chat, a participant or an agent sends
// Steer messages — "a calm bay", "now a storm", warmer, different seed — and the
// shared generative video reacts live. It rides the existing tvcp-ai/1 JSON
// contract, so any transport (HTTP, the call's data channel, a local pipe) can
// carry it.
package visualchat

import (
	"encoding/json"
	"errors"
	"strings"
)

// Protocol is the steer message protocol tag (extends tvcp-ai/1).
const Protocol = "tvcp-ai/1"

// Steer is a single visual-chat control message. All fields are optional; only
// the non-zero ones are applied, so a steer can nudge just the prompt, just the
// style, etc.
type Steer struct {
	Protocol string `json:"protocol,omitempty"`
	Kind     string `json:"kind,omitempty"` // always "steer" on the wire

	// Prompt replaces the generation prompt outright.
	Prompt string `json:"prompt,omitempty"`
	// Append, if set, is appended to the current prompt instead of replacing it
	// (e.g. ", at sunset"). Ignored when Prompt is set.
	Append string `json:"append,omitempty"`
	// Style is a named modifier blended into the prompt ("watercolor", "noir").
	Style string `json:"style,omitempty"`
	// Strength hints how strongly to change from the previous frame, 0..1
	// (img2img denoising strength); <=0 means "unset / keep".
	Strength float64 `json:"strength,omitempty"`
	// Seed pins the random seed; 0 means "unset / random".
	Seed int64 `json:"seed,omitempty"`
}

// ErrNotSteer is returned when a decoded message is not a steer.
var ErrNotSteer = errors.New("visualchat: not a steer message")

// Encode serializes a steer to JSON, filling in protocol/kind.
func (s Steer) Encode() ([]byte, error) {
	s.Protocol = Protocol
	s.Kind = "steer"
	return json.Marshal(s)
}

// DecodeSteer parses a steer message, validating its kind.
func DecodeSteer(data []byte) (Steer, error) {
	var s Steer
	if err := json.Unmarshal(data, &s); err != nil {
		return Steer{}, err
	}
	if s.Kind != "" && s.Kind != "steer" {
		return Steer{}, ErrNotSteer
	}
	return s, nil
}

// IsEmpty reports whether the steer would change nothing.
func (s Steer) IsEmpty() bool {
	return s.Prompt == "" && s.Append == "" && s.Style == "" && s.Strength <= 0 && s.Seed == 0
}

// Apply returns the new prompt produced by applying this steer to current. The
// prompt rules: Prompt replaces; otherwise Append extends; Style is added as a
// trailing ", <style>" qualifier (once). Strength/Seed are surfaced separately
// via the State the Controller tracks.
func (s Steer) Apply(current string) string {
	out := current
	switch {
	case s.Prompt != "":
		out = s.Prompt
	case s.Append != "":
		if out == "" {
			out = s.Append
		} else {
			out = out + ", " + strings.TrimSpace(s.Append)
		}
	}
	if s.Style != "" {
		qualifier := ", " + strings.TrimSpace(s.Style)
		if !strings.Contains(out, qualifier) {
			out += qualifier
		}
	}
	return out
}
