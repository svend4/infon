package raydir

import (
	"sync"
	"testing"
)

// concurrency_test.go exercises the two concurrent types the way cmd/raymeet uses
// them — a receive goroutine, the main loop, and the capture/playback goroutines
// all touching the same value. Run under the race detector (CI runs `go test
// -race ./...`) these guard the locking in ChatSync and VoiceMixer.

func TestChatSyncConcurrent(t *testing.T) {
	cs := NewChatSync(0xAA, true)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				switch i % 4 {
				case 0:
					cs.Compose("me", "hi")
				case 1:
					cs.Observe(ChatMsg{ID: uint64(g)<<32 | uint64(i), Sender: "x", Text: "y"})
				case 2:
					cs.Observe(ChatMsg{ID: 7, Sender: "dup", Text: "z"}) // contended duplicate
				default:
					_ = cs.Recent()
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestVoiceMixerConcurrent(t *testing.T) {
	m := NewVoiceMixer(64)
	var wg sync.WaitGroup
	for s := uint32(1); s <= 6; s++ { // producers (speakers)
		wg.Add(1)
		go func(s uint32) {
			defer wg.Done()
			frame := make([]int16, 64)
			for i := 0; i < 3000; i++ {
				m.Add(s, frame)
			}
		}(s)
	}
	for c := 0; c < 2; c++ { // consumers (playback pull)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 6000; i++ {
				_ = m.Mix()
				_ = m.Active()
			}
		}()
	}
	wg.Wait()
}
