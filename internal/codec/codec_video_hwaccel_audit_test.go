package codec

import (
	"bytes"
	"testing"
)

// The software (AccelNone) decode path strips a 3-byte start code. Inputs that
// are too short for it, or that do not carry it, must be rejected with an error
// instead of panicking on input[3:].
func TestCodecVideoAuditDecodeFrameShortInput(t *testing.T) {
	accel, err := NewHardwareAccelerator(AcceleratorConfig{Type: AccelNone, CodecType: CodecTypeH264})
	if err != nil {
		t.Fatalf("NewHardwareAccelerator: %v", err)
	}

	for _, in := range [][]byte{nil, {}, {0x00}, {0x00, 0x00}, {0x01, 0x02, 0x03, 0x04}} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("DecodeFrame(%x) panicked: %v", in, r)
				}
			}()
			if _, _, _, err := accel.DecodeFrame(in); err == nil {
				t.Errorf("DecodeFrame(%x): expected error", in)
			}
		}()
	}
	if got := accel.GetStatistics().Errors; got != 5 {
		t.Errorf("Errors = %d, want 5", got)
	}

	// A frame produced by EncodeFrame still decodes to the original payload.
	payload := []byte{0x67, 0x42, 0x00, 0x1F, 0x8C}
	enc, err := accel.EncodeFrame(payload, 640, 480)
	if err != nil {
		t.Fatalf("EncodeFrame: %v", err)
	}
	dec, _, _, err := accel.DecodeFrame(enc)
	if err != nil {
		t.Fatalf("DecodeFrame(EncodeFrame(x)): %v", err)
	}
	if !bytes.Equal(dec, payload) {
		t.Errorf("DecodeFrame(EncodeFrame(x)) = %x, want %x", dec, payload)
	}
}

// When the requested hardware is unavailable and Fallback is set, the returned
// accelerator must be a fully initialized software accelerator, not an
// uninitialized shell whose Encode/DecodeFrame always fail.
func TestCodecVideoAuditFallbackInitializesSoftware(t *testing.T) {
	types := []AcceleratorType{AccelNVIDIA, AccelIntelQSV, AccelAMD, AccelAppleVT, AccelVAAPI, AccelVideoCore, "bogus"}
	for _, typ := range types {
		accel, err := NewHardwareAccelerator(AcceleratorConfig{Type: typ, CodecType: CodecTypeH264, Fallback: true})
		if err != nil {
			t.Fatalf("%s: NewHardwareAccelerator with Fallback: %v", typ, err)
		}
		if accel.GetType() != AccelNone {
			continue // the hardware is really present on this machine
		}

		if !accel.IsAvailable() {
			t.Errorf("%s: fallback accelerator IsAvailable() = false", typ)
		}
		if got, want := accel.GetMaxResolution(), (Resolution{Width: 7680, Height: 4320}); got != want {
			t.Errorf("%s: fallback max resolution = %v, want %v", typ, got, want)
		}

		payload := []byte{0x65, 0x88, 0x84}
		enc, err := accel.EncodeFrame(payload, 640, 480)
		if err != nil {
			t.Errorf("%s: fallback EncodeFrame: %v", typ, err)
			continue
		}
		dec, _, _, err := accel.DecodeFrame(enc)
		if err != nil {
			t.Errorf("%s: fallback DecodeFrame: %v", typ, err)
			continue
		}
		if !bytes.Equal(dec, payload) {
			t.Errorf("%s: fallback round trip = %x, want %x", typ, dec, payload)
		}
	}

	// Without Fallback an unavailable accelerator is still an error.
	if _, err := NewHardwareAccelerator(AcceleratorConfig{Type: "bogus", CodecType: CodecTypeH264}); err == nil {
		t.Error("unsupported accelerator without Fallback: expected error")
	}
}
