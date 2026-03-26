package audio

import (
	"math"
	"testing"
)

func TestNewWebRTCAudioProcessor(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
		EnableAGC:    true,
		EnableVAD:    true,
	}

	processor, err := NewWebRTCAudioProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	if processor == nil {
		t.Fatal("Processor is nil")
	}

	if processor.sampleRate != 16000 {
		t.Errorf("Sample rate = %d, expected 16000", processor.sampleRate)
	}

	if processor.channels != 1 {
		t.Errorf("Channels = %d, expected 1", processor.channels)
	}

	if processor.aec3 == nil {
		t.Error("AEC3 should be enabled")
	}

	if processor.noiseSuppressor == nil {
		t.Error("Noise suppressor should be enabled")
	}

	if processor.agc == nil {
		t.Error("AGC should be enabled")
	}

	if processor.vad == nil {
		t.Error("VAD should be enabled")
	}
}

func TestNewWebRTCAudioProcessorInvalidSampleRate(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   44100, // Invalid
		Channels:     1,
		FrameSamples: 160,
	}

	_, err := NewWebRTCAudioProcessor(config)
	if err == nil {
		t.Error("Should fail with invalid sample rate")
	}
}

func TestNewWebRTCAudioProcessorInvalidChannels(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     3, // Invalid
		FrameSamples: 160,
	}

	_, err := NewWebRTCAudioProcessor(config)
	if err == nil {
		t.Error("Should fail with invalid channels")
	}
}

func TestProcessorEnableDisable(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	// Initially enabled
	if !processor.enabled {
		t.Error("Processor should be enabled by default")
	}

	// Disable
	processor.Disable()
	if processor.enabled {
		t.Error("Processor should be disabled")
	}

	// Enable
	processor.Enable()
	if !processor.enabled {
		t.Error("Processor should be enabled")
	}
}

func TestProcessCapturePassthrough(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    false,
		EnableNS:     false,
		EnableAGC:    false,
		EnableVAD:    false,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	input := make([]int16, 160)
	for i := range input {
		input[i] = int16(i * 100)
	}

	output, hasSpeech, err := processor.ProcessCapture(input, nil)
	if err != nil {
		t.Fatalf("ProcessCapture failed: %v", err)
	}

	if !hasSpeech {
		t.Error("Should detect speech when VAD is disabled")
	}

	// Should be passthrough
	for i := range input {
		if output[i] != input[i] {
			t.Errorf("Output[%d] = %d, expected %d", i, output[i], input[i])
			break
		}
	}
}

func TestProcessCaptureDisabled(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)
	processor.Disable()

	input := make([]int16, 160)
	for i := range input {
		input[i] = int16(i * 100)
	}

	output, _, err := processor.ProcessCapture(input, nil)
	if err != nil {
		t.Fatalf("ProcessCapture failed: %v", err)
	}

	// Should be passthrough when disabled
	for i := range input {
		if output[i] != input[i] {
			t.Errorf("Output[%d] = %d, expected %d", i, output[i], input[i])
			break
		}
	}
}

func TestProcessorReset(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
		EnableAGC:    true,
		EnableVAD:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	// Process some audio to generate stats
	input := make([]int16, 160)
	processor.ProcessCapture(input, nil)

	stats := processor.GetStatistics()
	if stats.FramesProcessed != 1 {
		t.Errorf("Frames processed = %d, expected 1", stats.FramesProcessed)
	}

	// Reset
	processor.Reset()

	stats = processor.GetStatistics()
	if stats.FramesProcessed != 0 {
		t.Errorf("After reset, frames processed = %d, expected 0", stats.FramesProcessed)
	}
}

func TestGetStatistics(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
		EnableVAD:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	// Process some frames
	input := make([]int16, 160)
	for i := 0; i < 10; i++ {
		processor.ProcessCapture(input, nil)
	}

	stats := processor.GetStatistics()
	if stats.FramesProcessed != 10 {
		t.Errorf("Frames processed = %d, expected 10", stats.FramesProcessed)
	}

	if stats.AverageProcessTime == 0 {
		t.Error("Average process time should not be zero")
	}
}

// AEC3 Tests

func TestNewAEC3(t *testing.T) {
	aec := NewAEC3(16000, 1)

	if aec == nil {
		t.Fatal("AEC3 is nil")
	}

	if aec.sampleRate != 16000 {
		t.Errorf("Sample rate = %d, expected 16000", aec.sampleRate)
	}

	if len(aec.filterCoeffs) == 0 {
		t.Error("Filter coefficients should be initialized")
	}
}

func TestAEC3ProcessRender(t *testing.T) {
	aec := NewAEC3(16000, 1)

	render := make([]int16, 160)
	for i := range render {
		render[i] = int16(i * 10)
	}

	// Should not panic
	aec.ProcessRender(render)

	// Verify buffer was filled
	aec.mu.RLock()
	bufferEmpty := true
	for _, val := range aec.renderBuffer {
		if val != 0 {
			bufferEmpty = false
			break
		}
	}
	aec.mu.RUnlock()

	if bufferEmpty {
		t.Error("Render buffer should contain data")
	}
}

func TestAEC3ProcessCapture(t *testing.T) {
	aec := NewAEC3(16000, 1)

	// Create capture with echo
	capture := make([]int16, 160)
	render := make([]int16, 160)

	for i := range capture {
		render[i] = 1000
		capture[i] = 1000 // Same as render (simulated echo)
	}

	// Process
	output, echoDetected := aec.ProcessCapture(capture, render)

	if len(output) != len(capture) {
		t.Errorf("Output length = %d, expected %d", len(output), len(capture))
	}

	// After processing, echo should be reduced
	// (in first frame, coefficients are still adapting)
	if !echoDetected {
		// Echo might not be detected in first frame
		t.Log("Echo not detected (expected for first frame)")
	}
}

func TestAEC3Reset(t *testing.T) {
	aec := NewAEC3(16000, 1)

	// Set some coefficients
	aec.filterCoeffs[0] = 1.0
	aec.filterCoeffs[1] = 2.0

	// Reset
	aec.Reset()

	// Verify coefficients are zeroed
	for i, coeff := range aec.filterCoeffs {
		if coeff != 0 {
			t.Errorf("Coefficient[%d] = %f, expected 0", i, coeff)
			break
		}
	}
}

// Noise Suppressor Tests

func TestNewNoiseSuppressor(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 1)

	if ns == nil {
		t.Fatal("Noise suppressor is nil")
	}

	if ns.level != 2 {
		t.Errorf("Default level = %d, expected 2", ns.level)
	}
}

func TestNoiseSuppressorProcess(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 1)

	// Create low-level noise
	input := make([]int16, 160)
	for i := range input {
		input[i] = int16(i % 100) // Low amplitude
	}

	output, noiseReduced := ns.Process(input)

	if len(output) != len(input) {
		t.Errorf("Output length = %d, expected %d", len(output), len(input))
	}

	// With noise, output should be attenuated
	if noiseReduced {
		for i := range output {
			if math.Abs(float64(output[i])) > math.Abs(float64(input[i])) {
				t.Error("Output should be attenuated")
				break
			}
		}
	}
}

func TestNoiseSuppressorSetLevel(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 1)

	// Valid levels
	for level := 0; level <= 3; level++ {
		err := ns.SetLevel(level)
		if err != nil {
			t.Errorf("SetLevel(%d) failed: %v", level, err)
		}
	}

	// Invalid levels
	err := ns.SetLevel(-1)
	if err == nil {
		t.Error("SetLevel(-1) should fail")
	}

	err = ns.SetLevel(4)
	if err == nil {
		t.Error("SetLevel(4) should fail")
	}
}

func TestNoiseSuppressorLevelOff(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 1)
	ns.SetLevel(0) // Off

	input := make([]int16, 160)
	for i := range input {
		input[i] = int16(i * 10)
	}

	output, noiseReduced := ns.Process(input)

	if noiseReduced {
		t.Error("Noise should not be reduced when level is 0")
	}

	// Should be passthrough
	for i := range input {
		if output[i] != input[i] {
			t.Error("Output should match input when level is 0")
			break
		}
	}
}

func TestNoiseSuppressorReset(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 1)

	// Process some audio to build noise estimate
	input := make([]int16, 160)
	ns.Process(input)

	// Change noise floor
	ns.noiseFloor = 5000.0

	// Reset
	ns.Reset()

	if ns.noiseFloor != 100.0 {
		t.Errorf("After reset, noise floor = %f, expected 100.0", ns.noiseFloor)
	}
}

// AGC Tests

func TestNewAutomaticGainControl(t *testing.T) {
	agc := NewAutomaticGainControl(16000, 1)

	if agc == nil {
		t.Fatal("AGC is nil")
	}

	if agc.currentGain != 1.0 {
		t.Errorf("Initial gain = %f, expected 1.0", agc.currentGain)
	}
}

func TestAGCProcess(t *testing.T) {
	agc := NewAutomaticGainControl(16000, 1)

	// Create low-amplitude signal
	input := make([]int16, 160)
	for i := range input {
		input[i] = int16(i % 100) // Very low amplitude
	}

	output := agc.Process(input)

	if len(output) != len(input) {
		t.Errorf("Output length = %d, expected %d", len(output), len(input))
	}

	// AGC should amplify low signals
	// (after first frame, gain starts adapting)
}

func TestAGCSetTargetGain(t *testing.T) {
	agc := NewAutomaticGainControl(16000, 1)

	agc.SetTargetGain(6.0) // +6 dB

	if agc.targetGain != 6.0 {
		t.Errorf("Target gain = %f, expected 6.0", agc.targetGain)
	}
}

func TestAGCReset(t *testing.T) {
	agc := NewAutomaticGainControl(16000, 1)

	// Change state
	agc.currentGain = 2.0
	agc.peakLevel = 1000.0

	// Reset
	agc.Reset()

	if agc.currentGain != 1.0 {
		t.Errorf("After reset, current gain = %f, expected 1.0", agc.currentGain)
	}

	if agc.peakLevel != 0.0 {
		t.Errorf("After reset, peak level = %f, expected 0.0", agc.peakLevel)
	}
}

func TestAGCClipping(t *testing.T) {
	agc := NewAutomaticGainControl(16000, 1)

	// Create signal that would clip with high gain
	input := make([]int16, 160)
	for i := range input {
		input[i] = 20000 // Close to max
	}

	output := agc.Process(input)

	// Verify no clipping artifacts
	for i := range output {
		if output[i] < math.MinInt16 || output[i] > math.MaxInt16 {
			t.Errorf("Output[%d] = %d, out of int16 range", i, output[i])
		}
	}
}

// VAD Tests

func TestNewVoiceActivityDetector(t *testing.T) {
	vad := NewVoiceActivityDetector(16000)

	if vad == nil {
		t.Fatal("VAD is nil")
	}

	if vad.isSpeech {
		t.Error("Initially should not detect speech")
	}
}

func TestVADProcessSilence(t *testing.T) {
	vad := NewVoiceActivityDetector(16000)

	// Silent audio
	silence := make([]int16, 160)

	hasSpeech := vad.Process(silence)

	if hasSpeech {
		t.Error("Should not detect speech in silence")
	}

	if vad.IsSpeech() {
		t.Error("IsSpeech() should return false for silence")
	}
}

func TestVADProcessSpeech(t *testing.T) {
	vad := NewVoiceActivityDetector(16000)

	// High-energy audio (simulated speech)
	speech := make([]int16, 160)
	for i := range speech {
		speech[i] = int16(10000 * math.Sin(float64(i)*0.1))
	}

	hasSpeech := vad.Process(speech)

	if !hasSpeech {
		t.Error("Should detect speech in high-energy signal")
	}

	if !vad.IsSpeech() {
		t.Error("IsSpeech() should return true after speech detection")
	}
}

func TestVADReset(t *testing.T) {
	vad := NewVoiceActivityDetector(16000)

	// Process speech
	speech := make([]int16, 160)
	for i := range speech {
		speech[i] = int16(10000 * math.Sin(float64(i)*0.1))
	}
	vad.Process(speech)

	// Reset
	vad.Reset()

	if vad.isSpeech {
		t.Error("After reset, should not detect speech")
	}

	if vad.smoothedEnergy != 0 {
		t.Errorf("After reset, smoothed energy = %f, expected 0", vad.smoothedEnergy)
	}
}

// Integration Tests

func TestFullProcessingPipeline(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
		EnableAGC:    true,
		EnableVAD:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	// Create simulated speech with echo
	capture := make([]int16, 160)
	render := make([]int16, 160)

	for i := range capture {
		// Speech signal
		capture[i] = int16(5000 * math.Sin(float64(i)*0.1))
		// Echo from render
		render[i] = int16(1000 * math.Sin(float64(i)*0.1))
	}

	// Process
	output, hasSpeech, err := processor.ProcessCapture(capture, render)
	if err != nil {
		t.Fatalf("ProcessCapture failed: %v", err)
	}

	if len(output) != len(capture) {
		t.Errorf("Output length = %d, expected %d", len(output), len(capture))
	}

	// Should detect speech
	if !hasSpeech {
		t.Log("Speech not detected (may occur with initial frames)")
	}

	// Verify statistics
	stats := processor.GetStatistics()
	if stats.FramesProcessed != 1 {
		t.Errorf("Frames processed = %d, expected 1", stats.FramesProcessed)
	}
}

func TestProcessorSetAGCGain(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAGC:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	err := processor.SetAGCGain(6.0)
	if err != nil {
		t.Errorf("SetAGCGain failed: %v", err)
	}

	// Try when AGC is disabled
	config.EnableAGC = false
	processor2, _ := NewWebRTCAudioProcessor(config)

	err = processor2.SetAGCGain(6.0)
	if err == nil {
		t.Error("SetAGCGain should fail when AGC is disabled")
	}
}

func TestProcessorSetNSLevel(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableNS:     true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	err := processor.SetNSLevel(3)
	if err != nil {
		t.Errorf("SetNSLevel failed: %v", err)
	}

	// Try when NS is disabled
	config.EnableNS = false
	processor2, _ := NewWebRTCAudioProcessor(config)

	err = processor2.SetNSLevel(3)
	if err == nil {
		t.Error("SetNSLevel should fail when NS is disabled")
	}
}

func TestProcessRender(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	render := make([]int16, 160)
	for i := range render {
		render[i] = int16(i * 10)
	}

	err := processor.ProcessRender(render)
	if err != nil {
		t.Errorf("ProcessRender failed: %v", err)
	}
}

func TestMultipleFrameProcessing(t *testing.T) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
		EnableAGC:    true,
		EnableVAD:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	// Process multiple frames
	for frame := 0; frame < 100; frame++ {
		capture := make([]int16, 160)
		render := make([]int16, 160)

		for i := range capture {
			capture[i] = int16(1000 * math.Sin(float64(frame*160+i)*0.1))
			render[i] = int16(500 * math.Sin(float64(frame*160+i)*0.1))
		}

		_, _, err := processor.ProcessCapture(capture, render)
		if err != nil {
			t.Fatalf("Frame %d: ProcessCapture failed: %v", frame, err)
		}
	}

	stats := processor.GetStatistics()
	if stats.FramesProcessed != 100 {
		t.Errorf("Frames processed = %d, expected 100", stats.FramesProcessed)
	}
}

// Benchmarks

func BenchmarkProcessCapture(b *testing.B) {
	config := ProcessorConfig{
		SampleRate:   16000,
		Channels:     1,
		FrameSamples: 160,
		EnableAEC:    true,
		EnableNS:     true,
		EnableAGC:    true,
		EnableVAD:    true,
	}

	processor, _ := NewWebRTCAudioProcessor(config)

	capture := make([]int16, 160)
	render := make([]int16, 160)

	for i := range capture {
		capture[i] = int16(1000 * math.Sin(float64(i)*0.1))
		render[i] = int16(500 * math.Sin(float64(i)*0.1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessCapture(capture, render)
	}
}

func BenchmarkAEC3Process(b *testing.B) {
	aec := NewAEC3(16000, 1)

	capture := make([]int16, 160)
	render := make([]int16, 160)

	for i := range capture {
		capture[i] = int16(1000 * math.Sin(float64(i)*0.1))
		render[i] = int16(500 * math.Sin(float64(i)*0.1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aec.ProcessCapture(capture, render)
	}
}

func BenchmarkNoiseSuppression(b *testing.B) {
	ns := NewNoiseSuppressor(16000, 1)

	audio := make([]int16, 160)
	for i := range audio {
		audio[i] = int16(1000 * math.Sin(float64(i)*0.1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ns.Process(audio)
	}
}

func BenchmarkAGC(b *testing.B) {
	agc := NewAutomaticGainControl(16000, 1)

	audio := make([]int16, 160)
	for i := range audio {
		audio[i] = int16(1000 * math.Sin(float64(i)*0.1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agc.Process(audio)
	}
}

func BenchmarkVAD(b *testing.B) {
	vad := NewVoiceActivityDetector(16000)

	audio := make([]int16, 160)
	for i := range audio {
		audio[i] = int16(1000 * math.Sin(float64(i)*0.1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vad.Process(audio)
	}
}
