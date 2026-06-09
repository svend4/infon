package raytrace

import "testing"

func TestBroadcastSpectatorFreezesOnLoss(t *testing.T) {
	const ecc = 10
	a := movingAnim() // 8 frames, sphere 0 orbits
	packets := EncodeBroadcast(a, ecc)
	if len(packets) != len(a.Frames) {
		t.Fatalf("packets = %d, want %d", len(packets), len(a.Frames))
	}

	spec := NewSpectator(ecc)
	// Tick 0: good.
	w0, live0 := spec.Recv(packets[0])
	if !live0 {
		t.Fatal("first packet should decode")
	}
	// Tick 1: lost -> freeze on tick 0's scene.
	wMiss, liveMiss := spec.Miss()
	if liveMiss {
		t.Error("a missed tick must not be live")
	}
	if wMiss.Spheres[0].Center.Sub(w0.Spheres[0].Center).Len() > 1e-9 {
		t.Error("a missed tick should freeze on the last good scene")
	}
	// Tick 2: good again -> updates and differs from tick 0 (the sphere moved).
	w2, live2 := spec.Recv(packets[2])
	if !live2 {
		t.Fatal("packet 2 should decode")
	}
	if w2.Spheres[0].Center.Sub(w0.Spheres[0].Center).Len() < 1e-3 {
		t.Error("scene should have advanced by tick 2")
	}
}

func TestBroadcastPacketIsIndependentlyDecodable(t *testing.T) {
	const ecc = 10
	a := movingAnim()
	packets := EncodeBroadcast(a, ecc)
	// Any single packet decodes on its own (no dependence on earlier packets).
	spec := NewSpectator(ecc)
	if _, live := spec.Recv(packets[5]); !live {
		t.Error("a mid-stream packet must decode independently")
	}
	// A corrupt-beyond-budget packet freezes (returns not-live), never panics.
	bad := append([]byte(nil), packets[6]...)
	for i := range bad {
		bad[i] ^= 0xFF
	}
	if _, live := spec.Recv(bad); live {
		t.Error("a destroyed packet should not be live")
	}
}
