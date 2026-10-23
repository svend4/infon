package audio

import (
	"sync"
	"testing"
)

// In a call Process runs on the audio goroutine while the main goroutine reads
// GetStatistics for the status line. Run under -race.
func TestVADConcurrentStatisticsAudit(t *testing.T) {
	v := NewVAD(16000)
	frame := make([]int16, 320)
	for i := range frame {
		frame[i] = int16((i % 40) * 500)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			v.Process(frame)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			_ = v.GetStatistics()
			_ = v.IsSpeaking()
		}
	}()
	wg.Wait()
	if got := v.GetStatistics().TotalFrames; got != 2000 {
		t.Fatalf("TotalFrames = %d, want 2000", got)
	}
}
