package raydir

import (
	"reflect"
	"testing"
)

// A batch of messages round-trips through encode/decode unchanged.
func TestChatMsgsRoundTrip(t *testing.T) {
	in := []ChatMsg{
		{ID: 0x1111_0000_0000_0001, Sender: "alice", Text: "hello"},
		{ID: 0x2222_0000_0000_0007, Sender: "bob", Text: "hi there 👋"},
		{ID: 3, Sender: "", Text: ""},
	}
	out, err := DecodeChatMsgs(EncodeChatMsgs(in))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch:\n in=%v\nout=%v", in, out)
	}
}

// Decoding rejects truncated or over-count batches instead of panicking.
func TestDecodeChatMsgsBad(t *testing.T) {
	if _, err := DecodeChatMsgs(nil); err == nil {
		t.Error("empty data should error")
	}
	if _, err := DecodeChatMsgs([]byte{0x00, 0x01}); err == nil {
		t.Error("count=1 with no body should error")
	}
	good := EncodeChatMsgs([]ChatMsg{{ID: 1, Sender: "x", Text: "yz"}})
	if _, err := DecodeChatMsgs(good[:len(good)-1]); err == nil {
		t.Error("truncated body should error")
	}
}

// Compose hands out unique, origin-tagged, monotonically increasing IDs.
func TestComposeUniqueIDs(t *testing.T) {
	c := NewChatSync(0xABCD, false)
	a := c.Compose("me", "one")
	b := c.Compose("me", "two")
	if a.ID == b.ID {
		t.Fatal("composed IDs must be unique")
	}
	if a.Origin() != 0xABCD || b.Origin() != 0xABCD {
		t.Fatalf("origin not tagged: %x %x", a.Origin(), b.Origin())
	}
	if b.ID <= a.ID {
		t.Fatal("sequence should increase")
	}
}

// Observe dedups: a message (and its re-broadcasts) is new exactly once.
func TestObserveDedup(t *testing.T) {
	c := NewChatSync(1, true)
	m := ChatMsg{ID: 42, Sender: "bob", Text: "hi"}
	if !c.Observe(m) {
		t.Fatal("first sighting should be new")
	}
	if c.Observe(m) {
		t.Fatal("second sighting (a re-broadcast) should be a duplicate")
	}
}

// A peer never re-observes its own composed message as new (so it isn't shown
// twice when the hub relays it back).
func TestComposeThenObserveSelf(t *testing.T) {
	c := NewChatSync(7, false)
	m := c.Compose("me", "echo")
	if c.Observe(m) {
		t.Fatal("a peer's own message echoed back must be a duplicate")
	}
}

// The hub re-broadcasts everything it observes; a guest re-broadcasts only its
// own composed messages.
func TestRecentRingRole(t *testing.T) {
	hub := NewChatSync(1, true)
	hub.Observe(ChatMsg{ID: 100, Sender: "g", Text: "from guest"})
	if got := hub.Recent(); len(got) != 1 || got[0].ID != 100 {
		t.Fatalf("hub should re-broadcast observed messages, got %v", got)
	}

	guest := NewChatSync(2, false)
	guest.Observe(ChatMsg{ID: 200, Sender: "h", Text: "from hub"})
	if got := guest.Recent(); len(got) != 0 {
		t.Fatalf("guest should not re-broadcast others' messages, got %v", got)
	}
	own := guest.Compose("me", "mine")
	if got := guest.Recent(); len(got) != 1 || got[0].ID != own.ID {
		t.Fatalf("guest should re-broadcast its own message, got %v", got)
	}
}

// The re-broadcast ring is bounded.
func TestRecentRingBounded(t *testing.T) {
	c := NewChatSync(1, true)
	for i := 0; i < 100; i++ {
		c.Observe(ChatMsg{ID: uint64(i + 1), Sender: "s", Text: "m"})
	}
	if got := len(c.Recent()); got != c.ringMax {
		t.Fatalf("ring should cap at %d, got %d", c.ringMax, got)
	}
}

// Dedup memory is bounded but still dedups far past the re-broadcast ring.
func TestSeenBounded(t *testing.T) {
	c := NewChatSync(1, false)
	c.seenMax = 8
	for i := 0; i < 100; i++ {
		c.Observe(ChatMsg{ID: uint64(i + 1), Sender: "s", Text: "m"})
	}
	c.mu.Lock()
	n := len(c.seen)
	c.mu.Unlock()
	if n > c.seenMax {
		t.Fatalf("seen set should be bounded to %d, got %d", c.seenMax, n)
	}
}
