package brain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// CapabilitiesHandler defaulted the protocol by assigning to its captured
// argument on every request, a data race between concurrent requests.
// Run with -race.
func TestAuditCapabilitiesHandlerConcurrent(t *testing.T) {
	h := CapabilitiesHandler(Capabilities{Kinds: []string{"move", "draw"}})
	var wg sync.WaitGroup
	errs := make(chan string, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil))
			var c Capabilities
			if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
				errs <- err.Error()
				return
			}
			if c.Protocol != Protocol {
				errs <- "protocol " + c.Protocol
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
