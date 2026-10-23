package audio

import "testing"

// Steady background noise louder than twice the initial noise floor (a fan,
// HVAC, high microphone gain) must be learned as noise instead of being
// reported as speech forever, and speech-like bursts over that noise must
// still be detected afterwards.
func TestVADAuditLearnsLoudSteadyNoise(t *testing.T) {
	const (
		sampleRate = 16000
		frameLen   = 320 // 20 ms, as used by cmd/tvcp/call.go
	)
	vad := NewVAD(sampleRate)
	vad.SetSensitivity(0.7) // as in cmd/tvcp/call.go
	seed := uint32(3)

	// 5 s of steady noise at about 400 RMS; the last 2 s must be silence.
	for k := 0; k < 250; k++ {
		speaking := vad.Process(auditNoiseFrame(&seed, frameLen, 700))
		if k >= 150 && speaking {
			t.Fatalf("steady noise frame %d (%.2f s) reported as speech, noise floor %.0f",
				k, float64(k*frameLen)/sampleRate, vad.GetStatistics().NoiseFloor)
		}
	}

	// 10 s of speech-like bursts (400 ms voiced, 100 ms pause) over the same
	// noise: every voiced frame after the onset delay must be detected.
	for burst := 0; burst < 20; burst++ {
		for k := 0; k < 20; k++ {
			frame := auditNoiseFrame(&seed, frameLen, 700)
			tone := auditSineFrame(burst*20+k, frameLen, sampleRate, 220, 2500)
			for i := range frame {
				frame[i] += tone[i]
			}
			speaking := vad.Process(frame)
			if k >= 3 && !speaking {
				t.Fatalf("burst %d frame %d not detected as speech, noise floor %.0f",
					burst, k, vad.GetStatistics().NoiseFloor)
			}
		}
		for k := 0; k < 5; k++ {
			vad.Process(auditNoiseFrame(&seed, frameLen, 700))
		}
	}
}
