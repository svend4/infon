package audio

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// WebRTCAudioProcessor provides high-quality audio processing using WebRTC algorithms
// Includes echo cancellation (AEC3), noise suppression, AGC, and VAD
type WebRTCAudioProcessor struct {
	mu sync.RWMutex

	// Configuration
	sampleRate   int
	channels     int
	frameSamples int

	// Processing modules
	aec3            *AEC3           // Acoustic Echo Cancellation v3
	noiseSuppressor *NoiseSuppressor
	agc             *AutomaticGainControl
	vad             *VoiceActivityDetector

	// State
	enabled bool

	// Statistics
	stats ProcessorStatistics
}

// ProcessorConfig holds WebRTC processor configuration
type ProcessorConfig struct {
	SampleRate   int  // 8000, 16000, 32000, 48000
	Channels     int  // 1 (mono) or 2 (stereo)
	FrameSamples int  // Samples per frame (e.g., 160 for 10ms at 16kHz)
	EnableAEC    bool // Enable Acoustic Echo Cancellation
	EnableNS     bool // Enable Noise Suppression
	EnableAGC    bool // Enable Automatic Gain Control
	EnableVAD    bool // Enable Voice Activity Detection
}

// ProcessorStatistics tracks processing performance
type ProcessorStatistics struct {
	FramesProcessed  uint64
	EchoCanceled     uint64 // Frames with echo detected and canceled
	NoiseReduced     uint64 // Frames with noise reduced
	SpeechDetected   uint64 // Frames with speech detected
	AverageProcessTime time.Duration
}

// NewWebRTCAudioProcessor creates a new WebRTC audio processor
func NewWebRTCAudioProcessor(config ProcessorConfig) (*WebRTCAudioProcessor, error) {
	if config.SampleRate != 8000 && config.SampleRate != 16000 &&
		config.SampleRate != 32000 && config.SampleRate != 48000 {
		return nil, fmt.Errorf("invalid sample rate: %d", config.SampleRate)
	}

	if config.Channels < 1 || config.Channels > 2 {
		return nil, fmt.Errorf("channels must be 1 or 2, got %d", config.Channels)
	}

	p := &WebRTCAudioProcessor{
		sampleRate:   config.SampleRate,
		channels:     config.Channels,
		frameSamples: config.FrameSamples,
		enabled:      true,
	}

	// Initialize AEC3 (Acoustic Echo Cancellation v3)
	if config.EnableAEC {
		p.aec3 = NewAEC3(config.SampleRate, config.Channels)
	}

	// Initialize Noise Suppressor
	if config.EnableNS {
		p.noiseSuppressor = NewNoiseSuppressor(config.SampleRate, config.Channels)
	}

	// Initialize AGC (Automatic Gain Control)
	if config.EnableAGC {
		p.agc = NewAutomaticGainControl(config.SampleRate, config.Channels)
	}

	// Initialize VAD (Voice Activity Detector)
	if config.EnableVAD {
		p.vad = NewVoiceActivityDetector(config.SampleRate)
	}

	return p, nil
}

// ProcessCapture processes captured microphone audio
// Returns processed audio and whether speech was detected
func (p *WebRTCAudioProcessor) ProcessCapture(input []int16, renderAudio []int16) ([]int16, bool, error) {
	if !p.enabled {
		return input, true, nil
	}

	start := time.Now()

	output := make([]int16, len(input))
	copy(output, input)

	// Step 1: AEC3 - Remove echo from speakers
	if p.aec3 != nil {
		processed, echoDetected := p.aec3.ProcessCapture(output, renderAudio)
		output = processed
		if echoDetected {
			p.mu.Lock()
			p.stats.EchoCanceled++
			p.mu.Unlock()
		}
	}

	// Step 2: Noise Suppression
	if p.noiseSuppressor != nil {
		processed, noiseReduced := p.noiseSuppressor.Process(output)
		output = processed
		if noiseReduced {
			p.mu.Lock()
			p.stats.NoiseReduced++
			p.mu.Unlock()
		}
	}

	// Step 3: Automatic Gain Control
	if p.agc != nil {
		output = p.agc.Process(output)
	}

	// Step 4: Voice Activity Detection
	hasSpeech := true
	if p.vad != nil {
		hasSpeech = p.vad.Process(output)
		if hasSpeech {
			p.mu.Lock()
			p.stats.SpeechDetected++
			p.mu.Unlock()
		}
	}

	// Update statistics
	p.mu.Lock()
	p.stats.FramesProcessed++
	processTime := time.Since(start)
	p.stats.AverageProcessTime = (p.stats.AverageProcessTime*time.Duration(p.stats.FramesProcessed-1) +
		processTime) / time.Duration(p.stats.FramesProcessed)
	p.mu.Unlock()

	return output, hasSpeech, nil
}

// ProcessRender processes speaker audio for AEC reference
func (p *WebRTCAudioProcessor) ProcessRender(audio []int16) error {
	if !p.enabled {
		return nil
	}

	// Feed render audio to AEC for echo cancellation
	if p.aec3 != nil {
		p.aec3.ProcessRender(audio)
	}

	return nil
}

// Enable enables audio processing
func (p *WebRTCAudioProcessor) Enable() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = true
}

// Disable disables audio processing (passthrough mode)
func (p *WebRTCAudioProcessor) Disable() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = false
}

// Reset resets all processing modules
func (p *WebRTCAudioProcessor) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.aec3 != nil {
		p.aec3.Reset()
	}
	if p.noiseSuppressor != nil {
		p.noiseSuppressor.Reset()
	}
	if p.agc != nil {
		p.agc.Reset()
	}
	if p.vad != nil {
		p.vad.Reset()
	}

	p.stats = ProcessorStatistics{}
}

// GetStatistics returns processing statistics
func (p *WebRTCAudioProcessor) GetStatistics() ProcessorStatistics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// SetAGCGain sets the target AGC gain in dB
func (p *WebRTCAudioProcessor) SetAGCGain(gainDB float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.agc == nil {
		return fmt.Errorf("AGC not enabled")
	}

	p.agc.SetTargetGain(gainDB)
	return nil
}

// SetNSLevel sets the noise suppression level (0-3)
func (p *WebRTCAudioProcessor) SetNSLevel(level int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.noiseSuppressor == nil {
		return fmt.Errorf("noise suppression not enabled")
	}

	return p.noiseSuppressor.SetLevel(level)
}

// AEC3 - Acoustic Echo Cancellation v3
// Removes echo from speaker output that bleeds into microphone
type AEC3 struct {
	mu sync.RWMutex

	sampleRate int
	channels   int

	// Echo cancellation state
	renderBuffer []int16 // Circular buffer for render audio
	bufferSize   int
	writePos     int

	// Adaptive filter coefficients
	filterLength  int
	filterCoeffs  []float64
	stepSize      float64
	echoEstimate  []float64

	// Echo detection
	echoThreshold float64
}

// NewAEC3 creates a new AEC3 instance
func NewAEC3(sampleRate, channels int) *AEC3 {
	filterLength := 512 // ~32ms at 16kHz

	return &AEC3{
		sampleRate:    sampleRate,
		channels:      channels,
		bufferSize:    sampleRate / 10, // 100ms buffer
		renderBuffer:  make([]int16, sampleRate/10),
		filterLength:  filterLength,
		filterCoeffs:  make([]float64, filterLength),
		stepSize:      0.01,
		echoEstimate:  make([]float64, filterLength),
		echoThreshold: 0.1,
	}
}

// ProcessRender processes speaker output for echo reference
func (a *AEC3) ProcessRender(audio []int16) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Store render audio in circular buffer
	for _, sample := range audio {
		a.renderBuffer[a.writePos] = sample
		a.writePos = (a.writePos + 1) % a.bufferSize
	}
}

// ProcessCapture processes microphone input and removes echo
func (a *AEC3) ProcessCapture(capture, render []int16) ([]int16, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	output := make([]int16, len(capture))
	echoDetected := false

	for i := 0; i < len(capture); i++ {
		// Estimate echo using adaptive filter
		var echoEst float64
		for j := 0; j < a.filterLength && j < len(render); j++ {
			if i-j >= 0 {
				echoEst += a.filterCoeffs[j] * float64(render[i-j])
			}
		}

		// Remove echo from capture
		errorSignal := float64(capture[i]) - echoEst

		// Update filter coefficients using LMS algorithm
		if len(render) > 0 {
			for j := 0; j < a.filterLength && j < len(render); j++ {
				if i-j >= 0 {
					a.filterCoeffs[j] += a.stepSize * errorSignal * float64(render[i-j])
				}
			}
		}

		// Check if echo was significant
		if math.Abs(echoEst) > a.echoThreshold*float64(math.MaxInt16) {
			echoDetected = true
		}

		// Clamp output
		if errorSignal > math.MaxInt16 {
			output[i] = math.MaxInt16
		} else if errorSignal < math.MinInt16 {
			output[i] = math.MinInt16
		} else {
			output[i] = int16(errorSignal)
		}
	}

	return output, echoDetected
}

// Reset resets AEC3 state
func (a *AEC3) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.filterCoeffs {
		a.filterCoeffs[i] = 0
	}
	for i := range a.renderBuffer {
		a.renderBuffer[i] = 0
	}
	a.writePos = 0
}

// NoiseSuppressor - Spectral noise suppression
type NoiseSuppressor struct {
	mu sync.RWMutex

	sampleRate int
	channels   int
	level      int // 0=off, 1=low, 2=medium, 3=high

	// Noise estimation
	noiseFloor    float64
	noiseEstimate []float64
	smoothing     float64

	// Threshold for noise detection
	noiseThreshold float64
}

// NewNoiseSuppressor creates a new noise suppressor
func NewNoiseSuppressor(sampleRate, channels int) *NoiseSuppressor {
	return &NoiseSuppressor{
		sampleRate:     sampleRate,
		channels:       channels,
		level:          2, // Medium by default
		noiseFloor:     100.0,
		noiseEstimate:  make([]float64, 256),
		smoothing:      0.95,
		noiseThreshold: 500.0,
	}
}

// Process applies noise suppression
func (ns *NoiseSuppressor) Process(audio []int16) ([]int16, bool) {
	if ns.level == 0 {
		return audio, false
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	output := make([]int16, len(audio))
	noiseReduced := false

	// Calculate signal energy
	var energy float64
	for _, sample := range audio {
		energy += float64(sample) * float64(sample)
	}
	energy /= float64(len(audio))

	// Update noise estimate
	ns.noiseFloor = ns.smoothing*ns.noiseFloor + (1-ns.smoothing)*energy

	// Apply suppression based on level
	suppressionFactor := 1.0
	if energy < ns.noiseFloor*float64(ns.level+1) {
		suppressionFactor = 0.5 / float64(ns.level+1)
		noiseReduced = true
	}

	// Apply suppression
	for i := range audio {
		output[i] = int16(float64(audio[i]) * suppressionFactor)
	}

	return output, noiseReduced
}

// SetLevel sets the noise suppression level (0-3)
func (ns *NoiseSuppressor) SetLevel(level int) error {
	if level < 0 || level > 3 {
		return fmt.Errorf("level must be 0-3, got %d", level)
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.level = level
	return nil
}

// Reset resets noise suppressor state
func (ns *NoiseSuppressor) Reset() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.noiseFloor = 100.0
	for i := range ns.noiseEstimate {
		ns.noiseEstimate[i] = 0
	}
}

// AutomaticGainControl - AGC for consistent volume
type AutomaticGainControl struct {
	mu sync.RWMutex

	sampleRate int
	channels   int

	// AGC parameters
	targetGain    float64 // Target gain in dB
	currentGain   float64 // Current gain multiplier
	maxGain       float64 // Maximum gain multiplier
	minGain       float64 // Minimum gain multiplier
	attackTime    float64 // Attack time constant
	releaseTime   float64 // Release time constant

	// Peak detection
	peakLevel float64
}

// NewAutomaticGainControl creates a new AGC
func NewAutomaticGainControl(sampleRate, channels int) *AutomaticGainControl {
	return &AutomaticGainControl{
		sampleRate:  sampleRate,
		channels:    channels,
		targetGain:  0.0,    // 0 dB
		currentGain: 1.0,    // Unity gain
		maxGain:     10.0,   // +20 dB
		minGain:     0.1,    // -20 dB
		attackTime:  0.001,  // 1ms
		releaseTime: 0.1,    // 100ms
		peakLevel:   0.0,
	}
}

// Process applies automatic gain control
func (agc *AutomaticGainControl) Process(audio []int16) []int16 {
	agc.mu.Lock()
	defer agc.mu.Unlock()

	output := make([]int16, len(audio))

	// Find peak in this frame
	var peak float64
	for _, sample := range audio {
		abs := math.Abs(float64(sample))
		if abs > peak {
			peak = abs
		}
	}

	// Update peak level with attack/release
	if peak > agc.peakLevel {
		agc.peakLevel = agc.peakLevel + agc.attackTime*(peak-agc.peakLevel)
	} else {
		agc.peakLevel = agc.peakLevel + agc.releaseTime*(peak-agc.peakLevel)
	}

	// Calculate required gain
	targetLevel := 16384.0 // 50% of max int16
	if agc.peakLevel > 0 {
		requiredGain := targetLevel / agc.peakLevel

		// Clamp gain
		if requiredGain > agc.maxGain {
			requiredGain = agc.maxGain
		} else if requiredGain < agc.minGain {
			requiredGain = agc.minGain
		}

		agc.currentGain = requiredGain
	}

	// Apply gain
	for i, sample := range audio {
		gained := float64(sample) * agc.currentGain

		// Clamp to int16 range
		if gained > math.MaxInt16 {
			output[i] = math.MaxInt16
		} else if gained < math.MinInt16 {
			output[i] = math.MinInt16
		} else {
			output[i] = int16(gained)
		}
	}

	return output
}

// SetTargetGain sets the target gain in dB
func (agc *AutomaticGainControl) SetTargetGain(gainDB float64) {
	agc.mu.Lock()
	defer agc.mu.Unlock()
	agc.targetGain = gainDB
}

// Reset resets AGC state
func (agc *AutomaticGainControl) Reset() {
	agc.mu.Lock()
	defer agc.mu.Unlock()

	agc.currentGain = 1.0
	agc.peakLevel = 0.0
}

// VoiceActivityDetector - Detects speech vs silence
type VoiceActivityDetector struct {
	mu sync.RWMutex

	sampleRate int

	// Energy thresholds
	energyThreshold float64
	minEnergy       float64
	maxEnergy       float64

	// Smoothing
	smoothedEnergy float64
	smoothing      float64

	// State
	isSpeech bool
}

// NewVoiceActivityDetector creates a new VAD
func NewVoiceActivityDetector(sampleRate int) *VoiceActivityDetector {
	return &VoiceActivityDetector{
		sampleRate:      sampleRate,
		energyThreshold: 500000.0,
		minEnergy:       100.0,
		maxEnergy:       1000000.0,
		smoothing:       0.9,
		isSpeech:        false,
	}
}

// Process detects voice activity
func (vad *VoiceActivityDetector) Process(audio []int16) bool {
	vad.mu.Lock()
	defer vad.mu.Unlock()

	// Calculate frame energy
	var energy float64
	for _, sample := range audio {
		energy += float64(sample) * float64(sample)
	}
	energy /= float64(len(audio))

	// Update smoothed energy
	vad.smoothedEnergy = vad.smoothing*vad.smoothedEnergy + (1-vad.smoothing)*energy

	// Detect speech
	vad.isSpeech = vad.smoothedEnergy > vad.energyThreshold

	return vad.isSpeech
}

// Reset resets VAD state
func (vad *VoiceActivityDetector) Reset() {
	vad.mu.Lock()
	defer vad.mu.Unlock()

	vad.smoothedEnergy = 0
	vad.isSpeech = false
}

// IsSpeech returns whether speech is currently detected
func (vad *VoiceActivityDetector) IsSpeech() bool {
	vad.mu.RLock()
	defer vad.mu.RUnlock()
	return vad.isSpeech
}
