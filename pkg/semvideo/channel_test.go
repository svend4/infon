package semvideo_test

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/semvideo"
)

// Losing a P-frame freezes the receiver; the next keyframe re-syncs it exactly.
func TestReceiverResyncAfterLoss(t *testing.T) {
	fs := frames() // 3 marks frames (from semvideo_test.go)
	ecc := 16
	pkts := semvideo.EncodeFrames(fs, ecc, 2) // K, D, K
	if len(pkts) != 3 || pkts[0].Kind != semvideo.KeyFrame || pkts[2].Kind != semvideo.KeyFrame {
		t.Fatalf("unexpected GOP packetization: %+v", pkts)
	}
	r := semvideo.NewReceiver(ecc)
	r.Apply(pkts[0]) // keyframe 0
	if f, _ := r.Frame(); !reflect.DeepEqual(f.Marks.Rows, fs[0].Marks.Rows) {
		t.Fatal("frame 0 wrong")
	}
	r.MarkLoss() // packet 1 (a P-frame) dropped -> freeze, desync
	if f, _ := r.Frame(); !reflect.DeepEqual(f.Marks.Rows, fs[0].Marks.Rows) {
		t.Fatal("should have frozen on frame 0 after loss")
	}
	// corrupt a burst in the next keyframe; RS must still recover it.
	for i := 4; i < 4+ecc/2 && i < len(pkts[2].Payload); i++ {
		pkts[2].Payload[i] ^= 0xFF
	}
	r.Apply(pkts[2]) // keyframe 2 re-syncs
	if f, _ := r.Frame(); !reflect.DeepEqual(f.Marks.Rows, fs[2].Marks.Rows) {
		t.Fatal("did not re-sync to frame 2 after keyframe")
	}
}

// Wire round-trips the packet header.
func TestPacketWire(t *testing.T) {
	p := semvideo.Packet{Idx: 0x1234, Kind: semvideo.PFrame, Payload: []byte{1, 2, 3}}
	q, ok := semvideo.ParseWire(p.Wire())
	if !ok || q.Idx != p.Idx || q.Kind != p.Kind || !reflect.DeepEqual(q.Payload, p.Payload) {
		t.Fatalf("wire round-trip failed: %+v", q)
	}
}

func TestMarksIoU(t *testing.T) {
	fs := frames()
	if v := semvideo.MarksIoU(fs[0], fs[0]); v < 0.999 {
		t.Fatalf("self IoU = %.3f, want 1", v)
	}
	if v := semvideo.MarksIoU(fs[0], fs[2]); v <= 0 || v >= 1 {
		t.Fatalf("different-frame IoU = %.3f, want in (0,1)", v)
	}
}
