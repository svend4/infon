// Command capsdemo shows tvcp-ai/1 capability negotiation (pkg/brain): a brain
// serves /v1/capabilities and /v1/decide on one endpoint; a client fetches the
// capabilities and checks support before sending. Both ends run in-process.
//
//	go run ./cmd/capsdemo
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/svend4/infon/pkg/brain"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	srv := &http.Server{Handler: brain.Mux(brain.ReferenceCapabilities())}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	base := "http://" + ln.Addr().String()
	caps, err := brain.FetchCapabilities(base + "/v1/capabilities")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("brain at %s advertises %s:\n", base, caps.Protocol)
	fmt.Printf("  kinds:   %v\n  games:   %v\n  formats: %v\n", caps.Kinds, caps.Games, caps.Formats)

	fmt.Println("\nnegotiation before sending:")
	for _, q := range []struct{ kind, game string }{
		{"move", "rpg"}, {"image", ""}, {"move", "chess"}, {"sketch", ""}, {"dream", ""},
	} {
		ok := caps.Supports(q.kind, q.game)
		label := q.kind
		if q.game != "" {
			label += ":" + q.game
		}
		mark := "supported"
		if !ok {
			mark = "NOT supported -> pick another brain"
		}
		fmt.Printf("  %-14s %s\n", label, mark)
	}
}
