// Command planbrain serves the arena lookahead Planner as a tvcp-ai/1 endpoint,
// so any client can point -brain at it by URL. For game:rpg it plans the next
// moves from a rich RpgState; other kinds fall back to the reference policy.
//
//	go run ./cmd/planbrain -addr 127.0.0.1:8099
//	# then: go run ./cmd/arenaserver ... -brain http://127.0.0.1:8099/v1/decide
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/planbrain"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8099", "listen address")
	flag.Parse()

	http.HandleFunc("/v1/decide", func(w http.ResponseWriter, r *http.Request) {
		var req brain.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(brain.Response{Protocol: brain.Protocol, Error: "bad request: " + err.Error()})
			return
		}
		resp, _ := planbrain.Brain{}.Decide(req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	fmt.Printf("planbrain (tvcp-ai/1, rollout planner) on http://%s/v1/decide\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
