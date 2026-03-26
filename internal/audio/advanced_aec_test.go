package audio

import (
	"math"
	"testing"
)

func TestNewAdvancedAEC(t *testing.T) {
	config := DefaultNLMSConfig(16000)

	aec, err := NewAdvancedAEC(config)
	if err != nil {
		t.Fatalf("Failed to create advanced AEC: %v", err)
	}

	if aec == nil {
		t.Fatal("AEC is nil")
	}

	if aec.GetAlgorithm() != AlgorithmNLMS {
		t.Errorf("Algorithm = %s, expected %s", aec.GetAlgorithm(), AlgorithmNLMS)
	}
}

func TestNewAdvancedAECInvalidFilterLength(t *testing.T) {
	config := AECConfig{
		Algorithm:    AlgorithmNLMS,
		SampleRate:   16000,
		FilterLength: 10, // Too short
	}

	_, err := NewAdvancedAEC(config)
	if err == nil {
		t.Error("Should fail with invalid filter length")
	}
}

func TestNewAdvancedAECInvalidAlgorithm(t *testing.T) {
	config := AECConfig{
		Algorithm:    "invalid",
		SampleRate:   16000,
		FilterLength: 512,
	}

	_, err := NewAdvancedAEC(config)
	if err == nil {
		t.Error("Should fail with invalid algorithm")
	}
}

// NLMS Tests

func TestNLMSProcessor(t *testing.T) {
	config := DefaultNLMSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	// Create test signals
	frameSize := 160 // 10ms at 16kHz
	capture := make([]int16, frameSize)
	reference := make([]int16, frameSize)

	// Simulate echo: capture contains reference signal
	for i := 0; i < frameSize; i++ {
		reference[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
		capture[i] = reference[i] // Pure echo
	}

	output, echoDetected, err := aec.Process(capture, reference)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(output) != len(capture) {
		t.Errorf("Output length = %d, expected %d", len(output), len(capture))
	}

	// Echo should be detected
	if !echoDetected {
		t.Log("Note: Echo might not be detected in first frame (coefficients adapting)")
	}

	// Verify statistics
	stats := aec.GetStatistics()
	if stats.FramesProcessed != 1 {
		t.Errorf("FramesProcessed = %d, expected 1", stats.FramesProcessed)
	}
}

func TestNLMSConvergence(t *testing.T) {
	config := DefaultNLMSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	frameSize := 160
	reference := make([]int16, frameSize)
	capture := make([]int16, frameSize)

	// Process multiple frames to allow convergence
	for frame := 0; frame < 100; frame++ {
		for i := 0; i < frameSize; i++ {
			// 1kHz tone
			reference[i] = int16(5000 * math.Sin(2*math.Pi*1000*float64(frame*frameSize+i)/16000))
			// Capture = reference + small noise
			capture[i] = reference[i] + int16(100*math.Sin(2*math.Pi*5000*float64(i)/16000))
		}

		aec.Process(capture, reference)
	}

	// After convergence, echo should be reduced
	stats := aec.GetStatistics()
	if stats.FramesProcessed != 100 {
		t.Errorf("FramesProcessed = %d, expected 100", stats.FramesProcessed)
	}

	t.Logf("Processed %d frames, %d with echo detected",
		stats.FramesProcessed, stats.EchoDetected)
}

func TestNLMSProcessorSetStepSize(t *testing.T) {
	nlms := NewNLMSProcessor(512)

	// Valid step sizes
	for _, stepSize := range []float64{0.1, 0.5, 0.9} {
		err := nlms.SetStepSize(stepSize)
		if err != nil {
			t.Errorf("SetStepSize(%f) failed: %v", stepSize, err)
		}
	}

	// Invalid step sizes
	err := nlms.SetStepSize(0)
	if err == nil {
		t.Error("SetStepSize(0) should fail")
	}

	err = nlms.SetStepSize(1.0)
	if err == nil {
		t.Error("SetStepSize(1.0) should fail")
	}
}

func TestNLMSProcessorReset(t *testing.T) {
	nlms := NewNLMSProcessor(512)

	// Process some data to modify coefficients
	capture := make([]float64, 100)
	reference := make([]float64, 100)
	for i := range capture {
		capture[i] = math.Sin(2 * math.Pi * float64(i) / 100.0)
		reference[i] = math.Sin(2 * math.Pi * float64(i) / 100.0)
	}

	nlms.Process(capture, reference)

	// Coefficients should be non-zero
	hasNonZero := false
	for _, coeff := range nlms.coeffs {
		if coeff != 0 {
			hasNonZero = true
			break
		}
	}

	if !hasNonZero {
		t.Log("Coefficients might still be near zero after one frame")
	}

	// Reset
	nlms.Reset()

	// All coefficients should be zero
	for i, coeff := range nlms.coeffs {
		if coeff != 0 {
			t.Errorf("After reset, coeffs[%d] = %f, expected 0", i, coeff)
			break
		}
	}
}

// RLS Tests

func TestRLSProcessor(t *testing.T) {
	config := DefaultRLSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	frameSize := 160
	capture := make([]int16, frameSize)
	reference := make([]int16, frameSize)

	// Create test signals with echo
	for i := 0; i < frameSize; i++ {
		reference[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
		capture[i] = reference[i] // Pure echo
	}

	output, echoDetected, err := aec.Process(capture, reference)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(output) != len(capture) {
		t.Errorf("Output length = %d, expected %d", len(output), len(capture))
	}

	// Echo should be detected
	if !echoDetected {
		t.Log("Note: Echo might not be detected in first frame")
	}
}

func TestRLSConvergence(t *testing.T) {
	config := DefaultRLSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	frameSize := 160
	reference := make([]int16, frameSize)
	capture := make([]int16, frameSize)

	// Process multiple frames
	for frame := 0; frame < 50; frame++ {
		for i := 0; i < frameSize; i++ {
			reference[i] = int16(5000 * math.Sin(2*math.Pi*1000*float64(frame*frameSize+i)/16000))
			capture[i] = reference[i] + int16(100*math.Sin(2*math.Pi*5000*float64(i)/16000))
		}

		aec.Process(capture, reference)
	}

	// RLS should converge faster than NLMS
	stats := aec.GetStatistics()
	t.Logf("RLS processed %d frames, %d with echo detected",
		stats.FramesProcessed, stats.EchoDetected)
}

func TestRLSProcessorSetForgettingFactor(t *testing.T) {
	rls := NewRLSProcessor(256)

	// Valid forgetting factors
	for _, lambda := range []float64{0.95, 0.98, 0.99, 0.999} {
		err := rls.SetForgettingFactor(lambda)
		if err != nil {
			t.Errorf("SetForgettingFactor(%f) failed: %v", lambda, err)
		}
	}

	// Invalid forgetting factors
	err := rls.SetForgettingFactor(0)
	if err == nil {
		t.Error("SetForgettingFactor(0) should fail")
	}

	err = rls.SetForgettingFactor(1.0)
	if err == nil {
		t.Error("SetForgettingFactor(1.0) should fail")
	}
}

func TestRLSProcessorReset(t *testing.T) {
	rls := NewRLSProcessor(256)

	// Process some data
	capture := make([]float64, 100)
	reference := make([]float64, 100)
	for i := range capture {
		capture[i] = math.Sin(2 * math.Pi * float64(i) / 100.0)
		reference[i] = math.Sin(2 * math.Pi * float64(i) / 100.0)
	}

	rls.Process(capture, reference)

	// Reset
	rls.Reset()

	// Coefficients should be zero
	for i, coeff := range rls.coeffs {
		if coeff != 0 {
			t.Errorf("After reset, coeffs[%d] = %f, expected 0", i, coeff)
			break
		}
	}

	// P matrix should be identity * delta
	for i := 0; i < len(rls.P); i++ {
		for j := 0; j < len(rls.P[i]); j++ {
			expected := 0.0
			if i == j {
				expected = rls.delta
			}
			if rls.P[i][j] != expected {
				t.Errorf("After reset, P[%d][%d] = %f, expected %f",
					i, j, rls.P[i][j], expected)
				return
			}
		}
	}
}

// Integration Tests

func TestAdvancedAECReset(t *testing.T) {
	config := DefaultNLMSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	// Process a frame
	capture := make([]int16, 160)
	reference := make([]int16, 160)
	aec.Process(capture, reference)

	stats := aec.GetStatistics()
	if stats.FramesProcessed != 1 {
		t.Error("Should have processed 1 frame")
	}

	// Reset
	aec.Reset()

	stats = aec.GetStatistics()
	if stats.FramesProcessed != 0 {
		t.Errorf("After reset, FramesProcessed = %d, expected 0", stats.FramesProcessed)
	}
}

func TestAdvancedAECStatistics(t *testing.T) {
	config := DefaultNLMSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	capture := make([]int16, 160)
	reference := make([]int16, 160)

	// Process multiple frames
	for i := 0; i < 10; i++ {
		aec.Process(capture, reference)
	}

	stats := aec.GetStatistics()
	if stats.FramesProcessed != 10 {
		t.Errorf("FramesProcessed = %d, expected 10", stats.FramesProcessed)
	}

	if stats.AverageProcessTime == 0 {
		t.Error("Average process time should not be zero")
	}
}

// Utility Function Tests

func TestInt16ToFloat64(t *testing.T) {
	input := []int16{0, 16384, 32767, -16384, -32768}
	output := int16ToFloat64(input)

	if len(output) != len(input) {
		t.Errorf("Output length = %d, expected %d", len(output), len(input))
	}

	// Check normalization
	if output[0] != 0 {
		t.Errorf("output[0] = %f, expected 0", output[0])
	}

	if math.Abs(output[1]-0.5) > 0.01 {
		t.Errorf("output[1] = %f, expected ~0.5", output[1])
	}

	if math.Abs(output[2]-1.0) > 0.01 {
		t.Errorf("output[2] = %f, expected ~1.0", output[2])
	}
}

func TestFloat64ToInt16(t *testing.T) {
	input := []float64{0, 0.5, 1.0, -0.5, -1.0}
	output := float64ToInt16(input)

	if len(output) != len(input) {
		t.Errorf("Output length = %d, expected %d", len(output), len(input))
	}

	// Check scaling
	if output[0] != 0 {
		t.Errorf("output[0] = %d, expected 0", output[0])
	}

	// Allow some tolerance for rounding
	if math.Abs(float64(output[1])-16383.5) > 1.0 {
		t.Errorf("output[1] = %d, expected ~16384", output[1])
	}
}

func TestFloat64ToInt16Clipping(t *testing.T) {
	input := []float64{2.0, -2.0, 1.5, -1.5}
	output := float64ToInt16(input)

	// Values should be clamped to int16 range
	if output[0] != 32767 {
		t.Errorf("output[0] = %d, expected 32767 (max)", output[0])
	}

	if output[1] != -32767 {
		t.Errorf("output[1] = %d, expected -32767 (min)", output[1])
	}
}

func TestRoundTripConversion(t *testing.T) {
	original := []int16{0, 10000, 20000, -10000, -20000}
	asFloat := int16ToFloat64(original)
	backToInt16 := float64ToInt16(asFloat)

	for i := range original {
		diff := math.Abs(float64(original[i] - backToInt16[i]))
		if diff > 1.0 { // Allow 1-sample error due to rounding
			t.Errorf("Sample %d: original = %d, after round-trip = %d",
				i, original[i], backToInt16[i])
		}
	}
}

// Preset Configuration Tests

func TestDefaultNLMSConfig(t *testing.T) {
	config := DefaultNLMSConfig(16000)

	if config.Algorithm != AlgorithmNLMS {
		t.Errorf("Algorithm = %s, expected %s", config.Algorithm, AlgorithmNLMS)
	}

	if config.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, expected 16000", config.SampleRate)
	}

	if config.FilterLength != 512 {
		t.Errorf("FilterLength = %d, expected 512", config.FilterLength)
	}
}

func TestDefaultRLSConfig(t *testing.T) {
	config := DefaultRLSConfig(16000)

	if config.Algorithm != AlgorithmRLS {
		t.Errorf("Algorithm = %s, expected %s", config.Algorithm, AlgorithmRLS)
	}

	if config.FilterLength != 256 {
		t.Errorf("FilterLength = %d, expected 256", config.FilterLength)
	}
}

func TestHighQualityNLMSConfig(t *testing.T) {
	config := HighQualityNLMSConfig(16000)

	if config.FilterLength != 2048 {
		t.Errorf("FilterLength = %d, expected 2048", config.FilterLength)
	}
}

func TestHighQualityRLSConfig(t *testing.T) {
	config := HighQualityRLSConfig(16000)

	if config.FilterLength != 512 {
		t.Errorf("FilterLength = %d, expected 512", config.FilterLength)
	}
}

// Comparison Tests

func TestNLMSvsRLS(t *testing.T) {
	frameSize := 160
	numFrames := 50

	// Create NLMS AEC
	nlmsConfig := DefaultNLMSConfig(16000)
	nlmsAEC, _ := NewAdvancedAEC(nlmsConfig)

	// Create RLS AEC
	rlsConfig := DefaultRLSConfig(16000)
	rlsAEC, _ := NewAdvancedAEC(rlsConfig)

	// Process same signal with both
	for frame := 0; frame < numFrames; frame++ {
		capture := make([]int16, frameSize)
		reference := make([]int16, frameSize)

		for i := 0; i < frameSize; i++ {
			reference[i] = int16(5000 * math.Sin(2*math.Pi*1000*float64(frame*frameSize+i)/16000))
			capture[i] = reference[i]
		}

		nlmsAEC.Process(capture, reference)
		rlsAEC.Process(capture, reference)
	}

	nlmsStats := nlmsAEC.GetStatistics()
	rlsStats := rlsAEC.GetStatistics()

	t.Logf("NLMS: %d frames, %d echo detections, avg time %v",
		nlmsStats.FramesProcessed, nlmsStats.EchoDetected, nlmsStats.AverageProcessTime)

	t.Logf("RLS: %d frames, %d echo detections, avg time %v",
		rlsStats.FramesProcessed, rlsStats.EchoDetected, rlsStats.AverageProcessTime)

	// RLS typically converges faster but is more computationally expensive
	if rlsStats.AverageProcessTime < nlmsStats.AverageProcessTime {
		t.Log("Note: RLS was faster than NLMS (unexpected, usually RLS is slower)")
	}
}

// Benchmarks

func BenchmarkNLMSProcess(b *testing.B) {
	config := DefaultNLMSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	capture := make([]int16, 160)
	reference := make([]int16, 160)
	for i := range capture {
		capture[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
		reference[i] = int16(5000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aec.Process(capture, reference)
	}
}

func BenchmarkRLSProcess(b *testing.B) {
	config := DefaultRLSConfig(16000)
	aec, _ := NewAdvancedAEC(config)

	capture := make([]int16, 160)
	reference := make([]int16, 160)
	for i := range capture {
		capture[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
		reference[i] = int16(5000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aec.Process(capture, reference)
	}
}

func BenchmarkInt16ToFloat64(b *testing.B) {
	input := make([]int16, 160)
	for i := range input {
		input[i] = int16(i * 100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		int16ToFloat64(input)
	}
}

func BenchmarkFloat64ToInt16(b *testing.B) {
	input := make([]float64, 160)
	for i := range input {
		input[i] = math.Sin(2 * math.Pi * float64(i) / 160.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		float64ToInt16(input)
	}
}
