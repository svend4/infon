package framecodec

import (
	"bytes"
	"math/rand"
	"testing"
)

func roundTrip(t *testing.T, prev, cur []byte) []byte {
	t.Helper()
	enc := Encode(prev, cur)
	got, err := Decode(prev, enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(got, cur) {
		t.Fatalf("round trip mismatch (mode %d)", enc[0])
	}
	return enc
}

func TestRoundTripAllRegimes(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	prev := make([]byte, 4096)
	for i := range prev {
		prev[i] = byte(rng.Intn(256))
	}
	// low motion: a handful of bytes changed
	low := append([]byte(nil), prev...)
	for i := 0; i < 5; i++ {
		low[rng.Intn(len(low))] ^= 0xff
	}
	// static (very compressible)
	static := make([]byte, 4096)
	// incompressible random of a different length (no delta)
	randDiff := make([]byte, 4097)
	for i := range randDiff {
		randDiff[i] = byte(rng.Intn(256))
	}

	if e := roundTrip(t, prev, low); e[0] != ModeDelta {
		t.Errorf("low-motion frame should pick DELTA, got mode %d (%d bytes)", e[0], len(e))
	}
	if e := roundTrip(t, nil, static); e[0] != ModeZlib {
		t.Errorf("a static frame should pick ZLIB, got mode %d", e[0])
	}
	if e := roundTrip(t, prev, randDiff); e[0] != ModeRaw {
		t.Errorf("incompressible, size-changed frame should fall back to RAW, got mode %d", e[0])
	}
	roundTrip(t, nil, prev) // first frame (no prev): must still round-trip
}

func TestDeltaIsTinyForOneChange(t *testing.T) {
	prev := bytes.Repeat([]byte{7}, 10000)
	cur := append([]byte(nil), prev...)
	cur[5000] = 9
	enc := Encode(prev, cur)
	if enc[0] != ModeDelta {
		t.Fatalf("expected DELTA, got mode %d", enc[0])
	}
	if len(enc) > 12 {
		t.Errorf("one changed byte should encode tiny, got %d bytes", len(enc))
	}
}

func TestStreamSavesOnLowMotion(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	w, h := 80, 45
	frame := make([]byte, w*h*11) // mimic an encoded terminal frame
	for i := range frame {
		frame[i] = byte(rng.Intn(256))
	}
	var s Stream
	var d Decoder
	for f := 0; f < 30; f++ {
		// nudge a small moving region each frame (low motion)
		for k := 0; k < 40; k++ {
			frame[(f*37+k)%len(frame)] = byte(rng.Intn(256))
		}
		enc := s.Push(frame)
		got, err := d.Push(enc)
		if err != nil {
			t.Fatalf("decode frame %d: %v", f, err)
		}
		if !bytes.Equal(got, frame) {
			t.Fatalf("stream round trip broke at frame %d", f)
		}
	}
	if s.Savings() <= 0 {
		t.Errorf("low-motion stream should save on the wire, got %.1f%%", s.Savings()*100)
	}
	if s.Delta == 0 {
		t.Errorf("expected some DELTA frames, got raw=%d zlib=%d delta=%d", s.Raw, s.Zlib, s.Delta)
	}
	if s.Frames != 30 {
		t.Errorf("frames = %d, want 30", s.Frames)
	}
}

func TestEmptyDecodeErrors(t *testing.T) {
	if _, err := Decode(nil, nil); err == nil {
		t.Error("decoding an empty frame should error")
	}
}
