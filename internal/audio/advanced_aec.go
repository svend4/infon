package audio

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// AdvancedAEC provides sophisticated echo cancellation using NLMS and RLS algorithms
// Offers superior performance compared to basic LMS for acoustic echo cancellation
type AdvancedAEC struct {
	mu sync.RWMutex

	// Configuration
	algorithm    AECAlgorithm
	sampleRate   int
	filterLength int

	// Algorithm-specific processors
	nlms *NLMSProcessor
	rls  *RLSProcessor

	// Statistics
	stats AECStatistics
}

// AECAlgorithm specifies the echo cancellation algorithm
type AECAlgorithm string

const (
	AlgorithmNLMS AECAlgorithm = "nlms" // Normalized Least Mean Squares
	AlgorithmRLS  AECAlgorithm = "rls"  // Recursive Least Squares
)

// AECStatistics tracks echo cancellation performance
type AECStatistics struct {
	FramesProcessed   uint64
	EchoDetected      uint64
	EchoReturned      float64 // Echo return loss (dB)
	ERLE              float64 // Echo Return Loss Enhancement (dB)
	AverageProcessTime time.Duration
}

// AECConfig holds configuration for advanced AEC
type AECConfig struct {
	Algorithm    AECAlgorithm
	SampleRate   int
	FilterLength int // Adaptive filter taps (typical: 512-2048)
}

// NewAdvancedAEC creates a new advanced echo canceller
func NewAdvancedAEC(config AECConfig) (*AdvancedAEC, error) {
	if config.FilterLength < 64 || config.FilterLength > 8192 {
		return nil, fmt.Errorf("filter length must be 64-8192, got %d", config.FilterLength)
	}

	aec := &AdvancedAEC{
		algorithm:    config.Algorithm,
		sampleRate:   config.SampleRate,
		filterLength: config.FilterLength,
	}

	// Initialize algorithm-specific processor
	switch config.Algorithm {
	case AlgorithmNLMS:
		aec.nlms = NewNLMSProcessor(config.FilterLength)
	case AlgorithmRLS:
		aec.rls = NewRLSProcessor(config.FilterLength)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", config.Algorithm)
	}

	return aec, nil
}

// Process processes a frame of microphone audio and removes echo
// capture: microphone input
// reference: speaker output (reference for echo)
// Returns: echo-canceled output and echo detection flag
func (aec *AdvancedAEC) Process(capture, reference []int16) ([]int16, bool, error) {
	start := time.Now()

	output := make([]int16, len(capture))
	echoDetected := false

	// Convert to float64 for processing
	captureFloat := int16ToFloat64(capture)
	referenceFloat := int16ToFloat64(reference)

	// Process based on algorithm
	var outputFloat []float64
	switch aec.algorithm {
	case AlgorithmNLMS:
		outputFloat, echoDetected = aec.nlms.Process(captureFloat, referenceFloat)
	case AlgorithmRLS:
		outputFloat, echoDetected = aec.rls.Process(captureFloat, referenceFloat)
	default:
		return nil, false, fmt.Errorf("unsupported algorithm")
	}

	// Convert back to int16
	output = float64ToInt16(outputFloat)

	// Update statistics
	processTime := time.Since(start)
	aec.mu.Lock()
	aec.stats.FramesProcessed++
	if echoDetected {
		aec.stats.EchoDetected++
	}
	aec.stats.AverageProcessTime = (aec.stats.AverageProcessTime*time.Duration(aec.stats.FramesProcessed-1) +
		processTime) / time.Duration(aec.stats.FramesProcessed)
	aec.mu.Unlock()

	return output, echoDetected, nil
}

// GetStatistics returns echo cancellation statistics
func (aec *AdvancedAEC) GetStatistics() AECStatistics {
	aec.mu.RLock()
	defer aec.mu.RUnlock()
	return aec.stats
}

// Reset resets the echo canceller state
func (aec *AdvancedAEC) Reset() {
	aec.mu.Lock()
	defer aec.mu.Unlock()

	if aec.nlms != nil {
		aec.nlms.Reset()
	}
	if aec.rls != nil {
		aec.rls.Reset()
	}

	aec.stats = AECStatistics{}
}

// GetAlgorithm returns the current algorithm
func (aec *AdvancedAEC) GetAlgorithm() AECAlgorithm {
	aec.mu.RLock()
	defer aec.mu.RUnlock()
	return aec.algorithm
}

// NLMS (Normalized Least Mean Squares) Processor

// NLMSProcessor implements the NLMS adaptive filtering algorithm
type NLMSProcessor struct {
	mu sync.RWMutex

	filterLength int
	coeffs       []float64 // Adaptive filter coefficients
	stepSize     float64   // Learning rate (mu)
	delta        float64   // Regularization parameter

	// Input buffer
	inputBuffer []float64
	bufferPos   int

	// Echo estimation
	echoThreshold float64
}

// NewNLMSProcessor creates a new NLMS processor
func NewNLMSProcessor(filterLength int) *NLMSProcessor {
	return &NLMSProcessor{
		filterLength:  filterLength,
		coeffs:        make([]float64, filterLength),
		stepSize:      0.5,  // Typical range: 0.1-0.9
		delta:         0.01, // Regularization to avoid division by zero
		inputBuffer:   make([]float64, filterLength),
		bufferPos:     0,
		echoThreshold: 0.01,
	}
}

// Process applies NLMS filtering
func (nlms *NLMSProcessor) Process(capture, reference []float64) ([]float64, bool) {
	nlms.mu.Lock()
	defer nlms.mu.Unlock()

	output := make([]float64, len(capture))
	echoDetected := false

	for i := 0; i < len(capture); i++ {
		// Add reference to circular buffer
		nlms.inputBuffer[nlms.bufferPos] = reference[i]
		nlms.bufferPos = (nlms.bufferPos + 1) % nlms.filterLength

		// Calculate echo estimate (y = w^T * x)
		var echoEstimate float64
		for j := 0; j < nlms.filterLength; j++ {
			idx := (nlms.bufferPos - j - 1 + nlms.filterLength) % nlms.filterLength
			echoEstimate += nlms.coeffs[j] * nlms.inputBuffer[idx]
		}

		// Calculate error (e = d - y)
		errorSignal := capture[i] - echoEstimate

		// Calculate input power (||x||^2)
		var inputPower float64
		for j := 0; j < nlms.filterLength; j++ {
			val := nlms.inputBuffer[j]
			inputPower += val * val
		}

		// NLMS update: w(n+1) = w(n) + (mu / (delta + ||x||^2)) * e(n) * x(n)
		normalizedStepSize := nlms.stepSize / (nlms.delta + inputPower)

		for j := 0; j < nlms.filterLength; j++ {
			idx := (nlms.bufferPos - j - 1 + nlms.filterLength) % nlms.filterLength
			nlms.coeffs[j] += normalizedStepSize * errorSignal * nlms.inputBuffer[idx]
		}

		// Check for echo
		if math.Abs(echoEstimate) > nlms.echoThreshold {
			echoDetected = true
		}

		output[i] = errorSignal
	}

	return output, echoDetected
}

// Reset resets NLMS processor state
func (nlms *NLMSProcessor) Reset() {
	nlms.mu.Lock()
	defer nlms.mu.Unlock()

	for i := range nlms.coeffs {
		nlms.coeffs[i] = 0
	}
	for i := range nlms.inputBuffer {
		nlms.inputBuffer[i] = 0
	}
	nlms.bufferPos = 0
}

// SetStepSize sets the NLMS learning rate
func (nlms *NLMSProcessor) SetStepSize(stepSize float64) error {
	if stepSize <= 0 || stepSize >= 1 {
		return fmt.Errorf("step size must be 0-1, got %f", stepSize)
	}

	nlms.mu.Lock()
	defer nlms.mu.Unlock()
	nlms.stepSize = stepSize
	return nil
}

// RLS (Recursive Least Squares) Processor

// RLSProcessor implements the RLS adaptive filtering algorithm
type RLSProcessor struct {
	mu sync.RWMutex

	filterLength int
	coeffs       []float64   // Adaptive filter coefficients
	P            [][]float64 // Inverse correlation matrix
	lambda       float64     // Forgetting factor (0.98-0.9999)
	delta        float64     // Initialization parameter for P

	// Input buffer
	inputBuffer []float64
	bufferPos   int

	// Echo estimation
	echoThreshold float64
}

// NewRLSProcessor creates a new RLS processor
func NewRLSProcessor(filterLength int) *RLSProcessor {
	// Initialize P as identity matrix scaled by delta
	P := make([][]float64, filterLength)
	delta := 1.0 // Large initial value
	for i := 0; i < filterLength; i++ {
		P[i] = make([]float64, filterLength)
		P[i][i] = delta
	}

	return &RLSProcessor{
		filterLength:  filterLength,
		coeffs:        make([]float64, filterLength),
		P:             P,
		lambda:        0.99,  // Typical: 0.98-0.9999
		delta:         1.0,
		inputBuffer:   make([]float64, filterLength),
		bufferPos:     0,
		echoThreshold: 0.01,
	}
}

// Process applies RLS filtering
func (rls *RLSProcessor) Process(capture, reference []float64) ([]float64, bool) {
	rls.mu.Lock()
	defer rls.mu.Unlock()

	output := make([]float64, len(capture))
	echoDetected := false

	for i := 0; i < len(capture); i++ {
		// Add reference to circular buffer
		rls.inputBuffer[rls.bufferPos] = reference[i]
		rls.bufferPos = (rls.bufferPos + 1) % rls.filterLength

		// Get current input vector x(n)
		x := make([]float64, rls.filterLength)
		for j := 0; j < rls.filterLength; j++ {
			idx := (rls.bufferPos - j - 1 + rls.filterLength) % rls.filterLength
			x[j] = rls.inputBuffer[idx]
		}

		// Calculate echo estimate: y(n) = w^T(n) * x(n)
		var echoEstimate float64
		for j := 0; j < rls.filterLength; j++ {
			echoEstimate += rls.coeffs[j] * x[j]
		}

		// Calculate a priori error: e(n) = d(n) - y(n)
		errorSignal := capture[i] - echoEstimate

		// RLS Update:
		// 1. Calculate k(n) = P(n-1)*x(n) / (lambda + x^T(n)*P(n-1)*x(n))
		// 2. Update coefficients: w(n) = w(n-1) + k(n)*e(n)
		// 3. Update P: P(n) = (1/lambda) * (P(n-1) - k(n)*x^T(n)*P(n-1))

		// Calculate P*x
		Px := make([]float64, rls.filterLength)
		for j := 0; j < rls.filterLength; j++ {
			for k := 0; k < rls.filterLength; k++ {
				Px[j] += rls.P[j][k] * x[k]
			}
		}

		// Calculate x^T*P*x (scalar)
		var xTPx float64
		for j := 0; j < rls.filterLength; j++ {
			xTPx += x[j] * Px[j]
		}

		// Calculate Kalman gain: k = Px / (lambda + xTPx)
		k := make([]float64, rls.filterLength)
		denominator := rls.lambda + xTPx
		for j := 0; j < rls.filterLength; j++ {
			k[j] = Px[j] / denominator
		}

		// Update coefficients: w = w + k*e
		for j := 0; j < rls.filterLength; j++ {
			rls.coeffs[j] += k[j] * errorSignal
		}

		// Update P matrix: P = (1/lambda) * (P - k*x^T*P)
		// Simplified update to avoid matrix operations
		factor := 1.0 / rls.lambda
		for j := 0; j < rls.filterLength; j++ {
			for m := 0; m < rls.filterLength; m++ {
				rls.P[j][m] = factor * (rls.P[j][m] - k[j]*Px[m])
			}
		}

		// Check for echo
		if math.Abs(echoEstimate) > rls.echoThreshold {
			echoDetected = true
		}

		output[i] = errorSignal
	}

	return output, echoDetected
}

// Reset resets RLS processor state
func (rls *RLSProcessor) Reset() {
	rls.mu.Lock()
	defer rls.mu.Unlock()

	// Reset coefficients
	for i := range rls.coeffs {
		rls.coeffs[i] = 0
	}

	// Reset P to identity * delta
	for i := 0; i < rls.filterLength; i++ {
		for j := 0; j < rls.filterLength; j++ {
			if i == j {
				rls.P[i][j] = rls.delta
			} else {
				rls.P[i][j] = 0
			}
		}
	}

	// Reset input buffer
	for i := range rls.inputBuffer {
		rls.inputBuffer[i] = 0
	}
	rls.bufferPos = 0
}

// SetForgettingFactor sets the RLS forgetting factor
func (rls *RLSProcessor) SetForgettingFactor(lambda float64) error {
	if lambda <= 0 || lambda >= 1 {
		return fmt.Errorf("forgetting factor must be 0-1, got %f", lambda)
	}

	rls.mu.Lock()
	defer rls.mu.Unlock()
	rls.lambda = lambda
	return nil
}

// Utility functions

// int16ToFloat64 converts int16 samples to float64
func int16ToFloat64(input []int16) []float64 {
	output := make([]float64, len(input))
	for i, sample := range input {
		output[i] = float64(sample) / 32768.0 // Normalize to [-1, 1]
	}
	return output
}

// float64ToInt16 converts float64 samples to int16
func float64ToInt16(input []float64) []int16 {
	output := make([]int16, len(input))
	for i, sample := range input {
		// Clamp to [-1, 1] and scale to int16 range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		output[i] = int16(sample * 32767.0)
	}
	return output
}

// Preset Configurations

// DefaultNLMSConfig returns default NLMS configuration
func DefaultNLMSConfig(sampleRate int) AECConfig {
	return AECConfig{
		Algorithm:    AlgorithmNLMS,
		SampleRate:   sampleRate,
		FilterLength: 512, // ~32ms at 16kHz
	}
}

// DefaultRLSConfig returns default RLS configuration
func DefaultRLSConfig(sampleRate int) AECConfig {
	return AECConfig{
		Algorithm:    AlgorithmRLS,
		SampleRate:   sampleRate,
		FilterLength: 256, // Shorter for RLS due to higher complexity
	}
}

// HighQualityNLMSConfig returns high-quality NLMS configuration
func HighQualityNLMSConfig(sampleRate int) AECConfig {
	return AECConfig{
		Algorithm:    AlgorithmNLMS,
		SampleRate:   sampleRate,
		FilterLength: 2048, // ~128ms at 16kHz
	}
}

// HighQualityRLSConfig returns high-quality RLS configuration
func HighQualityRLSConfig(sampleRate int) AECConfig {
	return AECConfig{
		Algorithm:    AlgorithmRLS,
		SampleRate:   sampleRate,
		FilterLength: 512, // ~32ms at 16kHz
	}
}
