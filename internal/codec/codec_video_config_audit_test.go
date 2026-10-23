package codec

import "testing"

// codecVideoAuditCtors are the video codec constructors exercised by the
// codec_video audit tests.
var codecVideoAuditCtors = map[string]func(CodecConfig) (Codec, error){
	"h264": NewH264Codec,
	"vp8":  NewVP8Codec,
	"vp9":  NewVP9Codec,
}

// codecVideoAuditCall runs f and turns a panic into a test failure.
func codecVideoAuditCall(t *testing.T, what string, f func() ([]byte, error)) ([]byte, error) {
	t.Helper()
	var out []byte
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%s panicked: %v", what, r)
			}
		}()
		out, err = f()
	}()
	return out, err
}

// A hand-built CodecConfig that leaves Framerate/Bitrate at zero (or sets a
// framerate below 1 fps, or a bitrate below 8 bits per frame) is accepted by the
// constructors, so Encode must not divide by zero on it.
func TestCodecVideoAuditEncodeDegenerateRateConfig(t *testing.T) {
	const w, h = 16, 16
	cfgs := []CodecConfig{
		{Width: w, Height: h},
		{Width: w, Height: h, Framerate: 30},
		{Width: w, Height: h, Bitrate: 2_000_000},
		{Width: w, Height: h, Framerate: 0.5, Bitrate: 2_000_000},
		{Width: w, Height: h, Framerate: 30, Bitrate: 100},
	}

	for name, ctor := range codecVideoAuditCtors {
		for _, cfg := range cfgs {
			c, err := ctor(cfg)
			if err != nil {
				t.Fatalf("%s %+v: constructor: %v", name, cfg, err)
			}
			frame := make([]byte, w*h*3/2)
			for i := range frame {
				frame[i] = byte(i%200 + 10)
			}

			enc, err := codecVideoAuditCall(t, name+" Encode", func() ([]byte, error) { return c.Encode(frame) })
			if err != nil {
				t.Fatalf("%s %+v: Encode: %v", name, cfg, err)
			}
			dec, err := codecVideoAuditCall(t, name+" Decode", func() ([]byte, error) { return c.Decode(enc) })
			if err != nil {
				t.Fatalf("%s %+v: Decode: %v", name, cfg, err)
			}
			if len(dec) != len(frame) {
				t.Fatalf("%s %+v: decoded %d bytes, want %d", name, cfg, len(dec), len(frame))
			}
		}
	}
}
