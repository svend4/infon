package raydir

import (
	"fmt"
	"math/rand"
	"testing"
)

// TestChatReliableUnderLoss models the cmd/raymeet relay over a 35%-lossy channel
// — guest A → hub → guest B, with A re-sending its ring and the hub re-broadcasting
// its ring — and asserts every message A composes eventually reaches B exactly once
// (deterministic with a fixed seed). This is the end-to-end proof of the ChatSync
// reliability model (ids + dedup + retransmit) that the per-method unit tests only
// cover in pieces.
func TestChatReliableUnderLoss(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const lossP = 0.35
	drops := func() bool { return rng.Float64() < lossP }

	hub := NewChatSync(1, true)
	A := NewChatSync(0xA, false)
	B := NewChatSync(0xB, false)

	gotB := map[uint64]string{} // id -> text, recorded the first time B accepts it
	cntB := map[uint64]int{}    // how many times B accepted an id (must end at 1)

	deliverB := func(payload []byte) {
		if drops() {
			return
		}
		msgs, err := DecodeChatMsgs(payload)
		if err != nil {
			return
		}
		for _, m := range msgs {
			if B.Observe(m) {
				gotB[m.ID] = m.Text
				cntB[m.ID]++
			}
		}
	}
	deliverHub := func(payload []byte) {
		if drops() {
			return
		}
		msgs, err := DecodeChatMsgs(payload)
		if err != nil {
			return
		}
		for _, m := range msgs {
			if hub.Observe(m) { // fresh at the hub -> relay it onward to B (lossy)
				deliverB(EncodeChatMsgs([]ChatMsg{m}))
			}
		}
	}

	// A composes a burst; each initial send to the hub may be dropped.
	var sent []ChatMsg
	for i := 0; i < 12; i++ {
		m := A.Compose("A", fmt.Sprintf("m%d", i))
		sent = append(sent, m)
		deliverHub(EncodeChatMsgs([]ChatMsg{m}))
	}

	// Retransmit rounds: A re-sends its ring to the hub; the hub re-broadcasts its
	// ring to B. Loss on every hop. A few dozen rounds make delivery near-certain.
	for round := 0; round < 80; round++ {
		deliverHub(EncodeChatMsgs(A.Recent()))
		deliverB(EncodeChatMsgs(hub.Recent()))
	}

	for _, m := range sent {
		if gotB[m.ID] != m.Text {
			t.Errorf("message %q (id %x) never reached B", m.Text, m.ID)
		}
		if cntB[m.ID] != 1 {
			t.Errorf("message %q accepted %d times by B, want exactly 1 (dedup)", m.Text, cntB[m.ID])
		}
	}
}
