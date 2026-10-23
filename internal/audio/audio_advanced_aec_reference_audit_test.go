package audio

import (
	"fmt"
	"testing"
)

// The reference (far-end) frame length is chosen by the remote side and may
// be shorter than the capture frame or empty; AdvancedAEC must treat the
// missing reference as silence instead of panicking, for both algorithms.
func TestAdvancedAECAuditShortReference(t *testing.T) {
	seed := uint32(11)
	capture := auditNoiseFrame(&seed, 160, 4000)
	for _, cfg := range []AECConfig{DefaultNLMSConfig(16000), {Algorithm: AlgorithmRLS, SampleRate: 16000, FilterLength: 64}} {
		for _, refLen := range []int{0, 1, 80, 159} {
			t.Run(fmt.Sprintf("%s/ref%d", cfg.Algorithm, refLen), func(t *testing.T) {
				aec, err := NewAdvancedAEC(cfg)
				if err != nil {
					t.Fatal(err)
				}
				var ref []int16
				if refLen > 0 {
					ref = auditNoiseFrame(&seed, refLen, 4000)
				}
				var out []int16
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("Process(capture[160], reference[%d]) panicked: %v", refLen, r)
						}
					}()
					out, _, err = aec.Process(capture, ref)
				}()
				if err != nil {
					t.Fatalf("Process: %v", err)
				}
				if len(out) != len(capture) {
					t.Fatalf("output length %d, want %d", len(out), len(capture))
				}
			})
		}
	}
}
