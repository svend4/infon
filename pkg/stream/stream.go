// Package stream adds a streaming transport to tvcp-ai/1: instead of one
// request -> one response, a brain can PUSH a sequence of responses (animation
// frames, world ticks, progressive results) as Server-Sent Events over a single
// HTTP connection. It is stdlib-only and transport-symmetric with pkg/brain —
// the request shape is identical; only the reply is a stream. This closes the
// audit's "HTTP request/response only, no streaming/backpressure" gap.
package stream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/svend4/infon/pkg/brain"
)

// ServeResponses writes responses from frames to w as SSE until the channel
// closes. Each event is a single `data: <json>\n\n` record, flushed immediately.
func ServeResponses(w http.ResponseWriter, frames <-chan brain.Response) error {
	fl, ok := w.(http.Flusher)
	if !ok {
		return errors.New("stream: ResponseWriter does not support flushing")
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	for resp := range frames {
		b, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return err
		}
		fl.Flush()
	}
	return nil
}

// Handler builds an SSE endpoint from a generator: given the decoded request it
// returns a channel of responses to stream (and is expected to close it).
func Handler(gen func(brain.Request) <-chan brain.Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req brain.Request
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		_ = ServeResponses(w, gen(req))
	}
}

// Subscribe POSTs req to an SSE endpoint and returns a channel of decoded
// responses. The channel closes when the stream ends, the context is cancelled,
// or an error occurs. Reading at the consumer's pace provides natural backpressure.
func Subscribe(ctx context.Context, url string, req brain.Request) (<-chan brain.Response, error) {
	if req.Protocol == "" {
		req.Protocol = brain.Protocol
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(hr)
	if err != nil {
		return nil, err
	}
	out := make(chan brain.Response)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			var rsp brain.Response
			if json.Unmarshal([]byte(data), &rsp) != nil {
				continue
			}
			select {
			case out <- rsp:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// Frames is a small helper: turn a slice of responses into a closed channel,
// handy for generators that already have all frames in hand.
func Frames(rs []brain.Response) <-chan brain.Response {
	ch := make(chan brain.Response, len(rs))
	for _, r := range rs {
		ch <- r
	}
	close(ch)
	return ch
}
