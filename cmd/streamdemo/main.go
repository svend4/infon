// Command streamdemo shows the streaming tvcp-ai transport (pkg/stream): a brain
// PUSHES a sequence of responses over Server-Sent Events on one connection. It
// runs both ends in-process — a server that streams eight "frames" and a client
// that prints them as they arrive.
//
//	go run ./cmd/streamdemo
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/stream"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gen := func(req brain.Request) <-chan brain.Response {
		ch := make(chan brain.Response)
		go func() {
			defer close(ch)
			const n = 8
			for i := 0; i < n; i++ {
				ch <- brain.Response{Protocol: brain.Protocol, Kind: "image",
					Reasoning: fmt.Sprintf("frame %d/%d painting %q", i+1, n, req.Prompt)}
				time.Sleep(50 * time.Millisecond)
			}
		}()
		return ch
	}
	srv := &http.Server{Handler: stream.Handler(gen)}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	url := "http://" + ln.Addr().String() + "/v1/stream"
	fmt.Printf("tvcp-ai stream (SSE) at %s\n", url)
	ch, err := stream.Subscribe(context.Background(), url, brain.Request{Kind: "image", Prompt: "a calm harbor at dawn"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	n := 0
	for r := range ch {
		n++
		fmt.Printf("  recv: %s\n", r.Reasoning)
	}
	fmt.Printf("received %d streamed frames over one connection\n", n)
}
