package codec

import (
	"sync"
	"testing"
)

// The VideoCodec setters take the codec mutex, so they are meant to be callable
// while another goroutine encodes and decodes (adaptive bitrate). Encode/Decode
// must therefore not read or write codec state lock-free. Run with -race.
func TestCodecVideoAuditConcurrentSettersAndCoding(t *testing.T) {
	for _, name := range []string{"h264", "vp8"} {
		c, err := codecVideoAuditCtors[name](DefaultVideoConfig(CodecType(name), 160, 128))
		if err != nil {
			t.Fatalf("%s: constructor: %v", name, err)
		}
		vc := c.(VideoCodec)

		// Large enough for every resolution the setter goroutine selects.
		frame := make([]byte, 320*240*3/2)
		for i := range frame {
			frame[i] = byte(i%251 + 1)
		}

		const iterations = 100
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = vc.SetBitrate(500_000 + 10_000*(i%50))
				_ = vc.SetFramerate(float64(15 + i%30))
				_ = vc.SetKeyframeInterval(1 + i%5)
				if i%2 == 0 {
					_ = vc.SetResolution(160, 128)
				} else {
					_ = vc.SetResolution(320, 240)
				}
			}
		}()
		for g := 0; g < 2; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					enc, err := c.Encode(frame)
					if err != nil {
						t.Errorf("%s: Encode: %v", name, err)
						return
					}
					if _, err := c.Decode(enc); err != nil {
						t.Errorf("%s: Decode: %v", name, err)
						return
					}
				}
			}()
		}
		wg.Wait()

		if got := c.GetStatistics().FramesEncoded; got != 2*iterations {
			t.Errorf("%s: FramesEncoded = %d, want %d", name, got, 2*iterations)
		}
	}
}
