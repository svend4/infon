package raydir

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

// loadSchemaEnums parses the rayscene JSON schema and returns the kind and anim
// enums declared in it.
func loadSchemaEnums(t *testing.T) (kinds, anims map[string]bool) {
	t.Helper()
	data, err := os.ReadFile("../../ai/schema/rayscene.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc struct {
		Definitions struct {
			Obj struct {
				Properties struct {
					Kind struct {
						Enum []string `json:"enum"`
					} `json:"kind"`
					Anim struct {
						Enum []string `json:"enum"`
					} `json:"anim"`
				} `json:"properties"`
			} `json:"obj"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	kinds, anims = map[string]bool{}, map[string]bool{}
	for _, k := range doc.Definitions.Obj.Properties.Kind.Enum {
		kinds[k] = true
	}
	for _, a := range doc.Definitions.Obj.Properties.Anim.Enum {
		anims[a] = true
	}
	return kinds, anims
}

// The schema's enums must stay in lockstep with what the renderer accepts, so the
// published contract can never drift from the code.
func TestSchemaMatchesRenderer(t *testing.T) {
	kinds, anims := loadSchemaEnums(t)
	if len(kinds) != len(knownKinds) {
		t.Errorf("schema kind enum (%d) and knownKinds (%d) differ in size", len(kinds), len(knownKinds))
	}
	for k := range knownKinds {
		if !kinds[k] {
			t.Errorf("renderer kind %q missing from the schema enum", k)
		}
	}
	for k := range kinds {
		if !knownKinds[k] {
			t.Errorf("schema kind %q is not accepted by the renderer", k)
		}
	}
	// anim enum = animKinds plus the empty (no-motion) string
	for a := range animKinds {
		if !anims[a] {
			t.Errorf("renderer motion %q missing from the schema enum", a)
		}
	}
	if !anims[""] {
		t.Error("schema anim enum should include the empty (no-motion) value")
	}
	for a := range anims {
		if a != "" && !animKinds[a] {
			t.Errorf("schema anim %q is not a renderer motion", a)
		}
	}
}

// The reference director's output conforms to the schema's enums and colour range.
func TestReferenceAuthorConformsToSchema(t *testing.T) {
	kinds, anims := loadSchemaEnums(t)
	b := brain.Local{}
	for _, p := range []string{"a forest with fireflies", "a crystal cave", "a surreal dreamscape with water", "a vast golden hall, grand and open"} {
		_, spec, _ := AuthorScene(b, p)
		for _, o := range spec.Objects {
			if !kinds[o.Kind] {
				t.Errorf("authored kind %q (prompt %q) not in the schema", o.Kind, p)
			}
			if !anims[o.Anim] {
				t.Errorf("authored anim %q (prompt %q) not in the schema", o.Anim, p)
			}
			for _, c := range o.Color {
				if c < 0 || c > 1 {
					t.Errorf("authored colour %.2f out of the schema's [0,1] range (prompt %q)", c, p)
				}
			}
		}
	}
}
