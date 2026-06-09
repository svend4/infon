package raydir

import (
	"reflect"
	"testing"
)

// A voice frame round-trips through encode/decode, origin and samples intact.
func TestVoiceRoundTrip(t *testing.T) {
	samples := []int16{0, 1, -1, 32767, -32768, 1234}
	origin, got, err := DecodeVoice(EncodeVoice(0xDEADBEEF, samples))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if origin != 0xDEADBEEF {
		t.Errorf("origin = %x, want DEADBEEF", origin)
	}
	if !reflect.DeepEqual(got, samples) {
		t.Errorf("samples = %v, want %v", got, samples)
	}
}

// Decoding rejects a truncated frame instead of panicking.
func TestDecodeVoiceBad(t *testing.T) {
	if _, _, err := DecodeVoice([]byte{1, 2, 3}); err == nil {
		t.Error("short header should error")
	}
	good := EncodeVoice(1, []int16{5, 6, 7})
	if _, _, err := DecodeVoice(good[:len(good)-1]); err == nil {
		t.Error("truncated samples should error")
	}
}

// Two speakers talking at once are summed into one frame, not serialised.
func TestMixSumsConcurrent(t *testing.T) {
	m := NewVoiceMixer(4)
	m.Add(1, []int16{100, 100, 100, 100})
	m.Add(2, []int16{10, 20, 30, 40})
	got := m.Mix()
	want := []int16{110, 120, 130, 140}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mixed = %v, want %v", got, want)
	}
	if m.Mix() != nil {
		t.Error("after draining, Mix should return nil (silence)")
	}
}

// One speaker's longer stream drains a frame at a time, in order.
func TestMixDrainsInFrames(t *testing.T) {
	m := NewVoiceMixer(2)
	m.Add(1, []int16{1, 2, 3, 4, 5})
	for _, want := range [][]int16{{1, 2}, {3, 4}, {5, 0}} {
		if got := m.Mix(); !reflect.DeepEqual(got, want) {
			t.Fatalf("frame = %v, want %v", got, want)
		}
	}
	if m.Mix() != nil {
		t.Error("stream exhausted, Mix should be nil")
	}
}

// Summed samples saturate at the int16 range instead of wrapping.
func TestMixSaturates(t *testing.T) {
	m := NewVoiceMixer(2)
	m.Add(1, []int16{30000, -30000})
	m.Add(2, []int16{30000, -30000})
	got := m.Mix()
	if got[0] != 32767 || got[1] != -32768 {
		t.Fatalf("saturation failed: %v", got)
	}
}

// A drained speaker is pruned from the active set.
func TestMixPrunesSilent(t *testing.T) {
	m := NewVoiceMixer(4)
	m.Add(1, []int16{1, 2})
	if m.Active() != 1 {
		t.Fatalf("Active = %d, want 1", m.Active())
	}
	m.Mix()
	if m.Active() != 0 {
		t.Errorf("drained speaker should be pruned, Active = %d", m.Active())
	}
}

// A source's buffer is bounded so a stuck/fast sender can't grow latency forever.
func TestMixBounded(t *testing.T) {
	m := NewVoiceMixer(2) // maxBuf = 100
	big := make([]int16, 500)
	for i := range big {
		big[i] = int16(i)
	}
	m.Add(1, big)
	m.mu.Lock()
	n := len(m.bufs[1])
	m.mu.Unlock()
	if n != m.maxBuf {
		t.Fatalf("buffer should cap at %d, got %d", m.maxBuf, n)
	}
}
