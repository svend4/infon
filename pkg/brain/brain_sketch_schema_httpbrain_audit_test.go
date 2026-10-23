package brain

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A non-2xx reply is a failure even when its body happens to parse as a
// Response: a 500 page must not be played as a move.
func TestAuditHTTPBrainRejectsErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"protocol":"tvcp-ai/1","kind":"move","move":{"row":1,"col":1}}`)
	}))
	defer srv.Close()
	resp, err := HTTPBrain{URL: srv.URL, HTTP: srv.Client()}.Decide(Request{Kind: "move", Game: "tictactoe"})
	if err == nil {
		t.Fatalf("status 500 accepted as a valid answer: %+v", resp)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q does not name the status", err)
	}

	// An error message in a non-2xx body is surfaced in the error.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"protocol":"tvcp-ai/1","error":"upstream down"}`)
	}))
	defer bad.Close()
	if _, err := (HTTPBrain{URL: bad.URL, HTTP: bad.Client()}).Decide(Request{Kind: "react"}); err == nil || !strings.Contains(err.Error(), "upstream down") {
		t.Errorf("502 with an error body: err = %v, want it to carry the message", err)
	}
}

// A brain that streams an endless JSON string must not be read into memory in
// full: Decide gives up past a fixed cap instead of accepting the reply.
func TestAuditHTTPBrainBoundsResponseBody(t *testing.T) {
	const total = 64 << 20
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"protocol":"tvcp-ai/1","kind":"react","cards":["x"],"reasoning":"`); err != nil {
			return
		}
		chunk := strings.Repeat("a", 64<<10)
		for n := 0; n < total; n += len(chunk) {
			if _, err := io.WriteString(w, chunk); err != nil {
				return // the client hung up, as it should
			}
		}
		_, _ = io.WriteString(w, `"}`)
	}))
	defer srv.Close()
	resp, err := HTTPBrain{URL: srv.URL, HTTP: srv.Client()}.Decide(Request{Kind: "react"})
	if err == nil {
		t.Fatalf("accepted a %d MiB response (reasoning %d bytes)", total>>20, len(resp.Reasoning))
	}
}
