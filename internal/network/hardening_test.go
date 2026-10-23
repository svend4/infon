package network

import "testing"

// These tests pin the input-validation guards added so that a single malformed UDP
// datagram cannot panic or exhaust memory in a receiver. Each case is a payload a
// peer (or a third party spoofing one) could send; none involves real hardware.

func TestParseNACKPayloadRejectsOverflowCount(t *testing.T) {
	// count=16384 makes the old uint16 arithmetic 2+count*4 wrap to 0, so the length
	// guard passed for a 2-byte payload and the loop sliced out of range.
	payload := []byte{0x40, 0x00} // count = 16384, no sequence bytes follow
	if _, err := ParseNACKPayload(payload); err == nil {
		t.Fatal("expected error for a count with no backing sequence bytes, got nil")
	}
	// A short but non-wrapping claim must also be rejected, not panic.
	if _, err := ParseNACKPayload([]byte{0x00, 0x05, 0x00, 0x00}); err == nil {
		t.Fatal("expected error for count=5 with only 1 sequence present")
	}
	// A well-formed payload still parses.
	seqs, err := ParseNACKPayload([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x2a})
	if err != nil || len(seqs) != 1 || seqs[0] != 42 {
		t.Fatalf("well-formed NACK: got %v, %v", seqs, err)
	}
}

func TestParseFragmentHeaderGuards(t *testing.T) {
	hdr := func(frameID uint32, fragID, total uint16) []byte {
		return []byte{
			byte(frameID >> 24), byte(frameID >> 16), byte(frameID >> 8), byte(frameID),
			byte(fragID >> 8), byte(fragID), byte(total >> 8), byte(total),
		}
	}
	cases := []struct {
		name    string
		payload []byte
		haveLen int
		wantOK  bool
	}{
		{"too short", []byte{1, 2, 3}, -1, false},
		{"totalFrags zero", hdr(1, 0, 0), -1, false},
		{"fragID >= totalFrags", hdr(1, 5, 1), -1, false},
		{"totalFrags absurd", hdr(1, 0, MaxFragsPerFrame+1), -1, false},
		{"buffer length disagrees", hdr(1, 0, 4), 2, false},
		{"valid", hdr(1, 0, 1), -1, true},
		{"valid matching buffer", hdr(1, 2, 4), 4, true},
	}
	for _, c := range cases {
		_, _, _, ok := ParseFragmentHeader(c.payload, c.haveLen)
		if ok != c.wantOK {
			t.Errorf("%s: ok=%v, want %v", c.name, ok, c.wantOK)
		}
	}
}

func TestAssembleDataRejectsForgedChunkLen(t *testing.T) {
	// One fragment whose declared chunkLen (0xFFFF) far exceeds the bytes present.
	frag := make([]byte, 20)
	frag[8], frag[9] = 0xFF, 0xFF
	if _, err := AssembleData([][]byte{frag}); err == nil {
		t.Fatal("expected error for chunkLen exceeding payload, got nil")
	}
}

func TestAssembleFrameRejectsAbsurdDimensions(t *testing.T) {
	frag := make([]byte, FragmentHeaderSize)
	frag[8], frag[9] = 0xFF, 0xFF   // width = 65535
	frag[10], frag[11] = 0xFF, 0xFF // height = 65535
	if _, err := AssembleFrame([][]byte{frag}); err == nil {
		t.Fatal("expected error for out-of-bound frame dimensions, got nil")
	}
}

func TestJitterBufferToleratesSequenceGaps(t *testing.T) {
	// Audio sequence numbers are drawn from a counter shared with video/text, so they
	// arrive monotonic but not contiguous and never start at 0. The buffer must still
	// hand them out in order rather than stalling on the missing nextExpected==0.
	jb := NewJitterBuffer(50)
	jb.minDelay = 0     // let adaptDelay settle to 0 so the test can drain in a tight loop
	jb.currentDelay = 0 // output as soon as present
	for _, seq := range []uint32{7, 9, 10, 13} {
		if !jb.Add(&Packet{Sequence: seq}) {
			t.Fatalf("Add(%d) rejected", seq)
		}
	}
	var got []uint32
	for {
		p := jb.Get()
		if p == nil {
			break
		}
		got = append(got, p.Sequence)
	}
	want := []uint32{7, 9, 10, 13}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
