package semdist

import "testing"

func TestGopRateDistortion(t *testing.T) {
	s := Scene(20, 10, 8)
	pts := GopCurve(s, []int{2, 8}, 6, 0.25, 7)
	if len(pts) != 2 {
		t.Fatalf("got %d points", len(pts))
	}
	small, big := pts[0], pts[1] // gop 2, gop 8
	if small.Bytes <= big.Bytes {
		t.Fatalf("smaller gop should cost more bytes: %d vs %d", small.Bytes, big.Bytes)
	}
	if small.IoU < big.IoU {
		t.Fatalf("smaller gop should preserve >= meaning under loss: %.3f vs %.3f", small.IoU, big.IoU)
	}
}

func TestEccRateDistortion(t *testing.T) {
	s := Scene(20, 10, 8)
	pts := EccCurve(s, []int{2, 8}, 3, 7)
	lo, hi := pts[0], pts[1]
	// at ~constant frame size, parity trades capacity for resilience: more parity
	// heals more corruption, so more meaning survives.
	if !(hi.IoU > lo.IoU) {
		t.Fatalf("more parity should preserve more meaning: ecc8 %.3f vs ecc2 %.3f", hi.IoU, lo.IoU)
	}
	if hi.IoU < 0.999 {
		t.Fatalf("ecc 8 should heal 3-byte corruption (IoU %.3f)", hi.IoU)
	}
	if lo.IoU >= 0.999 {
		t.Fatalf("ecc 2 should NOT heal 3-byte corruption (IoU %.3f)", lo.IoU)
	}
}
