package audio

import (
	"math"
	"sync"
)

// TestAudioSource generates test tones for audio testing
type TestAudioSource struct {
	format   AudioFormat
	isOpen   bool
	mu       sync.Mutex
	position int
	tone     string // "sine", "beep", "silence"
}

// NewTestAudioSource creates a new test audio source
func NewTestAudioSource(format AudioFormat) *TestAudioSource {
	return &TestAudioSource{
		format: format,
		tone:   "sine",
	}
}

// Open opens the test audio source
func (t *TestAudioSource) Open() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isOpen {
		return ErrDeviceBusy
	}

	t.isOpen = true
	t.position = 0
	return nil
}

// Close closes the test audio source
func (t *TestAudioSource) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isOpen {
		return nil
	}

	t.isOpen = false
	return nil
}

// Read generates test audio samples
func (t *TestAudioSource) Read(buffer []int16) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isOpen {
		return 0, ErrDeviceNotOpen
	}

	switch t.tone {
	case "sine":
		t.generateSine(buffer, 440.0) // A4 note
	case "beep":
		t.generateBeep(buffer)
	case "silence":
		for i := range buffer {
			buffer[i] = 0
		}
	default:
		t.generateSine(buffer, 440.0)
	}

	return len(buffer), nil
}

// IsOpen returns whether the source is open
func (t *TestAudioSource) IsOpen() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isOpen
}

// GetFormat returns the audio format
func (t *TestAudioSource) GetFormat() AudioFormat {
	return t.format
}

// SetTone sets the type of test tone to generate
func (t *TestAudioSource) SetTone(tone string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tone = tone
}

// generateSine generates a sine wave at the specified frequency
func (t *TestAudioSource) generateSine(buffer []int16, frequency float64) {
	amplitude := int16(10000) // ~30% of max amplitude (32767)

	for i := range buffer {
		phase := 2.0 * math.Pi * frequency * float64(t.position) / float64(t.format.SampleRate)
		sample := float64(amplitude) * math.Sin(phase)
		buffer[i] = int16(sample)

		t.position++
		if t.position >= t.format.SampleRate {
			t.position = 0 // Reset to avoid overflow
		}
	}
}

// generateBeep generates a beeping pattern
func (t *TestAudioSource) generateBeep(buffer []int16) {
	amplitude := int16(10000)
	beepFreq := 800.0 // Hz
	beepDuration := t.format.SampleRate / 2 // 0.5 seconds on, 0.5 seconds off

	for i := range buffer {
		cyclePos := t.position % (beepDuration * 2)

		if cyclePos < beepDuration {
			// Beep on
			phase := 2.0 * math.Pi * beepFreq * float64(cyclePos) / float64(t.format.SampleRate)
			sample := float64(amplitude) * math.Sin(phase)
			buffer[i] = int16(sample)
		} else {
			// Beep off (silence)
			buffer[i] = 0
		}

		t.position++
	}
}

// TestAudioSink is a test audio playback device (discards audio)
type TestAudioSink struct {
	format      AudioFormat
	isOpen      bool
	mu          sync.Mutex
	totalFrames int64
}

// NewTestAudioSink creates a new test audio sink
func NewTestAudioSink(format AudioFormat) *TestAudioSink {
	return &TestAudioSink{
		format: format,
	}
}

// Open opens the test audio sink
func (t *TestAudioSink) Open() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isOpen {
		return ErrDeviceBusy
	}

	t.isOpen = true
	t.totalFrames = 0
	return nil
}

// Close closes the test audio sink
func (t *TestAudioSink) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isOpen {
		return nil
	}

	t.isOpen = false
	return nil
}

// Write accepts audio samples (discards them for testing)
func (t *TestAudioSink) Write(buffer []int16) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isOpen {
		return 0, ErrDeviceNotOpen
	}

	t.totalFrames += int64(len(buffer))
	return len(buffer), nil
}

// IsOpen returns whether the sink is open
func (t *TestAudioSink) IsOpen() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isOpen
}

// GetFormat returns the audio format
func (t *TestAudioSink) GetFormat() AudioFormat {
	return t.format
}

// GetTotalFrames returns the total number of frames written
func (t *TestAudioSink) GetTotalFrames() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalFrames
}
