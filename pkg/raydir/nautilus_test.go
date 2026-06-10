package raydir

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// nautilus_test.go keeps the federation manifest honest: it must be valid JSON,
// carry the core fields, and every file it points at (schema, adapters, interop
// doc) must actually exist — so the manifest can never drift from the repo. Go runs
// the test with its working dir at this package, so the repo root is ../..
func TestNautilusManifestIsHonest(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "nautilus.json"))
	if err != nil {
		t.Fatalf("read nautilus.json: %v", err)
	}
	var m struct {
		Format    string   `json:"format"`
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Protocols []string `json:"protocols"`
		Schemas   []string `json:"schemas"`
		Adapters  []struct {
			Path string `json:"path"`
		} `json:"adapters"`
		Links []struct {
			ID string `json:"id"`
		} `json:"links"`
		InteropDoc string `json:"interop_doc"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("nautilus.json is not valid JSON: %v", err)
	}
	if m.Format == "" || m.ID == "" || m.Name == "" {
		t.Errorf("manifest is missing core fields: %+v", m)
	}
	if len(m.Protocols) == 0 || len(m.Links) == 0 {
		t.Errorf("manifest should advertise protocols and federation links")
	}
	refs := append([]string{m.InteropDoc}, m.Schemas...)
	for _, a := range m.Adapters {
		refs = append(refs, a.Path)
	}
	for _, r := range refs {
		if r == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, r)); err != nil {
			t.Errorf("manifest points at a missing file: %s", r)
		}
	}
}
