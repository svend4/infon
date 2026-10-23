package audio

import (
	"math"
	"math/cmplx"
	"testing"
)

// allPassConfig returns a filter configuration that should leave the signal
// untouched: the passband covers DC..Nyquist and no noise is subtracted.
func allPassConfig(sampleRate, fftSize int) FilterConfig {
	return FilterConfig{
		SampleRate:      sampleRate,
		FFTSize:         fftSize,
		LowCutoffHz:     0,
		HighCutoffHz:    float64(sampleRate) / 2,
		SuppressionGain: 0,
	}
}

// With unit gain everywhere, applySpectralFiltering must keep the spectrum of
// a real frame Hermitian (X[N-k] == conj(X[k])), so the IFFT gives back the
// original frame, phase included.
func TestApplySpectralFilteringAllPassRoundTrip(t *testing.T) {
	const n = 64
	f, err := NewFFTBandPassFilter(allPassConfig(16000, n))
	if err != nil {
		t.Fatal(err)
	}

	x := make([]float64, n)
	for i := range x {
		ph := 2 * math.Pi * float64(i) / n
		x[i] = 1000*math.Sin(3*ph) + 500*math.Cos(7*ph+0.3) + 250*math.Sin(11*ph+1.1)
	}

	spectrum := fft(x)
	f.applySpectralFiltering(spectrum)

	for k := 1; k < n/2; k++ {
		if d := cmplx.Abs(spectrum[n-k] - cmplx.Conj(spectrum[k])); d > 1e-6 {
			t.Fatalf("bin %d is not the conjugate of bin %d: %v vs %v", n-k, k, spectrum[n-k], spectrum[k])
		}
	}

	y := ifft(spectrum)
	for i := range x {
		if math.Abs(y[i]-x[i]) > 1e-6 {
			t.Fatalf("sample %d: got %f, want %f", i, y[i], x[i])
		}
	}
}

// Process must return len(input) filtered samples per call whatever the call
// size, so a continuous signal fed in 20ms (or odd-sized) chunks comes out as
// the same continuous signal after a fixed latency, with no zeroed stretches.
func TestFFTBandPassFilterProcessStreamsAnyChunkSize(t *testing.T) {
	const (
		sampleRate = 16000
		fftSize    = 512
		total      = 16000
		amplitude  = 10000.0
	)

	signal := make([]int16, total)
	for i := range signal {
		ts := float64(i) / sampleRate
		signal[i] = int16(0.6*amplitude*math.Sin(2*math.Pi*440*ts) +
			0.4*amplitude*math.Sin(2*math.Pi*1250*ts+0.7))
	}

	for _, chunk := range []int{160, 320, 37, 1000} {
		f, err := NewFFTBandPassFilter(allPassConfig(sampleRate, fftSize))
		if err != nil {
			t.Fatal(err)
		}

		var out []int16
		for off := 0; off < total; off += chunk {
			end := off + chunk
			if end > total {
				end = total
			}
			got := f.Process(signal[off:end])
			if len(got) != end-off {
				t.Fatalf("chunk %d: Process returned %d samples for %d input", chunk, len(got), end-off)
			}
			out = append(out, got...)
		}

		// Find the delay at which the output best matches the input.
		const settle = 2 * fftSize
		bestDelay, bestErr := -1, math.Inf(1)
		for d := 0; d <= 2*fftSize; d++ {
			var sum float64
			for i := settle; i < total; i++ {
				sum += math.Abs(float64(out[i]) - float64(signal[i-d]))
			}
			if sum < bestErr {
				bestDelay, bestErr = d, sum
			}
		}

		if bestDelay != fftSize-1 {
			t.Errorf("chunk %d: latency = %d samples, want %d", chunk, bestDelay, fftSize-1)
		}

		// Allow for the Hann overlap-add ripple and int16 rounding.
		tol := 0.01 * amplitude
		for i := settle; i < total; i++ {
			if diff := math.Abs(float64(out[i]) - float64(signal[i-bestDelay])); diff > tol {
				t.Fatalf("chunk %d: sample %d differs from input delayed by %d by %.0f (tolerance %.0f)",
					chunk, i, bestDelay, diff, tol)
			}
		}
	}
}
