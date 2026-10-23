package brain

import (
	"encoding/json"
	"os"
	"testing"
)

// The published JSON Schemas must accept every request the reference brain
// advertises and the conformance battery sends; the request schema's game enum
// used to list only tictactoe, wordle and uno.
func TestAuditRequestSchemaCoversCapabilities(t *testing.T) {
	data, err := os.ReadFile("../../ai/schema/tvcp-ai.request.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	enum := func(field string) map[string]bool {
		m := map[string]bool{}
		for _, v := range doc.Properties[field].Enum {
			m[v] = true
		}
		return m
	}
	caps := ReferenceCapabilities()
	for field, vals := range map[string][]string{"kind": caps.Kinds, "game": caps.Games, "format": caps.Formats} {
		allowed := enum(field)
		for _, v := range vals {
			if !allowed[v] {
				t.Errorf("request schema rejects advertised %s %q", field, v)
			}
		}
	}
	kinds, games, formats := enum("kind"), enum("game"), enum("format")
	for _, c := range ConformanceBattery() {
		if !kinds[c.Req.Kind] {
			t.Errorf("%s: schema rejects kind %q", c.Name, c.Req.Kind)
		}
		if c.Req.Game != "" && !games[c.Req.Game] {
			t.Errorf("%s: schema rejects game %q", c.Name, c.Req.Game)
		}
		if c.Req.Format != "" && !formats[c.Req.Format] {
			t.Errorf("%s: schema rejects format %q", c.Name, c.Req.Format)
		}
	}
}

// The response schema documents every payload field a Response can carry, with
// a type matching what the reference brain emits for each battery case.
func TestAuditResponseSchemaCoversPayloads(t *testing.T) {
	data, err := os.ReadFile("../../ai/schema/tvcp-ai.response.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc struct {
		Properties map[string]struct {
			Type json.RawMessage `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	allows := func(field, jsonType string) bool {
		p, ok := doc.Properties[field]
		if !ok {
			return false
		}
		var one string
		if json.Unmarshal(p.Type, &one) == nil {
			return one == jsonType
		}
		var many []string
		_ = json.Unmarshal(p.Type, &many)
		for _, t := range many {
			if t == jsonType {
				return true
			}
		}
		return false
	}
	typeOf := func(raw json.RawMessage) string {
		var v any
		if json.Unmarshal(raw, &v) != nil {
			return "invalid"
		}
		switch v.(type) {
		case map[string]any:
			return "object"
		case []any:
			return "array"
		case nil:
			return "null"
		}
		return "scalar"
	}
	for _, c := range ConformanceBattery() {
		resp := Reference(c.Req)
		for field, raw := range map[string]json.RawMessage{
			"scene": resp.Scene, "sketch": resp.Sketch, "image": resp.Image, "tangram": resp.Tangram,
			"world": resp.World, "rpg": resp.Rpg, "ray": resp.Ray,
		} {
			if len(raw) == 0 {
				continue
			}
			if ty := typeOf(raw); !allows(field, ty) {
				t.Errorf("%s: response schema does not allow %s of type %s", c.Name, field, ty)
			}
		}
	}
}
