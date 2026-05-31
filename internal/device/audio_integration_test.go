package device_test

import (
	"math"
	"testing"

	"github.com/svend4/infon/internal/audio/reactive"
	"github.com/svend4/infon/internal/device"
)

// Proves the audio and video subsystems connect end to end: features extracted
// from a real tone, pushed into the reactive generator, produce a valid frame.
func TestAudioFeaturesDriveGenerator(t *testing.T) {
	samples := make([]int16, 2048)
	for i := range samples {
		samples[i] = int16(28000.0 * math.Sin(float64(i)*2*math.Pi*100/16000))
	}
	f := reactive.Extract(samples, 16000)
	if f.Level == 0 {
		t.Fatal("expected non-zero level from a loud tone")
	}
	g := device.NewAudioReactiveGenerator()
	g.SetLevels(f.Level, f.Bass, f.Mid, f.Treble)
	if img, err := g.Generate(32, 32, 0, 0); err != nil || img == nil {
		t.Fatalf("generate: %v", err)
	}
}
