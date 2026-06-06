package stream

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func TestSSERoundTrip(t *testing.T) {
	gen := func(req brain.Request) <-chan brain.Response {
		ch := make(chan brain.Response)
		go func() {
			defer close(ch)
			for i := 0; i < 5; i++ {
				ch <- brain.Response{Protocol: brain.Protocol, Kind: "image",
					Reasoning: fmt.Sprintf("frame %d for %q", i, req.Prompt)}
			}
		}()
		return ch
	}
	srv := httptest.NewServer(Handler(gen))
	defer srv.Close()

	ch, err := Subscribe(context.Background(), srv.URL, brain.Request{Kind: "image", Prompt: "harbor"})
	if err != nil {
		t.Fatal(err)
	}
	var got []brain.Response
	for r := range ch {
		got = append(got, r)
	}
	if len(got) != 5 {
		t.Fatalf("received %d frames, want 5", len(got))
	}
	if got[0].Reasoning != `frame 0 for "harbor"` || got[4].Reasoning != `frame 4 for "harbor"` {
		t.Fatalf("stream order/content wrong: %q .. %q", got[0].Reasoning, got[4].Reasoning)
	}
}

func TestFramesHelper(t *testing.T) {
	n := 0
	for range Frames([]brain.Response{{Kind: "a"}, {Kind: "b"}, {Kind: "c"}}) {
		n++
	}
	if n != 3 {
		t.Fatalf("Frames yielded %d, want 3", n)
	}
}
