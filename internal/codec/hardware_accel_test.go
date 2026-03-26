package codec

import (
	"runtime"
	"testing"
)

func TestNewHardwareAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone, // Always available
		CodecType: CodecTypeH264,
		DeviceID:  0,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create accelerator: %v", err)
	}

	if accel == nil {
		t.Fatal("Accelerator is nil")
	}

	if !accel.IsAvailable() {
		t.Error("Accelerator should be available")
	}
}

func TestNewHardwareAcceleratorAutoDetect(t *testing.T) {
	config := AcceleratorConfig{
		Type:      "", // Auto-detect
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create accelerator: %v", err)
	}

	// Should fall back to software if no hardware available
	accelType := accel.GetType()
	if accelType == "" {
		t.Error("Accelerator type should not be empty")
	}

	t.Logf("Detected accelerator: %s", accelType)
}

func TestHardwareAcceleratorEncodeFrame(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	// Create test frame (YUV420)
	width := 640
	height := 480
	input := make([]byte, width*height*3/2)
	for i := range input {
		input[i] = byte(i % 256)
	}

	output, err := accel.EncodeFrame(input, width, height)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("Encoded output is empty")
	}

	t.Logf("Encoded %d bytes to %d bytes", len(input), len(output))

	// Verify statistics
	stats := accel.GetStatistics()
	if stats.FramesEncoded != 1 {
		t.Errorf("FramesEncoded = %d, expected 1", stats.FramesEncoded)
	}

	if stats.AverageEncodeTime == 0 {
		t.Error("Average encode time should not be zero")
	}
}

func TestHardwareAcceleratorDecodeFrame(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	// Create test encoded frame
	input := []byte{0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1F, 0x8C}

	output, width, height, err := accel.DecodeFrame(input)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("Decoded output is empty")
	}

	if width <= 0 || height <= 0 {
		t.Errorf("Invalid resolution: %dx%d", width, height)
	}

	t.Logf("Decoded to %dx%d, %d bytes", width, height, len(output))

	// Verify statistics
	stats := accel.GetStatistics()
	if stats.FramesDecoded != 1 {
		t.Errorf("FramesDecoded = %d, expected 1", stats.FramesDecoded)
	}
}

func TestHardwareAcceleratorMaxResolution(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	// Try to encode frame exceeding maximum resolution
	width := 9000
	height := 9000
	input := make([]byte, 1024) // Small buffer for test

	_, err := accel.EncodeFrame(input, width, height)
	if err == nil {
		t.Error("EncodeFrame should fail for oversized resolution")
	}
}

func TestHardwareAcceleratorStatistics(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	// Initial statistics
	stats := accel.GetStatistics()
	if stats.FramesEncoded != 0 {
		t.Errorf("Initial FramesEncoded = %d, expected 0", stats.FramesEncoded)
	}

	// Encode some frames
	input := make([]byte, 640*480*3/2)
	for i := 0; i < 10; i++ {
		accel.EncodeFrame(input, 640, 480)
	}

	stats = accel.GetStatistics()
	if stats.FramesEncoded != 10 {
		t.Errorf("FramesEncoded = %d, expected 10", stats.FramesEncoded)
	}

	if stats.AverageEncodeTime == 0 {
		t.Error("Average encode time should be calculated")
	}
}

func TestHardwareAcceleratorReset(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	// Encode a frame
	input := make([]byte, 640*480*3/2)
	accel.EncodeFrame(input, 640, 480)

	stats := accel.GetStatistics()
	if stats.FramesEncoded != 1 {
		t.Error("Should have encoded 1 frame")
	}

	// Reset
	accel.Reset()

	stats = accel.GetStatistics()
	if stats.FramesEncoded != 0 {
		t.Errorf("After reset, FramesEncoded = %d, expected 0", stats.FramesEncoded)
	}
}

func TestHardwareAcceleratorClose(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	err := accel.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if accel.IsAvailable() {
		t.Error("Accelerator should not be available after Close")
	}
}

func TestHardwareAcceleratorGetters(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	if accel.GetType() != AccelNone {
		t.Errorf("GetType() = %s, expected %s", accel.GetType(), AccelNone)
	}

	maxRes := accel.GetMaxResolution()
	if maxRes.Width <= 0 || maxRes.Height <= 0 {
		t.Error("Max resolution should be positive")
	}

	codecs := accel.GetSupportedCodecs()
	if len(codecs) == 0 {
		t.Error("Should support at least one codec")
	}
}

// Accelerator Type Tests

func TestAcceleratorTypes(t *testing.T) {
	types := []AcceleratorType{
		AccelNVIDIA,
		AccelIntelQSV,
		AccelAMD,
		AccelAppleVT,
		AccelVAAPI,
		AccelVideoCore,
		AccelNone,
	}

	for _, accelType := range types {
		if string(accelType) == "" {
			t.Errorf("Accelerator type %v has empty string", accelType)
		}
	}
}

func TestDetectAccelerator(t *testing.T) {
	accelType := DetectAccelerator()

	if accelType == "" {
		t.Error("DetectAccelerator should return a valid type")
	}

	t.Logf("Detected accelerator: %s", accelType)

	// On most systems without GPU, should detect AccelNone
	// On macOS, should detect AccelAppleVT
	if runtime.GOOS == "darwin" {
		if accelType != AccelAppleVT && accelType != AccelNone {
			t.Logf("Note: Expected AccelAppleVT on macOS, got %s", accelType)
		}
	}
}

func TestListAvailableAccelerators(t *testing.T) {
	accelerators := ListAvailableAccelerators()

	if len(accelerators) == 0 {
		t.Error("Should have at least one accelerator (AccelNone)")
	}

	// AccelNone should always be available
	hasNone := false
	for _, accel := range accelerators {
		if accel == AccelNone {
			hasNone = true
			break
		}
	}

	if !hasNone {
		t.Error("AccelNone should always be in available accelerators")
	}

	t.Logf("Available accelerators: %v", accelerators)
}

func TestGetAcceleratorInfo(t *testing.T) {
	types := []AcceleratorType{
		AccelNVIDIA,
		AccelIntelQSV,
		AccelAMD,
		AccelAppleVT,
		AccelVAAPI,
		AccelVideoCore,
		AccelNone,
	}

	for _, accelType := range types {
		info, err := GetAcceleratorInfo(accelType)
		if err != nil {
			t.Errorf("GetAcceleratorInfo(%s) failed: %v", accelType, err)
			continue
		}

		if info == nil {
			t.Errorf("Info is nil for %s", accelType)
			continue
		}

		if info.Name == "" {
			t.Errorf("Name is empty for %s", accelType)
		}

		if info.MaxResolution.Width <= 0 || info.MaxResolution.Height <= 0 {
			t.Errorf("Invalid max resolution for %s", accelType)
		}

		if len(info.SupportedCodecs) == 0 {
			t.Errorf("No supported codecs for %s", accelType)
		}

		if info.MaxInstances <= 0 {
			t.Errorf("Invalid max instances for %s", accelType)
		}

		t.Logf("%s: %s, max %dx%d, %d codecs, %d instances",
			accelType, info.Name,
			info.MaxResolution.Width, info.MaxResolution.Height,
			len(info.SupportedCodecs), info.MaxInstances)
	}
}

func TestGetAcceleratorInfoInvalid(t *testing.T) {
	_, err := GetAcceleratorInfo("invalid")
	if err == nil {
		t.Error("GetAcceleratorInfo should fail for invalid type")
	}
}

// Platform-Specific Tests

func TestNVIDIAAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNVIDIA,
		CodecType: CodecTypeH264,
		Fallback:  true, // Fall back if NVIDIA not available
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create NVIDIA accelerator: %v", err)
	}

	// Should fall back to software if NVIDIA not available
	if accel.GetType() == AccelNone {
		t.Log("NVIDIA not available, fell back to software")
	} else {
		t.Logf("Using accelerator: %s", accel.GetType())
	}
}

func TestIntelQSVAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelIntelQSV,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create Intel QSV accelerator: %v", err)
	}

	if accel.GetType() == AccelNone {
		t.Log("Intel QSV not available, fell back to software")
	}
}

func TestAMDAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelAMD,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create AMD accelerator: %v", err)
	}

	if accel.GetType() == AccelNone {
		t.Log("AMD not available, fell back to software")
	}
}

func TestAppleVTAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelAppleVT,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create Apple VT accelerator: %v", err)
	}

	if runtime.GOOS == "darwin" {
		if accel.GetType() != AccelAppleVT {
			t.Log("VideoToolbox might not be available on this macOS version")
		}
	} else {
		if accel.GetType() != AccelNone {
			t.Error("VideoToolbox should only be available on macOS")
		}
	}
}

func TestVAAPIAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelVAAPI,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create VA-API accelerator: %v", err)
	}

	if runtime.GOOS == "linux" {
		if accel.GetType() == AccelNone {
			t.Log("VA-API not available on this Linux system")
		}
	} else {
		if accel.GetType() != AccelNone {
			t.Error("VA-API should only be available on Linux")
		}
	}
}

func TestVideoCoreAccelerator(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelVideoCore,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, err := NewHardwareAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create VideoCore accelerator: %v", err)
	}

	if accel.GetType() == AccelNone {
		t.Log("VideoCore not available (not on Raspberry Pi)")
	}
}

// Concurrent Encoding Tests

func TestConcurrentEncoding(t *testing.T) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	// Encode multiple frames concurrently
	const numFrames = 10
	done := make(chan bool, numFrames)
	input := make([]byte, 640*480*3/2)

	for i := 0; i < numFrames; i++ {
		go func() {
			_, err := accel.EncodeFrame(input, 640, 480)
			if err != nil {
				t.Errorf("Concurrent encode failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < numFrames; i++ {
		<-done
	}

	stats := accel.GetStatistics()
	if stats.FramesEncoded != numFrames {
		t.Errorf("FramesEncoded = %d, expected %d", stats.FramesEncoded, numFrames)
	}
}

// Resolution Tests

func TestResolution(t *testing.T) {
	resolutions := []struct {
		width  int
		height int
		name   string
	}{
		{640, 480, "VGA"},
		{1280, 720, "720p"},
		{1920, 1080, "1080p"},
		{3840, 2160, "4K"},
	}

	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)

	for _, res := range resolutions {
		input := make([]byte, res.width*res.height*3/2)

		_, err := accel.EncodeFrame(input, res.width, res.height)
		if err != nil {
			t.Errorf("Failed to encode %s (%dx%d): %v",
				res.name, res.width, res.height, err)
		} else {
			t.Logf("Successfully encoded %s (%dx%d)",
				res.name, res.width, res.height)
		}
	}
}

// Benchmarks

func BenchmarkHardwareEncodeH264_640x480(b *testing.B) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)
	input := make([]byte, 640*480*3/2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accel.EncodeFrame(input, 640, 480)
	}
}

func BenchmarkHardwareEncodeH264_1920x1080(b *testing.B) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)
	input := make([]byte, 1920*1080*3/2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accel.EncodeFrame(input, 1920, 1080)
	}
}

func BenchmarkHardwareDecode(b *testing.B) {
	config := AcceleratorConfig{
		Type:      AccelNone,
		CodecType: CodecTypeH264,
		Fallback:  true,
	}

	accel, _ := NewHardwareAccelerator(config)
	input := []byte{0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1F, 0x8C}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accel.DecodeFrame(input)
	}
}

func BenchmarkDetectAccelerator(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DetectAccelerator()
	}
}

func BenchmarkListAvailableAccelerators(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ListAvailableAccelerators()
	}
}
