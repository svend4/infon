package audio

import (
	"sync"
	"testing"
)

// Enable/Disable and SetLevel are meant to be called from a control goroutine
// while the audio goroutine processes frames. Run under -race.
func TestWebRTCProcessorAuditConcurrentEnableAndLevel(t *testing.T) {
	p, err := NewWebRTCAudioProcessor(ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableNS:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ns := NewNoiseSuppressor(16000, 1)
	seed := uint32(19)
	frame := auditNoiseFrame(&seed, 160, 2000)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			if _, _, err := p.ProcessCapture(frame, nil); err != nil {
				t.Error(err)
				return
			}
			_ = p.ProcessRender(frame)
			ns.Process(frame)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			if i%2 == 0 {
				p.Disable()
			} else {
				p.Enable()
			}
			_ = p.SetNSLevel(i % 4)
			_ = ns.SetLevel(i % 4)
		}
	}()
	wg.Wait()
}
