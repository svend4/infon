package brain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Handler decoded the whole POST body with no cap; a request is a few hundred
// bytes, so a multi-megabyte body is refused instead of being held in memory.
func TestAuditHandlerRejectsOversizeBody(t *testing.T) {
	body := `{"protocol":"tvcp-ai/1","kind":"react","prompt":"` + strings.Repeat("a", 4<<20) + `"}`
	rec := httptest.NewRecorder()
	Handler()(rec, httptest.NewRequest(http.MethodPost, "/v1/decide", strings.NewReader(body)))
	if rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("4 MiB request answered with status %d, want a 4xx", rec.Code)
	}

	// An ordinary request still works.
	rec = httptest.NewRecorder()
	Handler()(rec, httptest.NewRequest(http.MethodPost, "/v1/decide", strings.NewReader(`{"kind":"react","prompt":"hi"}`)))
	var resp Response
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &resp) != nil || len(resp.Cards) == 0 {
		t.Fatalf("small request: status %d body %q", rec.Code, rec.Body.String())
	}
}

// refRpg pairs every unit with every enemy. A state of tens of thousands of
// "{}" entries fits in a small body but costs ~10^9 steps, so the reference
// commander bounds how many units and enemies it considers (1024 each).
func TestAuditRefRpgBoundsWork(t *testing.T) {
	const n = 30000
	entries := strings.TrimSuffix(strings.Repeat("{},", n), ",")
	state := json.RawMessage(`{"units":[` + entries + `],"enemies":[` + entries + `]}`)
	resp := Reference(Request{Kind: "move", Game: "rpg", State: state})
	var moves []struct{ ID, DX, DY int }
	if err := json.Unmarshal(resp.Rpg, &moves); err != nil {
		t.Fatalf("rpg reply: %v (%q)", err, resp.Rpg)
	}
	if len(moves) > 1024 {
		t.Fatalf("answered %d units, want at most 1024", len(moves))
	}

	// A state with no units is answered with an empty list, not null.
	resp = Reference(Request{Kind: "move", Game: "rpg", State: json.RawMessage(`{"units":[],"enemies":[]}`)})
	if string(resp.Rpg) != "[]" {
		t.Fatalf("rpg reply for no units = %s, want []", resp.Rpg)
	}
}
