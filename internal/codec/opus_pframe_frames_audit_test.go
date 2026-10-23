package codec

import "testing"

// The frame count travels in one byte. An input of 256 or more frames must
// either be rejected by Encode or decode back to its original length; it must
// not wrap to a short (or empty) output.
func TestOpusPframeFrameCountDoesNotWrap(t *testing.T) {
	c, err := NewOpusCodec(DefaultAudioConfig(CodecTypeOpus))
	if err != nil {
		t.Fatalf("NewOpusCodec: %v", err)
	}
	opus := c.(*OpusCodec)
	frameBytes := opus.GetFrameSize() * 2 * opus.GetChannels()

	for _, frames := range []int{1, 255, 256, 257, 300} {
		pcm := make([]byte, frames*frameBytes)
		encoded, err := c.Encode(pcm)
		if err != nil {
			if frames <= 255 {
				t.Fatalf("%d frames: Encode: %v", frames, err)
			}
			continue
		}
		decoded, err := c.Decode(encoded)
		if err != nil {
			t.Fatalf("%d frames: Decode: %v", frames, err)
		}
		if len(decoded) != len(pcm) {
			t.Errorf("%d frames: decoded %d bytes, want %d", frames, len(decoded), len(pcm))
		}
	}
}

// A codec constructed at a sample rate must use the same 20 ms frame size as
// one switched to that rate with SetSampleRate, or two peers at the same rate
// disagree on how many samples a frame holds.
func TestOpusPframeFrameSizeFollowsSampleRate(t *testing.T) {
	for _, rate := range []int{8000, 12000, 16000, 24000, 48000} {
		cfg := DefaultAudioConfig(CodecTypeOpus)
		cfg.SampleRate = rate
		c, err := NewOpusCodec(cfg)
		if err != nil {
			t.Fatalf("NewOpusCodec(%d): %v", rate, err)
		}
		opus := c.(*OpusCodec)
		if got, want := opus.GetFrameSize(), rate*20/1000; got != want {
			t.Errorf("NewOpusCodec(%d): frame size %d, want %d (20 ms)", rate, got, want)
		}

		other, err := NewOpusCodec(DefaultAudioConfig(CodecTypeOpus))
		if err != nil {
			t.Fatalf("NewOpusCodec: %v", err)
		}
		switched := other.(*OpusCodec)
		if err := switched.SetSampleRate(rate); err != nil {
			t.Fatalf("SetSampleRate(%d): %v", rate, err)
		}
		if opus.GetFrameSize() != switched.GetFrameSize() {
			t.Errorf("rate %d: constructed frame size %d != SetSampleRate frame size %d",
				rate, opus.GetFrameSize(), switched.GetFrameSize())
		}
	}
}
