package brain

import (
	"strings"
	"testing"
)

// The human-readable spec (docs/TVCP_AI_PROTOCOL.md) mirrors exactly what
// ReferenceCapabilities advertises. These tests bind that advertised surface to
// the conformance battery so the two can never drift apart: every advertised
// kind and game must be exercised by the battery, and the battery may not test a
// kind the reference brain does not advertise. If someone adds a game to the
// capabilities without a conformance case (or vice versa), CI fails here.

// batteryKindsAndGames derives the set of kinds and (move-)games the conformance
// battery exercises from its case names, which are "kind" or "move/<game>" or
// "image/<format>".
func batteryKindsAndGames() (kinds, games map[string]bool) {
	kinds, games = map[string]bool{}, map[string]bool{}
	for _, c := range ConformanceBattery() {
		seg := strings.SplitN(c.Name, "/", 2)
		kinds[seg[0]] = true
		if seg[0] == "move" && len(seg) == 2 {
			games[seg[1]] = true
		}
	}
	return
}

func TestSpecCoversAdvertisedKindsAndGames(t *testing.T) {
	caps := ReferenceCapabilities()
	kinds, games := batteryKindsAndGames()
	for _, k := range caps.Kinds {
		if !kinds[k] {
			t.Errorf("advertised kind %q is not exercised by the conformance battery", k)
		}
	}
	for _, g := range caps.Games {
		if !games[g] {
			t.Errorf("advertised game %q has no move/%s conformance case", g, g)
		}
	}
}

func TestBatteryHasNoUnadvertisedKinds(t *testing.T) {
	caps := ReferenceCapabilities()
	kinds, _ := batteryKindsAndGames()
	for k := range kinds {
		if !capHas(caps.Kinds, k) {
			t.Errorf("battery tests kind %q that ReferenceCapabilities does not advertise", k)
		}
	}
}

func TestProtocolVersionString(t *testing.T) {
	if Protocol != "tvcp-ai/1" {
		t.Fatalf("protocol constant drifted: %q", Protocol)
	}
	if ReferenceCapabilities().Protocol != Protocol {
		t.Fatal("ReferenceCapabilities must advertise the Protocol constant")
	}
}
