package codec

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// HardwareAccelerator provides interface to hardware-accelerated video encoding/decoding
// Supports: NVIDIA NVENC/NVDEC, Intel QSV, AMD VCE/VCN, Apple VideoToolbox, VA-API
type HardwareAccelerator struct {
	mu sync.RWMutex

	// Configuration
	codecType     CodecType
	acceleratorType AcceleratorType
	deviceID      int

	// Capabilities
	maxResolution Resolution
	supportedCodecs []CodecType
	maxInstances  int

	// State
	initialized bool
	instances   int

	// Statistics
	stats HWAccelStatistics
}

// AcceleratorType represents the hardware acceleration platform
type AcceleratorType string

const (
	AccelNVIDIA        AcceleratorType = "nvidia"    // NVIDIA NVENC/NVDEC
	AccelIntelQSV      AcceleratorType = "intel_qsv" // Intel Quick Sync Video
	AccelAMD           AcceleratorType = "amd"       // AMD VCE/VCN
	AccelAppleVT       AcceleratorType = "apple_vt"  // Apple VideoToolbox
	AccelVAAPI         AcceleratorType = "vaapi"     // VA-API (Linux)
	AccelVideoCore     AcceleratorType = "videocore" // Raspberry Pi
	AccelNone          AcceleratorType = "none"      // Software fallback
)

// Resolution represents video resolution
type Resolution struct {
	Width  int
	Height int
}

// HWAccelStatistics tracks hardware acceleration performance
type HWAccelStatistics struct {
	FramesEncoded     uint64
	FramesDecoded     uint64
	AverageEncodeTime time.Duration
	AverageDecodeTime time.Duration
	GPUUsage          float64 // Percentage
	VRAMUsed          uint64  // Bytes
	Errors            uint64
}

// AcceleratorConfig holds hardware accelerator configuration
type AcceleratorConfig struct {
	Type      AcceleratorType
	CodecType CodecType
	DeviceID  int  // GPU device ID (0 = default)
	Fallback  bool // Fall back to software if hardware unavailable
}

// NewHardwareAccelerator creates a new hardware accelerator
func NewHardwareAccelerator(config AcceleratorConfig) (*HardwareAccelerator, error) {
	// Auto-detect accelerator if not specified
	if config.Type == "" {
		config.Type = DetectAccelerator()
	}

	accel := &HardwareAccelerator{
		codecType:       config.CodecType,
		acceleratorType: config.Type,
		deviceID:        config.DeviceID,
	}

	// Initialize based on type
	err := accel.initialize()
	if err != nil {
		if config.Fallback {
			// Fall back to software
			accel.acceleratorType = AccelNone
			return accel, nil
		}
		return nil, fmt.Errorf("failed to initialize accelerator: %w", err)
	}

	return accel, nil
}

// initialize initializes the hardware accelerator
func (ha *HardwareAccelerator) initialize() error {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	switch ha.acceleratorType {
	case AccelNVIDIA:
		return ha.initNVIDIA()
	case AccelIntelQSV:
		return ha.initIntelQSV()
	case AccelAMD:
		return ha.initAMD()
	case AccelAppleVT:
		return ha.initAppleVT()
	case AccelVAAPI:
		return ha.initVAAPI()
	case AccelVideoCore:
		return ha.initVideoCore()
	case AccelNone:
		// Software fallback - always available
		ha.maxResolution = Resolution{Width: 7680, Height: 4320} // 8K
		ha.supportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP8, CodecTypeVP9}
		ha.maxInstances = 999 // No practical limit for software
		ha.initialized = true
		return nil
	default:
		return fmt.Errorf("unsupported accelerator type: %s", ha.acceleratorType)
	}
}

// NVIDIA NVENC/NVDEC initialization
func (ha *HardwareAccelerator) initNVIDIA() error {
	// Check if NVIDIA GPU is available
	if !isNVIDIAAvailable() {
		return fmt.Errorf("NVIDIA GPU not found")
	}

	// Set capabilities
	ha.maxResolution = Resolution{Width: 8192, Height: 8192}
	ha.supportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP9}
	ha.maxInstances = 32 // NVENC supports up to 32 concurrent sessions

	ha.initialized = true
	return nil
}

// Intel Quick Sync Video initialization
func (ha *HardwareAccelerator) initIntelQSV() error {
	if !isIntelQSVAvailable() {
		return fmt.Errorf("Intel QSV not available")
	}

	ha.maxResolution = Resolution{Width: 4096, Height: 4096}
	ha.supportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP9}
	ha.maxInstances = 16

	ha.initialized = true
	return nil
}

// AMD VCE/VCN initialization
func (ha *HardwareAccelerator) initAMD() error {
	if !isAMDAvailable() {
		return fmt.Errorf("AMD GPU not found")
	}

	ha.maxResolution = Resolution{Width: 4096, Height: 4096}
	ha.supportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265}
	ha.maxInstances = 16

	ha.initialized = true
	return nil
}

// Apple VideoToolbox initialization
func (ha *HardwareAccelerator) initAppleVT() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("VideoToolbox only available on macOS/iOS")
	}

	ha.maxResolution = Resolution{Width: 4096, Height: 4096}
	ha.supportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265}
	ha.maxInstances = 8

	ha.initialized = true
	return nil
}

// VA-API initialization (Linux)
func (ha *HardwareAccelerator) initVAAPI() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("VA-API only available on Linux")
	}

	if !isVAAPIAvailable() {
		return fmt.Errorf("VA-API not available")
	}

	ha.maxResolution = Resolution{Width: 4096, Height: 4096}
	ha.supportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP8, CodecTypeVP9}
	ha.maxInstances = 16

	ha.initialized = true
	return nil
}

// Raspberry Pi VideoCore initialization
func (ha *HardwareAccelerator) initVideoCore() error {
	if !isVideoCoreAvailable() {
		return fmt.Errorf("VideoCore not available")
	}

	ha.maxResolution = Resolution{Width: 1920, Height: 1080}
	ha.supportedCodecs = []CodecType{CodecTypeH264}
	ha.maxInstances = 2

	ha.initialized = true
	return nil
}

// EncodeFrame encodes a video frame using hardware acceleration
func (ha *HardwareAccelerator) EncodeFrame(input []byte, width, height int) ([]byte, error) {
	if !ha.initialized {
		return nil, fmt.Errorf("accelerator not initialized")
	}

	start := time.Now()

	ha.mu.Lock()
	if ha.instances >= ha.maxInstances {
		ha.mu.Unlock()
		return nil, fmt.Errorf("max concurrent instances reached: %d", ha.maxInstances)
	}
	ha.instances++
	ha.mu.Unlock()

	defer func() {
		ha.mu.Lock()
		ha.instances--
		ha.mu.Unlock()
	}()

	// Check resolution
	if width > ha.maxResolution.Width || height > ha.maxResolution.Height {
		return nil, fmt.Errorf("resolution %dx%d exceeds maximum %dx%d",
			width, height, ha.maxResolution.Width, ha.maxResolution.Height)
	}

	var output []byte
	var err error

	// Encode based on accelerator type
	switch ha.acceleratorType {
	case AccelNVIDIA:
		output, err = ha.encodeNVIDIA(input, width, height)
	case AccelIntelQSV:
		output, err = ha.encodeIntelQSV(input, width, height)
	case AccelAMD:
		output, err = ha.encodeAMD(input, width, height)
	case AccelAppleVT:
		output, err = ha.encodeAppleVT(input, width, height)
	case AccelVAAPI:
		output, err = ha.encodeVAAPI(input, width, height)
	case AccelVideoCore:
		output, err = ha.encodeVideoCore(input, width, height)
	case AccelNone:
		// Software encode (simplified)
		output = append([]byte{0x00, 0x00, 0x01}, input...)
	default:
		return nil, fmt.Errorf("unsupported accelerator type: %s", ha.acceleratorType)
	}

	if err != nil {
		ha.mu.Lock()
		ha.stats.Errors++
		ha.mu.Unlock()
		return nil, err
	}

	// Update statistics
	encodeTime := time.Since(start)
	ha.mu.Lock()
	ha.stats.FramesEncoded++
	ha.stats.AverageEncodeTime = (ha.stats.AverageEncodeTime*time.Duration(ha.stats.FramesEncoded-1) +
		encodeTime) / time.Duration(ha.stats.FramesEncoded)
	ha.mu.Unlock()

	return output, nil
}

// DecodeFrame decodes a video frame using hardware acceleration
func (ha *HardwareAccelerator) DecodeFrame(input []byte) ([]byte, int, int, error) {
	if !ha.initialized {
		return nil, 0, 0, fmt.Errorf("accelerator not initialized")
	}

	start := time.Now()

	ha.mu.Lock()
	if ha.instances >= ha.maxInstances {
		ha.mu.Unlock()
		return nil, 0, 0, fmt.Errorf("max concurrent instances reached")
	}
	ha.instances++
	ha.mu.Unlock()

	defer func() {
		ha.mu.Lock()
		ha.instances--
		ha.mu.Unlock()
	}()

	var output []byte
	var width, height int
	var err error

	// Decode based on accelerator type
	switch ha.acceleratorType {
	case AccelNVIDIA:
		output, width, height, err = ha.decodeNVIDIA(input)
	case AccelIntelQSV:
		output, width, height, err = ha.decodeIntelQSV(input)
	case AccelAMD:
		output, width, height, err = ha.decodeAMD(input)
	case AccelAppleVT:
		output, width, height, err = ha.decodeAppleVT(input)
	case AccelVAAPI:
		output, width, height, err = ha.decodeVAAPI(input)
	case AccelVideoCore:
		output, width, height, err = ha.decodeVideoCore(input)
	case AccelNone:
		// Software decode (simplified)
		output = input[3:] // Remove start code
		width = 640
		height = 480
	default:
		return nil, 0, 0, fmt.Errorf("unsupported accelerator type")
	}

	if err != nil {
		ha.mu.Lock()
		ha.stats.Errors++
		ha.mu.Unlock()
		return nil, 0, 0, err
	}

	// Update statistics
	decodeTime := time.Since(start)
	ha.mu.Lock()
	ha.stats.FramesDecoded++
	ha.stats.AverageDecodeTime = (ha.stats.AverageDecodeTime*time.Duration(ha.stats.FramesDecoded-1) +
		decodeTime) / time.Duration(ha.stats.FramesDecoded)
	ha.mu.Unlock()

	return output, width, height, nil
}

// Platform-specific encode implementations (simplified placeholders)

func (ha *HardwareAccelerator) encodeNVIDIA(input []byte, width, height int) ([]byte, error) {
	// In production: Use NVIDIA Video Codec SDK
	// Simulate GPU encoding
	output := make([]byte, len(input)/10) // 10:1 compression
	copy(output, []byte{0x00, 0x00, 0x01, 0x67}) // H.264 SPS
	return output, nil
}

func (ha *HardwareAccelerator) encodeIntelQSV(input []byte, width, height int) ([]byte, error) {
	// In production: Use Intel Media SDK
	output := make([]byte, len(input)/10)
	copy(output, []byte{0x00, 0x00, 0x01, 0x67})
	return output, nil
}

func (ha *HardwareAccelerator) encodeAMD(input []byte, width, height int) ([]byte, error) {
	// In production: Use AMD Advanced Media Framework (AMF)
	output := make([]byte, len(input)/10)
	copy(output, []byte{0x00, 0x00, 0x01, 0x67})
	return output, nil
}

func (ha *HardwareAccelerator) encodeAppleVT(input []byte, width, height int) ([]byte, error) {
	// In production: Use VideoToolbox framework
	output := make([]byte, len(input)/10)
	copy(output, []byte{0x00, 0x00, 0x01, 0x67})
	return output, nil
}

func (ha *HardwareAccelerator) encodeVAAPI(input []byte, width, height int) ([]byte, error) {
	// In production: Use VA-API
	output := make([]byte, len(input)/10)
	copy(output, []byte{0x00, 0x00, 0x01, 0x67})
	return output, nil
}

func (ha *HardwareAccelerator) encodeVideoCore(input []byte, width, height int) ([]byte, error) {
	// In production: Use OpenMAX or MMAL
	output := make([]byte, len(input)/10)
	copy(output, []byte{0x00, 0x00, 0x01, 0x67})
	return output, nil
}

// Platform-specific decode implementations (simplified placeholders)

func (ha *HardwareAccelerator) decodeNVIDIA(input []byte) ([]byte, int, int, error) {
	output := make([]byte, len(input)*10) // 10x expansion
	return output, 1920, 1080, nil
}

func (ha *HardwareAccelerator) decodeIntelQSV(input []byte) ([]byte, int, int, error) {
	output := make([]byte, len(input)*10)
	return output, 1920, 1080, nil
}

func (ha *HardwareAccelerator) decodeAMD(input []byte) ([]byte, int, int, error) {
	output := make([]byte, len(input)*10)
	return output, 1920, 1080, nil
}

func (ha *HardwareAccelerator) decodeAppleVT(input []byte) ([]byte, int, int, error) {
	output := make([]byte, len(input)*10)
	return output, 1920, 1080, nil
}

func (ha *HardwareAccelerator) decodeVAAPI(input []byte) ([]byte, int, int, error) {
	output := make([]byte, len(input)*10)
	return output, 1920, 1080, nil
}

func (ha *HardwareAccelerator) decodeVideoCore(input []byte) ([]byte, int, int, error) {
	output := make([]byte, len(input)*10)
	return output, 1920, 1080, nil
}

// GetStatistics returns acceleration statistics
func (ha *HardwareAccelerator) GetStatistics() HWAccelStatistics {
	ha.mu.RLock()
	defer ha.mu.RUnlock()
	return ha.stats
}

// Reset resets statistics
func (ha *HardwareAccelerator) Reset() {
	ha.mu.Lock()
	defer ha.mu.Unlock()
	ha.stats = HWAccelStatistics{}
}

// Close releases hardware resources
func (ha *HardwareAccelerator) Close() error {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	ha.initialized = false
	return nil
}

// GetType returns the accelerator type
func (ha *HardwareAccelerator) GetType() AcceleratorType {
	ha.mu.RLock()
	defer ha.mu.RUnlock()
	return ha.acceleratorType
}

// IsAvailable checks if accelerator is available and initialized
func (ha *HardwareAccelerator) IsAvailable() bool {
	ha.mu.RLock()
	defer ha.mu.RUnlock()
	return ha.initialized
}

// GetMaxResolution returns maximum supported resolution
func (ha *HardwareAccelerator) GetMaxResolution() Resolution {
	ha.mu.RLock()
	defer ha.mu.RUnlock()
	return ha.maxResolution
}

// GetSupportedCodecs returns list of supported codecs
func (ha *HardwareAccelerator) GetSupportedCodecs() []CodecType {
	ha.mu.RLock()
	defer ha.mu.RUnlock()
	return ha.supportedCodecs
}

// Hardware Detection Functions

// DetectAccelerator auto-detects available hardware acceleration
func DetectAccelerator() AcceleratorType {
	switch runtime.GOOS {
	case "windows":
		if isNVIDIAAvailable() {
			return AccelNVIDIA
		}
		if isIntelQSVAvailable() {
			return AccelIntelQSV
		}
		if isAMDAvailable() {
			return AccelAMD
		}
	case "linux":
		if isNVIDIAAvailable() {
			return AccelNVIDIA
		}
		if isVAAPIAvailable() {
			return AccelVAAPI
		}
		if isVideoCoreAvailable() {
			return AccelVideoCore
		}
	case "darwin":
		return AccelAppleVT
	}

	return AccelNone
}

func isNVIDIAAvailable() bool {
	// In production: Check for nvidia-smi or CUDA runtime
	// Simplified check
	return false // Would detect NVIDIA GPU
}

func isIntelQSVAvailable() bool {
	// In production: Check for Intel GPU via device enumeration
	return false
}

func isAMDAvailable() bool {
	// In production: Check for AMD GPU
	return false
}

func isVAAPIAvailable() bool {
	// In production: Check for /dev/dri/renderD* devices
	return false
}

func isVideoCoreAvailable() bool {
	// In production: Check for Raspberry Pi hardware
	return false
}

// ListAvailableAccelerators returns all available accelerators
func ListAvailableAccelerators() []AcceleratorType {
	accelerators := []AcceleratorType{}

	if isNVIDIAAvailable() {
		accelerators = append(accelerators, AccelNVIDIA)
	}
	if isIntelQSVAvailable() {
		accelerators = append(accelerators, AccelIntelQSV)
	}
	if isAMDAvailable() {
		accelerators = append(accelerators, AccelAMD)
	}
	if runtime.GOOS == "darwin" {
		accelerators = append(accelerators, AccelAppleVT)
	}
	if isVAAPIAvailable() {
		accelerators = append(accelerators, AccelVAAPI)
	}
	if isVideoCoreAvailable() {
		accelerators = append(accelerators, AccelVideoCore)
	}

	// Software fallback always available
	accelerators = append(accelerators, AccelNone)

	return accelerators
}

// AcceleratorInfo provides detailed accelerator information
type AcceleratorInfo struct {
	Type            AcceleratorType
	Name            string
	MaxResolution   Resolution
	SupportedCodecs []CodecType
	MaxInstances    int
	DeviceCount     int
}

// GetAcceleratorInfo returns detailed information about an accelerator
func GetAcceleratorInfo(accelType AcceleratorType) (*AcceleratorInfo, error) {
	info := &AcceleratorInfo{
		Type: accelType,
	}

	switch accelType {
	case AccelNVIDIA:
		info.Name = "NVIDIA NVENC/NVDEC"
		info.MaxResolution = Resolution{8192, 8192}
		info.SupportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP9}
		info.MaxInstances = 32
		info.DeviceCount = getNVIDIADeviceCount()

	case AccelIntelQSV:
		info.Name = "Intel Quick Sync Video"
		info.MaxResolution = Resolution{4096, 4096}
		info.SupportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP9}
		info.MaxInstances = 16
		info.DeviceCount = 1

	case AccelAMD:
		info.Name = "AMD VCE/VCN"
		info.MaxResolution = Resolution{4096, 4096}
		info.SupportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265}
		info.MaxInstances = 16
		info.DeviceCount = getAMDDeviceCount()

	case AccelAppleVT:
		info.Name = "Apple VideoToolbox"
		info.MaxResolution = Resolution{4096, 4096}
		info.SupportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265}
		info.MaxInstances = 8
		info.DeviceCount = 1

	case AccelVAAPI:
		info.Name = "VA-API"
		info.MaxResolution = Resolution{4096, 4096}
		info.SupportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP8, CodecTypeVP9}
		info.MaxInstances = 16
		info.DeviceCount = getVAAPIDeviceCount()

	case AccelVideoCore:
		info.Name = "Raspberry Pi VideoCore"
		info.MaxResolution = Resolution{1920, 1080}
		info.SupportedCodecs = []CodecType{CodecTypeH264}
		info.MaxInstances = 2
		info.DeviceCount = 1

	case AccelNone:
		info.Name = "Software (No Hardware Acceleration)"
		info.MaxResolution = Resolution{7680, 4320}
		info.SupportedCodecs = []CodecType{CodecTypeH264, CodecTypeH265, CodecTypeVP8, CodecTypeVP9}
		info.MaxInstances = 999
		info.DeviceCount = 1

	default:
		return nil, fmt.Errorf("unknown accelerator type: %s", accelType)
	}

	return info, nil
}

func getNVIDIADeviceCount() int {
	// In production: Query via nvidia-smi or CUDA
	return 0
}

func getAMDDeviceCount() int {
	// In production: Query AMD devices
	return 0
}

func getVAAPIDeviceCount() int {
	// In production: Count /dev/dri/renderD* devices
	return 0
}
