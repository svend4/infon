// Command conform checks whether a tvcp-ai brain satisfies the protocol
// contract. With no -url it tests the in-process reference brain; with -url it
// POSTs the standard battery to a live endpoint (Ollama / OpenAI / Anthropic
// adapter) and reports PASS/FAIL per case. Exit code is non-zero if any fails.
//
//	go run ./cmd/conform                                  # reference brain
//	go run ./cmd/conform -url http://127.0.0.1:8092/v1/decide   # a live model
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/svend4/infon/pkg/brain"
)

func main() {
	url := flag.String("url", "", "tvcp-ai/1 endpoint to test; empty = in-process reference brain")
	flag.Parse()

	decide := func(r brain.Request) (brain.Response, error) { return brain.Reference(r), nil }
	target := "reference (in-process)"
	if *url != "" {
		hb := brain.HTTPBrain{URL: *url, HTTP: &http.Client{Timeout: 120 * time.Second}}
		decide = hb.Decide
		target = *url
	}

	fmt.Printf("tvcp-ai conformance — target: %s\n\n", target)
	results, ok := brain.RunConformance(decide)
	pass := 0
	for _, r := range results {
		status := "PASS"
		if !r.Pass {
			status = "FAIL"
		} else {
			pass++
		}
		fmt.Printf("  [%s] %-16s %s\n", status, r.Name, r.Detail)
	}
	fmt.Printf("\n%d/%d passed — ", pass, len(results))
	if ok {
		fmt.Println("CONFORMANT")
		return
	}
	fmt.Println("NOT CONFORMANT")
	os.Exit(1)
}
