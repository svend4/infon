// Package reactive extracts compact spectral features from audio frames so the
// graphics layer can react to sound (roadmap C4): a plasma that pulses to a
// voice, or a prompt that shifts with energy. It is self-contained (its own
// small FFT) to stay decoupled from the band-pass filter and free of cross-deps.
package reactive

import "math"

// Features summarizes one audio frame for visual reactivity. All values are
// normalized to roughly 0..1 for easy mapping to colors/motion.
type Features struct {
	Level  float64 // overall RMS loudness (0..1)
	Bass   float64 // energy in the low band (0..1)
	Mid    float64 // energy in the mid band (0..1)
	Treble float64 // energy in the high band (0..1)
}

// Extract computes Features from PCM samples (int16) at sampleRate Hz. It applies
// a Hann window, runs an FFT, and integrates magnitude over three bands. An
// empty input returns the zero Features.
func Extract(samples []int16, sampleRate int) Features {
	if len(samples) == 0 || sampleRate <= 0 {
		return Features{}
	}

	// RMS level (pre-FFT), normalized against full-scale int16.
	var sumSq float64
	for _, s := range samples {
		v := float64(s)
		sumSq += v * v
	}
	rms := math.Sqrt(sumSq / float64(len(samples)))
	level := clamp01(rms / 32768.0 * 4.0) // ×4: speech rarely hits full scale

	// Window + pad to a power of two for the FFT.
	n := nextPow2(len(samples))
	buf := make([]float64, n)
	win := hann(len(samples))
	for i := 0; i < len(samples); i++ {
		buf[i] = float64(samples[i]) / 32768.0 * win[i]
	}

	spec := fftReal(buf)
	half := n / 2

	// Band edges in Hz -> bin indices. freq(bin) = bin * sampleRate / n.
	binHz := float64(sampleRate) / float64(n)
	bassHi := int(250.0 / binHz)
	midHi := int(2000.0 / binHz)
	bass := bandEnergy(spec, 1, bassHi, half)
	mid := bandEnergy(spec, bassHi, midHi, half)
	treble := bandEnergy(spec, midHi, half, half)

	// Normalize bands by a soft reference so typical audio spans 0..1.
	norm := func(e float64, bins int) float64 {
		if bins <= 0 {
			return 0
		}
		return clamp01(math.Sqrt(e/float64(bins)) * 6.0)
	}
	return Features{
		Level:  level,
		Bass:   norm(bass, bassHi-1),
		Mid:    norm(mid, midHi-bassHi),
		Treble: norm(treble, half-midHi),
	}
}

func bandEnergy(spec []complex128, lo, hi, half int) float64 {
	if lo < 1 {
		lo = 1
	}
	if hi > half {
		hi = half
	}
	var e float64
	for i := lo; i < hi; i++ {
		re := real(spec[i])
		im := imag(spec[i])
		e += re*re + im*im
	}
	return e
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func hann(n int) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	for i := 0; i < n; i++ {
		w[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
	}
	return w
}

func nextPow2(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// fftReal runs a radix-2 Cooley–Tukey FFT on real input (length must be a power
// of two). Returns the complex spectrum.
func fftReal(in []float64) []complex128 {
	n := len(in)
	x := make([]complex128, n)
	for i, v := range in {
		x[i] = complex(v, 0)
	}
	fftInPlace(x)
	return x
}

func fftInPlace(x []complex128) {
	n := len(x)
	if n <= 1 {
		return
	}
	// Bit-reversal permutation.
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	// Butterfly.
	for length := 2; length <= n; length <<= 1 {
		ang := -2 * math.Pi / float64(length)
		wlen := complex(math.Cos(ang), math.Sin(ang))
		for i := 0; i < n; i += length {
			w := complex(1, 0)
			for k := 0; k < length/2; k++ {
				u := x[i+k]
				v := x[i+k+length/2] * w
				x[i+k] = u + v
				x[i+k+length/2] = u - v
				w *= wlen
			}
		}
	}
}
