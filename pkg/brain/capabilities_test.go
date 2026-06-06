package brain

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestReferenceCapabilitiesSupports(t *testing.T) {
	c := ReferenceCapabilities()
	if !c.Supports("move", "rpg") {
		t.Fatal("should support move/rpg")
	}
	if !c.Supports("image", "") {
		t.Fatal("should support image")
	}
	if c.Supports("move", "chess") {
		t.Fatal("should not claim chess")
	}
	if c.Supports("dream", "") {
		t.Fatal("should not claim an unknown kind")
	}
}

func TestCapabilitiesAndDecideOverMux(t *testing.T) {
	srv := httptest.NewServer(Mux(ReferenceCapabilities()))
	defer srv.Close()

	// /v1/capabilities advertises the reference set
	got, err := FetchCapabilities(srv.URL + "/v1/capabilities")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.Protocol != Protocol || !got.Supports("move", "tangram") || !got.Supports("react", "") {
		t.Fatalf("advertised caps wrong: %+v", got)
	}

	// /v1/decide on the same endpoint still answers a move
	h := HTTPBrain{URL: srv.URL + "/v1/decide"}
	st, _ := json.Marshal(TicTacToeState{Turn: "X", You: "X", Legal: [][2]int{{1, 1}}})
	resp, err := h.Decide(Request{Kind: "move", Game: "tictactoe", State: st})
	if err != nil || resp.Move == nil {
		t.Fatalf("decide over mux failed: %+v err=%v", resp, err)
	}
}
