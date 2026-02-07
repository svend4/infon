package codec

import (
	"testing"
	"time"
)

func TestCodecTypes(t *testing.T) {
	types := []CodecType{
		CodecTypeH264,
		CodecTypeH265,
		CodecTypeVP8,
		CodecTypeVP9,
		CodecTypeOpus,
		CodecTypeAAC,
		CodecTypePCM,
	}

	for _, ct := range types {
		if string(ct) == "" {
			t.Errorf("Codec type %v has empty string", ct)
		}
	}
}

func TestMediaTypes(t *testing.T) {
	if MediaTypeAudio != "audio" {
		t.Errorf("MediaTypeAudio = %s, expected audio", MediaTypeAudio)
	}

	if MediaTypeVideo != "video" {
		t.Errorf("MediaTypeVideo = %s, expected video", MediaTypeVideo)
	}
}

func TestDefaultVideoConfig(t *testing.T) {
	cfg := DefaultVideoConfig(CodecTypeH264, 1920, 1080)

	if cfg.Type != CodecTypeH264 {
		t.Errorf("Type = %s, expected h264", cfg.Type)
	}

	if cfg.Width != 1920 || cfg.Height != 1080 {
		t.Errorf("Resolution = %dx%d, expected 1920x1080", cfg.Width, cfg.Height)
	}

	if cfg.Bitrate != 2_000_000 {
		t.Errorf("Bitrate = %d, expected 2000000", cfg.Bitrate)
	}

	if cfg.Framerate != 30.0 {
		t.Errorf("Framerate = %f, expected 30.0", cfg.Framerate)
	}

	if cfg.KeyframeInterval != 60 {
		t.Errorf("KeyframeInterval = %d, expected 60", cfg.KeyframeInterval)
	}
}

func TestDefaultAudioConfig(t *testing.T) {
	cfg := DefaultAudioConfig(CodecTypeOpus)

	if cfg.Type != CodecTypeOpus {
		t.Errorf("Type = %s, expected opus", cfg.Type)
	}

	if cfg.Bitrate != 128_000 {
		t.Errorf("Bitrate = %d, expected 128000", cfg.Bitrate)
	}

	if cfg.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, expected 48000", cfg.SampleRate)
	}

	if cfg.Channels != 2 {
		t.Errorf("Channels = %d, expected 2", cfg.Channels)
	}
}

func TestBaseCodec(t *testing.T) {
	cfg := CodecConfig{
		Type:      CodecTypeH264,
		Bitrate:   1000000,
		Width:     1280,
		Height:    720,
		Framerate: 30.0,
	}

	bc := NewBaseCodec(CodecTypeH264, MediaTypeVideo, cfg)

	if bc.Type() != CodecTypeH264 {
		t.Errorf("Type() = %s, expected h264", bc.Type())
	}

	if bc.MediaType() != MediaTypeVideo {
		t.Errorf("MediaType() = %s, expected video", bc.MediaType())
	}

	// Test statistics
	stats := bc.GetStatistics()
	if stats.FramesEncoded != 0 {
		t.Errorf("Initial FramesEncoded = %d, expected 0", stats.FramesEncoded)
	}

	// Record some encoding
	bc.recordEncode(1000, 100, 10*time.Millisecond)
	bc.recordEncode(1000, 100, 10*time.Millisecond)

	stats = bc.GetStatistics()
	if stats.FramesEncoded != 2 {
		t.Errorf("FramesEncoded = %d, expected 2", stats.FramesEncoded)
	}

	if stats.BytesDecoded != 2000 {
		t.Errorf("BytesDecoded = %d, expected 2000", stats.BytesDecoded)
	}

	if stats.BytesEncoded != 200 {
		t.Errorf("BytesEncoded = %d, expected 200", stats.BytesEncoded)
	}

	if stats.AverageEncodeTime != 10*time.Millisecond {
		t.Errorf("AverageEncodeTime = %v, expected 10ms", stats.AverageEncodeTime)
	}

	// Test reset
	bc.Reset()
	stats = bc.GetStatistics()
	if stats.FramesEncoded != 0 {
		t.Errorf("After reset, FramesEncoded = %d, expected 0", stats.FramesEncoded)
	}
}

func TestCodecFactory(t *testing.T) {
	factory := NewCodecFactory()

	if factory == nil {
		t.Fatal("NewCodecFactory() returned nil")
	}

	// Test listing codecs
	codecs := factory.ListCodecs()
	if len(codecs) == 0 {
		t.Error("No codecs registered")
	}

	// Test creating H.264 codec
	cfg := DefaultVideoConfig(CodecTypeH264, 640, 480)
	codec, err := factory.Create(CodecTypeH264, cfg)
	if err != nil {
		t.Fatalf("Failed to create H.264 codec: %v", err)
	}

	if codec == nil {
		t.Fatal("Created codec is nil")
	}

	if codec.Type() != CodecTypeH264 {
		t.Errorf("Codec type = %s, expected h264", codec.Type())
	}

	codec.Close()
}

func TestGlobalFactory(t *testing.T) {
	codecs := ListAvailableCodecs()

	if len(codecs) == 0 {
		t.Error("No codecs available in global factory")
	}

	// Test creating a codec
	cfg := DefaultAudioConfig(CodecTypeOpus)
	codec, err := CreateCodec(CodecTypeOpus, cfg)
	if err != nil {
		t.Fatalf("Failed to create Opus codec: %v", err)
	}

	if codec == nil {
		t.Fatal("Created codec is nil")
	}

	codec.Close()
}

// H.264 Tests

func TestH264Codec(t *testing.T) {
	cfg := DefaultVideoConfig(CodecTypeH264, 640, 480)

	codec, err := NewH264Codec(cfg)
	if err != nil {
		t.Fatalf("Failed to create H.264 codec: %v", err)
	}
	defer codec.Close()

	h264 := codec.(*H264Codec)

	// Test encode
	yuv := make([]byte, 640*480*3/2) // YUV420
	for i := range yuv {
		yuv[i] = byte(i % 256)
	}

	encoded, err := h264.Encode(yuv)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data is empty")
	}

	t.Logf("Encoded %d bytes to %d bytes (%.2f%% compression)",
		len(yuv), len(encoded), 100.0*float64(len(encoded))/float64(len(yuv)))

	// Test decode
	decoded, err := h264.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(decoded) != len(yuv) {
		t.Errorf("Decoded size = %d, expected %d", len(decoded), len(yuv))
	}

	// Test statistics
	stats := h264.GetStatistics()
	if stats.FramesEncoded != 1 {
		t.Errorf("FramesEncoded = %d, expected 1", stats.FramesEncoded)
	}

	if stats.FramesDecoded != 1 {
		t.Errorf("FramesDecoded = %d, expected 1", stats.FramesDecoded)
	}
}

func TestH264SetBitrate(t *testing.T) {
	cfg := DefaultVideoConfig(CodecTypeH264, 640, 480)
	codec, _ := NewH264Codec(cfg)
	defer codec.Close()

	h264 := codec.(VideoCodec)

	err := h264.SetBitrate(1_000_000)
	if err != nil {
		t.Errorf("SetBitrate failed: %v", err)
	}

	// Test invalid bitrate
	err = h264.SetBitrate(50)
	if err == nil {
		t.Error("SetBitrate should fail for too low bitrate")
	}
}

func TestH264SetResolution(t *testing.T) {
	cfg := DefaultVideoConfig(CodecTypeH264, 640, 480)
	codec, _ := NewH264Codec(cfg)
	defer codec.Close()

	h264 := codec.(VideoCodec)

	err := h264.SetResolution(1920, 1080)
	if err != nil {
		t.Errorf("SetResolution failed: %v", err)
	}

	// Test invalid resolution
	err = h264.SetResolution(100, 100)
	if err == nil {
		t.Error("SetResolution should fail for too small resolution")
	}
}

// VP8 Tests

func TestVP8Codec(t *testing.T) {
	cfg := DefaultVideoConfig(CodecTypeVP8, 640, 480)

	codec, err := NewVP8Codec(cfg)
	if err != nil {
		t.Fatalf("Failed to create VP8 codec: %v", err)
	}
	defer codec.Close()

	// Test encode
	yuv := make([]byte, 640*480*3/2)
	for i := range yuv {
		yuv[i] = byte(i % 256)
	}

	encoded, err := codec.Encode(yuv)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data is empty")
	}

	// Test decode
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(decoded) != len(yuv) {
		t.Errorf("Decoded size = %d, expected %d", len(decoded), len(yuv))
	}
}

func TestVP9Codec(t *testing.T) {
	cfg := DefaultVideoConfig(CodecTypeVP9, 640, 480)

	codec, err := NewVP9Codec(cfg)
	if err != nil {
		t.Fatalf("Failed to create VP9 codec: %v", err)
	}
	defer codec.Close()

	if codec.Type() != CodecTypeVP9 {
		t.Errorf("Codec type = %s, expected vp9", codec.Type())
	}

	// Test basic encode/decode
	yuv := make([]byte, 640*480*3/2)
	encoded, err := codec.Encode(yuv)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data is empty")
	}
}

// Opus Tests

func TestOpusCodec(t *testing.T) {
	cfg := DefaultAudioConfig(CodecTypeOpus)

	codec, err := NewOpusCodec(cfg)
	if err != nil {
		t.Fatalf("Failed to create Opus codec: %v", err)
	}
	defer codec.Close()

	opus := codec.(*OpusCodec)

	// Test encode (960 samples * 2 bytes/sample * 2 channels = 3840 bytes)
	pcm := make([]byte, 960*2*2)
	for i := range pcm {
		pcm[i] = byte(i % 256)
	}

	encoded, err := opus.Encode(pcm)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data is empty")
	}

	t.Logf("Opus encoded %d bytes to %d bytes", len(pcm), len(encoded))

	// Test decode
	decoded, err := opus.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(decoded) == 0 {
		t.Error("Decoded data is empty")
	}

	// Test statistics
	stats := opus.GetStatistics()
	if stats.FramesEncoded != 1 {
		t.Errorf("FramesEncoded = %d, expected 1", stats.FramesEncoded)
	}
}

func TestOpusSetSampleRate(t *testing.T) {
	cfg := DefaultAudioConfig(CodecTypeOpus)
	codec, _ := NewOpusCodec(cfg)
	defer codec.Close()

	opus := codec.(AudioCodec)

	// Test valid sample rates
	validRates := []int{8000, 12000, 16000, 24000, 48000}
	for _, rate := range validRates {
		err := opus.SetSampleRate(rate)
		if err != nil {
			t.Errorf("SetSampleRate(%d) failed: %v", rate, err)
		}
	}

	// Test invalid sample rate
	err := opus.SetSampleRate(44100)
	if err == nil {
		t.Error("SetSampleRate should fail for invalid rate 44100")
	}
}

// AAC Tests

func TestAACCodec(t *testing.T) {
	cfg := DefaultAudioConfig(CodecTypeAAC)

	codec, err := NewAACCodec(cfg)
	if err != nil {
		t.Fatalf("Failed to create AAC codec: %v", err)
	}
	defer codec.Close()

	// Test encode
	pcm := make([]byte, 2048*2*2) // 2048 samples * 2 bytes * 2 channels
	for i := range pcm {
		pcm[i] = byte(i % 256)
	}

	encoded, err := codec.Encode(pcm)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) < 7 {
		t.Error("Encoded data should include ADTS header (7 bytes)")
	}

	// Test decode
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Decoded data should be filled (may be smaller than original due to framing)
	t.Logf("AAC decoded %d bytes to %d bytes", len(encoded), len(decoded))
}

func TestAACProfiles(t *testing.T) {
	profiles := []AACProfile{AACProfileLC, AACProfileHE, AACProfileHEv2}

	for _, profile := range profiles {
		if int(profile) < 1 || int(profile) > 3 {
			t.Errorf("Invalid AAC profile value: %d", profile)
		}
	}
}

// PCM Tests

func TestPCMCodec(t *testing.T) {
	cfg := DefaultAudioConfig(CodecTypePCM)

	codec, err := NewPCMCodec(cfg)
	if err != nil {
		t.Fatalf("Failed to create PCM codec: %v", err)
	}
	defer codec.Close()

	// Test encode (passthrough)
	pcm := make([]byte, 1024)
	for i := range pcm {
		pcm[i] = byte(i % 256)
	}

	encoded, err := codec.Encode(pcm)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) != len(pcm) {
		t.Errorf("Encoded size = %d, expected %d", len(encoded), len(pcm))
	}

	// Verify passthrough
	for i := range pcm {
		if encoded[i] != pcm[i] {
			t.Errorf("Encoded data differs at index %d", i)
			break
		}
	}

	// Test decode (passthrough)
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(decoded) != len(encoded) {
		t.Errorf("Decoded size = %d, expected %d", len(decoded), len(encoded))
	}
}

// Benchmarks

func BenchmarkH264Encode(b *testing.B) {
	cfg := DefaultVideoConfig(CodecTypeH264, 640, 480)
	codec, _ := NewH264Codec(cfg)
	defer codec.Close()

	yuv := make([]byte, 640*480*3/2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Encode(yuv)
	}
}

func BenchmarkOpusEncode(b *testing.B) {
	cfg := DefaultAudioConfig(CodecTypeOpus)
	codec, _ := NewOpusCodec(cfg)
	defer codec.Close()

	pcm := make([]byte, 960*2*2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Encode(pcm)
	}
}

func BenchmarkPCMEncode(b *testing.B) {
	cfg := DefaultAudioConfig(CodecTypePCM)
	codec, _ := NewPCMCodec(cfg)
	defer codec.Close()

	pcm := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Encode(pcm)
	}
}
