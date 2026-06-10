package raydir

import (
	"encoding/binary"
	"errors"
	"sync"
)

// chatsync.go makes group chat reliable over a lossy UDP relay without a server
// of record. Each message carries a globally-unique ID — the origin's id in the
// high 32 bits, a per-origin sequence in the low 32 — so every peer can dedup
// re-broadcasts and a dropped message can simply be re-sent. The hub keeps a
// recent ring of every message it has seen and re-broadcasts it; a guest keeps a
// ring of its own messages and re-sends them to the hub. Receivers dedup by ID,
// so retransmission is free of double-display: eventual delivery, no duplicates.

// ChatMsg is one reliable chat message.
type ChatMsg struct {
	ID     uint64 // origin id (high 32) | per-origin sequence (low 32)
	Sender string // display name
	Text   string
}

// Origin returns the id of the peer that first composed the message.
func (m ChatMsg) Origin() uint32 { return uint32(m.ID >> 32) }

const (
	chatMaxSender = 255
	chatMaxText   = 1024
	chatMaxBatch  = 64
)

var errChatMsg = errors.New("raydir: malformed chat batch")

// EncodeChatMsgs encodes a batch of messages: [count u16] then per message
// [id u64][senderLen u16][sender][textLen u16][text]. Over-long fields are
// truncated so a single bad message can't blow the packet budget.
func EncodeChatMsgs(msgs []ChatMsg) []byte {
	if len(msgs) > chatMaxBatch {
		msgs = msgs[:chatMaxBatch]
	}
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(msgs)))
	var tmp [8]byte
	for _, m := range msgs {
		s := m.Sender
		if len(s) > chatMaxSender {
			s = s[:chatMaxSender]
		}
		t := m.Text
		if len(t) > chatMaxText {
			t = t[:chatMaxText]
		}
		binary.BigEndian.PutUint64(tmp[:], m.ID)
		buf = append(buf, tmp[:]...)
		binary.BigEndian.PutUint16(tmp[:2], uint16(len(s)))
		buf = append(buf, tmp[0], tmp[1])
		buf = append(buf, s...)
		binary.BigEndian.PutUint16(tmp[:2], uint16(len(t)))
		buf = append(buf, tmp[0], tmp[1])
		buf = append(buf, t...)
	}
	return buf
}

// DecodeChatMsgs parses a batch produced by EncodeChatMsgs.
func DecodeChatMsgs(data []byte) ([]ChatMsg, error) {
	if len(data) < 2 {
		return nil, errChatMsg
	}
	n := int(binary.BigEndian.Uint16(data[:2]))
	if n > chatMaxBatch {
		return nil, errChatMsg
	}
	off := 2
	out := make([]ChatMsg, 0, n)
	for i := 0; i < n; i++ {
		if off+10 > len(data) {
			return nil, errChatMsg
		}
		var m ChatMsg
		m.ID = binary.BigEndian.Uint64(data[off : off+8])
		off += 8
		sl := int(binary.BigEndian.Uint16(data[off : off+2]))
		off += 2
		if sl > chatMaxSender || off+sl+2 > len(data) {
			return nil, errChatMsg
		}
		m.Sender = string(data[off : off+sl])
		off += sl
		tl := int(binary.BigEndian.Uint16(data[off : off+2]))
		off += 2
		if tl > chatMaxText || off+tl > len(data) {
			return nil, errChatMsg
		}
		m.Text = string(data[off : off+tl])
		off += tl
		out = append(out, m)
	}
	return out, nil
}

// ChatSync assigns message IDs, dedups incoming messages, and keeps a bounded
// ring of messages to re-broadcast for reliable delivery. It is safe for
// concurrent use (the receive goroutine and the main loop both touch it).
type ChatSync struct {
	origin    uint32
	keepObs   bool // hub: also re-broadcast messages received from others
	mu        sync.Mutex
	seq       uint32
	ring      []ChatMsg
	ringMax   int
	seen      map[uint64]struct{}
	seenOrder []uint64
	seenMax   int
}

// NewChatSync returns a sync for the given origin id. keepObserved should be true
// on the hub, so it re-broadcasts every message it relays (full delivery to all
// peers); a guest sets it false and re-sends only its own messages to the hub.
func NewChatSync(origin uint32, keepObserved bool) *ChatSync {
	return &ChatSync{
		origin: origin, keepObs: keepObserved,
		ringMax: 16, seenMax: 1024, seen: map[uint64]struct{}{},
	}
}

// Compose stamps a new outbound message with the next unique ID, records it for
// dedup and re-broadcast, and returns it.
func (c *ChatSync) Compose(sender, text string) ChatMsg {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	m := ChatMsg{ID: uint64(c.origin)<<32 | uint64(c.seq), Sender: sender, Text: text}
	c.markSeen(m.ID)
	c.pushRing(m)
	return m
}

// Observe records an incoming message and reports whether it is new (first time
// seen). Duplicates — including this peer's own re-broadcasts echoed back — return
// false so they are neither displayed nor relayed twice. On the hub, new messages
// also enter the re-broadcast ring.
func (c *ChatSync) Observe(m ChatMsg) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.seen[m.ID]; ok {
		return false
	}
	c.markSeen(m.ID)
	if c.keepObs {
		c.pushRing(m)
	}
	return true
}

// Recent returns a copy of the re-broadcast ring (most recent first dropped past
// the cap), for periodic retransmission.
func (c *ChatSync) Recent() []ChatMsg {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ChatMsg(nil), c.ring...)
}

func (c *ChatSync) pushRing(m ChatMsg) {
	c.ring = append(c.ring, m)
	if len(c.ring) > c.ringMax {
		c.ring = append([]ChatMsg(nil), c.ring[len(c.ring)-c.ringMax:]...)
	}
}

func (c *ChatSync) markSeen(id uint64) {
	c.seen[id] = struct{}{}
	c.seenOrder = append(c.seenOrder, id)
	if len(c.seenOrder) > c.seenMax {
		drop := c.seenOrder[0]
		c.seenOrder = c.seenOrder[1:]
		delete(c.seen, drop)
	}
}
