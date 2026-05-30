// Command brainserver serves the reference tvcp-ai/1 brain over HTTP, so any
// client (game, draw) can use it. Replace it with an adapter to Ollama/OpenAI/
// Anthropic to plug a real neural network in behind the same format.
//
//	brainserver [host:port]   # default 127.0.0.1:8088 ; POST /v1/decide
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/svend4/infon/pkg/brain"
)

func main() {
	addr := "127.0.0.1:8088"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	http.HandleFunc("/v1/decide", brain.Handler())
	fmt.Println("tvcp-ai/1 brain server on", addr, "- POST /v1/decide")
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
