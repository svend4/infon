package screen

import (
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/svend4/infon/pkg/terminal"
)

func newSleepShare(t *testing.T) *ScreenShare {
	t.Helper()
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep command not available")
	}
	return NewScreenShare("sleep 5", 8, 4)
}

// waitClosed drains ch until it is closed, failing if that takes too long.
func waitClosed(t *testing.T, ch <-chan *terminal.Frame, what string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatalf("%s: frame channel not closed after Stop", what)
		}
	}
}

// A session can be stopped and started again: each session delivers frames on
// its channel, and that channel is closed once the session stops (so a
// consumer ranging over it terminates). Start after Stop must not send on the
// previous session's closed channel.
func TestScreenShareRestartAfterStop(t *testing.T) {
	ss := newSleepShare(t)
	for round := 0; round < 3; round++ {
		if err := ss.Start(); err != nil {
			t.Fatalf("round %d: Start: %v", round, err)
		}
		ch := ss.GetFrameChannel()
		select {
		case f, ok := <-ch:
			if !ok || f == nil {
				t.Fatalf("round %d: frame channel closed while running", round)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("round %d: no frame within 2s", round)
		}
		if err := ss.Stop(); err != nil {
			t.Fatalf("round %d: Stop: %v", round, err)
		}
		waitClosed(t, ch, "restart")
	}
}

// Stop racing the frame generator, plus readers polling the session from
// other goroutines, must neither panic nor trip the race detector.
func TestScreenShareConcurrentStop(t *testing.T) {
	ss := newSleepShare(t)
	for round := 0; round < 5; round++ {
		if err := ss.Start(); err != nil {
			t.Fatalf("round %d: Start: %v", round, err)
		}
		ch := ss.GetFrameChannel()
		var wg sync.WaitGroup
		stop := make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = ss.GetFrameChannel()
				_ = ss.IsRunning()
				_ = ss.GetBuffer()
				time.Sleep(time.Millisecond)
			}
		}()
		time.Sleep(time.Duration(40+20*round) * time.Millisecond)
		if err := ss.Stop(); err != nil {
			t.Fatalf("round %d: Stop: %v", round, err)
		}
		if err := ss.Stop(); err != nil { // idempotent
			t.Fatalf("round %d: second Stop: %v", round, err)
		}
		close(stop)
		wg.Wait()
		waitClosed(t, ch, "concurrent")
	}
}
