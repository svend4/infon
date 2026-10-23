package sync

import (
	"testing"
	"time"
)

// fakeClock replaces the package clock for the duration of the test and
// returns a function that steps it forward. Readiness and staleness are
// measured against it, so tests do not depend on how fast they run.
func fakeClock(t *testing.T) (advance func(time.Duration)) {
	t.Helper()
	cur := time.Unix(1_700_000_000, 0)
	old := clock
	clock = func() time.Time { return cur }
	t.Cleanup(func() { clock = old })
	return func(d time.Duration) { cur = cur.Add(d) }
}
