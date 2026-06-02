package dtn

import "testing"

func TestScheduleDeliversAndExpires(t *testing.T) {
	// 10 bundles (born 0..9); two passes; ttl 5; 3 per pass.
	rep := Simulate(10, []Contact{{Start: 2, End: 3}, {Start: 8, End: 9}}, 5, 3)
	if rep.Delivered+rep.Expired != rep.Total {
		t.Fatalf("delivered %d + expired %d != total %d", rep.Delivered, rep.Expired, rep.Total)
	}
	// pass@2 takes 0,1,2 ; pass@8 takes 3,4,5 ; 6,7,8,9 never make it.
	if rep.Delivered != 6 || rep.Expired != 4 {
		t.Fatalf("delivered %d / expired %d, want 6 / 4", rep.Delivered, rep.Expired)
	}
	for _, i := range []int{0, 1, 2, 3, 4, 5} {
		if !rep.Mask[i] {
			t.Fatalf("bundle %d should be delivered", i)
		}
	}
	for _, i := range []int{6, 7, 8, 9} {
		if rep.Mask[i] {
			t.Fatalf("bundle %d should have expired", i)
		}
	}
}

func TestTTLExpiry(t *testing.T) {
	// one late pass; a tight TTL expires the early bundles before it arrives.
	rep := Simulate(6, []Contact{{Start: 20, End: 21}}, 3, 100)
	// only bundles born within ttl of the pass (born >= 20-3 = 17) survive; none
	// are born that late (max born 5), so all expire.
	if rep.Delivered != 0 || rep.Expired != 6 {
		t.Fatalf("delivered %d / expired %d, want 0 / 6 (all too old)", rep.Delivered, rep.Expired)
	}
}

func TestCapacityLimitsPerPass(t *testing.T) {
	// one generous-TTL pass with capacity 2 delivers only the 2 oldest.
	rep := Simulate(5, []Contact{{Start: 4, End: 5}}, 100, 2)
	if rep.Delivered != 2 {
		t.Fatalf("delivered %d, want 2 (per-pass capacity)", rep.Delivered)
	}
	if !rep.Mask[0] || !rep.Mask[1] || rep.Mask[2] {
		t.Fatal("the two oldest bundles should be the ones delivered")
	}
}
