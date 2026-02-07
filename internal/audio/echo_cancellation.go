package audio

import (
	"math"
)

// EchoCanceller removes acoustic echo using adaptive filtering
type EchoCanceller struct {
	sampleRate int
	frameSize  int

	// Adaptive filter (simplified LMS - Least Mean Squares)
	filterLength   int
	filterWeights  []float64
	referenceBuffer []float64

	// Parameters
	stepSize       float64 // LMS adaptation step size
	regularization float64 // Regularization factor

	// State
	initialized bool
	frameCount  int

	// Statistics
	totalFrames       uint64
	echoDetected      uint64
	echoSuppressed    uint64
	avgEchoReduction  float64
}

// NewEchoCanceller creates a new echo canceller
func NewEchoCanceller(sampleRate, frameSize int) *EchoCanceller {
	// Filter length ~100ms for good echo cancellation
	filterLength := sampleRate / 10 // 100ms at 16kHz = 1600 taps

	return &EchoCanceller{
		sampleRate:      sampleRate,
		frameSize:       frameSize,
		filterLength:    filterLength,
		filterWeights:   make([]float64, filterLength),
		referenceBuffer: make([]float64, filterLength),
		stepSize:        0.01,   // Conservative step size
		regularization:  1e-6,   // Small regularization
		initialized:     false,
		frameCount:      0,
	}
}

// Process applies echo cancellation to input signal
// reference: audio being played (far-end)
// input: audio being captured (near-end + echo)
// returns: echo-cancelled output
func (ec *EchoCanceller) Process(input, reference []int16) []int16 {
	ec.frameCount++
	ec.totalFrames++

	// Convert to float64
	inputFloat := make([]float64, len(input))
	referenceFloat := make([]float64, len(reference))

	for i := range input {
		inputFloat[i] = float64(input[i])
	}
	for i := range reference {
		referenceFloat[i] = float64(reference[i])
	}

	// Apply echo cancellation
	output := ec.adaptiveFilter(inputFloat, referenceFloat)

	// Convert back to int16
	result := make([]int16, len(output))
	for i, val := range output {
		// Clamp to int16 range
		if val > 32767.0 {
			val = 32767.0
		}
		if val < -32768.0 {
			val = -32768.0
		}
		result[i] = int16(val)
	}

	return result
}

// adaptiveFilter performs LMS adaptive filtering for echo cancellation
func (ec *EchoCanceller) adaptiveFilter(input, reference []float64) []float64 {
	output := make([]float64, len(input))

	for i := 0; i < len(input); i++ {
		// Update reference buffer (shift and add new sample)
		ec.shiftBuffer(reference[i])

		// Estimate echo using adaptive filter
		echoEstimate := ec.estimateEcho()

		// Subtract echo estimate from input
		error := input[i] - echoEstimate
		output[i] = error

		// Update filter weights using LMS algorithm
		ec.updateWeights(error)

		// Track statistics
		echoLevel := math.Abs(echoEstimate)
		if echoLevel > 100.0 { // Threshold for echo detection
			ec.echoDetected++

			// Calculate reduction
			originalLevel := math.Abs(input[i])
			if originalLevel > 0 {
				reduction := echoLevel / originalLevel
				ec.avgEchoReduction += reduction
				ec.echoSuppressed++
			}
		}
	}

	return output
}

// shiftBuffer shifts reference buffer and adds new sample
func (ec *EchoCanceller) shiftBuffer(sample float64) {
	// Shift buffer (circular)
	for i := len(ec.referenceBuffer) - 1; i > 0; i-- {
		ec.referenceBuffer[i] = ec.referenceBuffer[i-1]
	}
	ec.referenceBuffer[0] = sample
}

// estimateEcho computes echo estimate using current filter weights
func (ec *EchoCanceller) estimateEcho() float64 {
	var echo float64

	// Convolve filter weights with reference buffer
	for i := 0; i < ec.filterLength && i < len(ec.referenceBuffer); i++ {
		echo += ec.filterWeights[i] * ec.referenceBuffer[i]
	}

	return echo
}

// updateWeights updates filter weights using LMS algorithm
func (ec *EchoCanceller) updateWeights(error float64) {
	// Normalized LMS (NLMS) for better stability
	// Calculate reference power
	var power float64
	for i := 0; i < ec.filterLength && i < len(ec.referenceBuffer); i++ {
		power += ec.referenceBuffer[i] * ec.referenceBuffer[i]
	}
	power += ec.regularization // Add regularization to avoid division by zero

	// Normalized step size
	normalizedStep := ec.stepSize / power

	// Update weights: w(n+1) = w(n) + μ * e(n) * x(n)
	for i := 0; i < ec.filterLength && i < len(ec.referenceBuffer); i++ {
		ec.filterWeights[i] += normalizedStep * error * ec.referenceBuffer[i]

		// Prevent weight overflow
		if ec.filterWeights[i] > 10.0 {
			ec.filterWeights[i] = 10.0
		}
		if ec.filterWeights[i] < -10.0 {
			ec.filterWeights[i] = -10.0
		}
	}
}

// SetStepSize adjusts adaptation speed (0.001-0.1)
func (ec *EchoCanceller) SetStepSize(stepSize float64) {
	if stepSize < 0.001 {
		stepSize = 0.001
	}
	if stepSize > 0.1 {
		stepSize = 0.1
	}
	ec.stepSize = stepSize
}

// Reset resets echo canceller state
func (ec *EchoCanceller) Reset() {
	ec.frameCount = 0
	ec.initialized = false

	// Clear filter weights
	for i := range ec.filterWeights {
		ec.filterWeights[i] = 0.0
	}

	// Clear reference buffer
	for i := range ec.referenceBuffer {
		ec.referenceBuffer[i] = 0.0
	}

	// Reset statistics
	ec.echoDetected = 0
	ec.echoSuppressed = 0
	ec.avgEchoReduction = 0.0
}

// GetStatistics returns echo cancellation statistics
func (ec *EchoCanceller) GetStatistics() EchoCancellationStatistics {
	var detectionRate float64
	var avgReduction float64

	if ec.totalFrames > 0 {
		detectionRate = float64(ec.echoDetected) / float64(ec.totalFrames) * 100.0
	}

	if ec.echoSuppressed > 0 {
		avgReduction = ec.avgEchoReduction / float64(ec.echoSuppressed) * 100.0
	}

	return EchoCancellationStatistics{
		TotalFrames:      ec.totalFrames,
		EchoDetected:     ec.echoDetected,
		EchoSuppressed:   ec.echoSuppressed,
		DetectionRate:    detectionRate,
		AvgReduction:     avgReduction,
		FilterConverged:  ec.frameCount > 100, // Consider converged after ~2 seconds
	}
}

// EchoCancellationStatistics contains EC statistics
type EchoCancellationStatistics struct {
	TotalFrames     uint64  // Total frames processed
	EchoDetected    uint64  // Frames with echo detected
	EchoSuppressed  uint64  // Frames with echo suppressed
	DetectionRate   float64 // Echo detection rate (%)
	AvgReduction    float64 // Average echo reduction (%)
	FilterConverged bool    // Filter convergence status
}

// SimpleEchoCanceller provides basic echo suppression using time delay
type SimpleEchoCanceller struct {
	delayBuffer    []int16
	delayLength    int
	suppressionGain float64
}

// NewSimpleEchoCanceller creates a simple echo canceller
func NewSimpleEchoCanceller(sampleRate, delayMs int) *SimpleEchoCanceller {
	delayLength := sampleRate * delayMs / 1000

	return &SimpleEchoCanceller{
		delayBuffer:    make([]int16, delayLength),
		delayLength:    delayLength,
		suppressionGain: 0.5, // 50% suppression
	}
}

// Process applies simple echo suppression
func (sec *SimpleEchoCanceller) Process(input, reference []int16) []int16 {
	output := make([]int16, len(input))

	for i := range input {
		// Simple approach: subtract scaled reference with delay
		delayedRef := sec.getDelayed(i)
		suppressed := float64(input[i]) - float64(delayedRef)*sec.suppressionGain

		// Clamp
		if suppressed > 32767.0 {
			suppressed = 32767.0
		}
		if suppressed < -32768.0 {
			suppressed = -32768.0
		}

		output[i] = int16(suppressed)

		// Update delay buffer
		if i < len(reference) {
			sec.updateDelay(reference[i])
		}
	}

	return output
}

// getDelayed returns delayed reference sample
func (sec *SimpleEchoCanceller) getDelayed(index int) int16 {
	if len(sec.delayBuffer) == 0 {
		return 0
	}
	return sec.delayBuffer[len(sec.delayBuffer)-1]
}

// updateDelay updates delay buffer
func (sec *SimpleEchoCanceller) updateDelay(sample int16) {
	// Shift buffer
	for i := len(sec.delayBuffer) - 1; i > 0; i-- {
		sec.delayBuffer[i] = sec.delayBuffer[i-1]
	}
	if len(sec.delayBuffer) > 0 {
		sec.delayBuffer[0] = sample
	}
}

// EchoSuppressor provides frequency-domain echo suppression
type EchoSuppressor struct {
	threshold float64
	gain      float64
}

// NewEchoSuppressor creates an echo suppressor
func NewEchoSuppressor(threshold, gain float64) *EchoSuppressor {
	return &EchoSuppressor{
		threshold: threshold,
		gain:      gain,
	}
}

// Process applies echo suppression based on energy comparison
func (es *EchoSuppressor) Process(input, reference []int16) []int16 {
	// Calculate energies
	inputEnergy := es.calculateEnergy(input)
	refEnergy := es.calculateEnergy(reference)

	// If reference is strong and input is similar, suppress echo
	if refEnergy > es.threshold && inputEnergy > es.threshold {
		// Apply suppression gain
		output := make([]int16, len(input))
		for i, sample := range input {
			suppressed := float64(sample) * es.gain
			if suppressed > 32767.0 {
				suppressed = 32767.0
			}
			if suppressed < -32768.0 {
				suppressed = -32768.0
			}
			output[i] = int16(suppressed)
		}
		return output
	}

	// No echo detected, return original
	return input
}

// calculateEnergy computes RMS energy
func (es *EchoSuppressor) calculateEnergy(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, s := range samples {
		val := float64(s)
		sum += val * val
	}

	return math.Sqrt(sum / float64(len(samples)))
}

// DoubleT alkDetector detects when both parties speak simultaneously
type DoubleTalkDetector struct {
	threshold      float64
	energyRatio    float64
	recentInput    []float64
	recentReference []float64
	historySize    int
}

// NewDoubleTalkDetector creates a double-talk detector
func NewDoubleTalkDetector() *DoubleTalkDetector {
	return &DoubleTalkDetector{
		threshold:      500.0,
		energyRatio:    0.5,  // 50% threshold for double-talk
		historySize:    10,   // 10 frames history
		recentInput:    make([]float64, 10),
		recentReference: make([]float64, 10),
	}
}

// Detect checks if double-talk is occurring
func (dtd *DoubleTalkDetector) Detect(input, reference []int16) bool {
	inputEnergy := dtd.calculateEnergy(input)
	refEnergy := dtd.calculateEnergy(reference)

	// Update history
	dtd.updateHistory(inputEnergy, refEnergy)

	// Calculate average energies
	avgInput := dtd.averageEnergy(dtd.recentInput)
	avgRef := dtd.averageEnergy(dtd.recentReference)

	// Both parties speaking if both energies are significant
	if avgInput > dtd.threshold && avgRef > dtd.threshold {
		// Check energy ratio
		ratio := avgInput / avgRef
		if ratio > dtd.energyRatio && ratio < (1.0/dtd.energyRatio) {
			return true // Double-talk detected
		}
	}

	return false // Single speaker or silence
}

// updateHistory updates energy history
func (dtd *DoubleTalkDetector) updateHistory(inputEnergy, refEnergy float64) {
	// Shift and add new values
	for i := len(dtd.recentInput) - 1; i > 0; i-- {
		dtd.recentInput[i] = dtd.recentInput[i-1]
		dtd.recentReference[i] = dtd.recentReference[i-1]
	}
	dtd.recentInput[0] = inputEnergy
	dtd.recentReference[0] = refEnergy
}

// averageEnergy calculates average of energy history
func (dtd *DoubleTalkDetector) averageEnergy(history []float64) float64 {
	if len(history) == 0 {
		return 0.0
	}

	var sum float64
	for _, e := range history {
		sum += e
	}
	return sum / float64(len(history))
}

// calculateEnergy computes RMS energy
func (dtd *DoubleTalkDetector) calculateEnergy(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, s := range samples {
		val := float64(s)
		sum += val * val
	}

	return math.Sqrt(sum / float64(len(samples)))
}
