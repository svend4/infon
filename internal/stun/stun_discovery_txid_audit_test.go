package stun

import (
	"encoding/binary"
	"testing"
	"time"
)

// RFC 5389 §6: the transaction ID is 96 bits chosen uniformly at random. It
// must not be a timestamp with a constant zero tail.
func TestTURNGenerateTransactionID_RandomAudit(t *testing.T) {
	tc := NewTURNClient("turn.example.com:3478", "user", "pass")

	const n = 16
	seen := make(map[[12]byte]bool, n)
	tailNonZero := false
	timestampLike := 0
	for i := 0; i < n; i++ {
		var id [12]byte
		tc.generateTransactionID(&id)
		if seen[id] {
			t.Fatalf("duplicate transaction ID %x", id)
		}
		seen[id] = true
		if id[8]|id[9]|id[10]|id[11] != 0 {
			tailNonZero = true
		}
		head := int64(binary.BigEndian.Uint64(id[:8]))
		if d := time.Since(time.Unix(0, head)); d > -time.Hour && d < time.Hour {
			timestampLike++
		}
	}
	if !tailNonZero {
		t.Fatalf("last 4 bytes of all %d transaction IDs are zero", n)
	}
	if timestampLike == n {
		t.Fatalf("all %d transaction IDs start with the current UnixNano timestamp", n)
	}
}
