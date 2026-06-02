package arena

import "github.com/svend4/infon/pkg/glyphqr"

// broadcast.go turns an arena match into a loss-tolerant semantic broadcast: a
// stream of RS-protected packets a spectator watches over a lossy channel. Every
// gop-th tick is a full keyframe; the rest are deltas vs the current keyframe
// (self-contained within a GOP, so a lost delta costs only its own tick). This
// is the unification: the agent-world rides the same Reed-Solomon channel as a
// semantic video call - "watch the AI war over a few hundred bytes per second".
const (
	pktKey   = 0
	pktDelta = 1
)

func packetWire(kind byte, tick int, payload []byte, ecc int) []byte {
	return glyphqr.RSEncodeBytes(append([]byte{kind, byte(tick >> 8), byte(tick)}, payload...), ecc)
}

func parsePacket(wire []byte, ecc int) (kind byte, tick int, payload []byte, ok bool) {
	d, err := glyphqr.RSDecodeBytes(wire, ecc)
	if err != nil || len(d) < 3 {
		return 0, 0, nil, false
	}
	return d[0], int(d[1])<<8 | int(d[2]), d[3:], true
}

func diffVs(key, fr []byte) []byte {
	var out []byte
	for i := 0; i < len(fr) && i < len(key); i++ {
		if fr[i] != key[i] {
			out = append(out, byte(i>>8), byte(i), fr[i])
		}
	}
	return out
}

func applyDelta(base, delta []byte) []byte {
	out := make([]byte, len(base))
	copy(out, base)
	for p := 0; p+3 <= len(delta); p += 3 {
		if idx := int(delta[p])<<8 | int(delta[p+1]); idx < len(out) {
			out[idx] = delta[p+2]
		}
	}
	return out
}

// EncodeMatch turns equal-width snapshot frames into broadcast packet wires.
func EncodeMatch(frames [][]byte, gop, ecc int) [][]byte {
	var wires [][]byte
	var key []byte
	for t, fr := range frames {
		if gop < 1 || t%gop == 0 {
			key = fr
			wires = append(wires, packetWire(pktKey, t, fr, ecc))
		} else {
			wires = append(wires, packetWire(pktDelta, t, diffVs(key, fr), ecc))
		}
	}
	return wires
}

// Spectate reconstructs what a spectator sees: a nil wire is a lost packet and a
// corrupt-beyond-budget packet decodes as lost; on a loss the spectator freezes
// on the last good frame until the next keyframe. Returns the frames and a live
// flag per tick (true = a packet was applied, false = frozen).
func Spectate(wires [][]byte, ecc, width int) (frames [][]byte, live []bool) {
	var base, last []byte
	froze := func() []byte {
		if last != nil {
			c := make([]byte, len(last))
			copy(c, last)
			return c
		}
		return make([]byte, width)
	}
	for _, w := range wires {
		if w == nil {
			frames, live = append(frames, froze()), append(live, false)
			continue
		}
		kind, _, payload, ok := parsePacket(w, ecc)
		if !ok {
			frames, live = append(frames, froze()), append(live, false)
			continue
		}
		var fr []byte
		switch {
		case kind == pktKey:
			base, fr = payload, payload
		case base != nil:
			fr = applyDelta(base, payload)
		default:
			frames, live = append(frames, froze()), append(live, false)
			continue
		}
		last = fr
		frames, live = append(frames, fr), append(live, true)
	}
	return
}

// LoadSnapshot rebuilds the units from a Snapshot (keeping terrain), so a
// spectator arena can render a received frame.
func (a *Arena) LoadSnapshot(b []byte) {
	n := len(b) / UnitBytes
	a.Units = make([]Unit, n)
	for i := 0; i < n; i++ {
		o := i * UnitBytes
		a.Units[i] = Unit{Kind: b[o], X: b[o+1], Y: b[o+2], HP: b[o+3], Faction: b[o+4] & 1, Alive: b[o+4]&2 != 0}
	}
}

// Spectator is a streaming receiver for a live networked viewer: feed it the
// packet wires it receives (Recv) and call Miss for ticks it never got; it
// tracks the current frame, freezing on loss and resyncing at keyframes.
type Spectator struct {
	ecc, width int
	base, last []byte
}

// NewSpectator makes a streaming receiver for width-byte frames.
func NewSpectator(ecc, width int) *Spectator { return &Spectator{ecc: ecc, width: width} }

func (s *Spectator) froze() []byte {
	if s.last != nil {
		c := make([]byte, len(s.last))
		copy(c, s.last)
		return c
	}
	return make([]byte, s.width)
}

// Recv applies one received packet wire and returns the current frame (live =
// true if the packet decoded and applied).
func (s *Spectator) Recv(wire []byte) (frame []byte, live bool) {
	kind, _, payload, ok := parsePacket(wire, s.ecc)
	if !ok {
		return s.froze(), false
	}
	switch {
	case kind == pktKey:
		s.base, s.last = payload, payload
		return payload, true
	case s.base != nil:
		fr := applyDelta(s.base, payload)
		s.last = fr
		return fr, true
	default:
		return s.froze(), false
	}
}

// Miss records a lost tick and returns the frozen frame.
func (s *Spectator) Miss() (frame []byte, live bool) { return s.froze(), false }

// PacketTick reads the tick number from a wire (or -1 if undecodable), so a
// receiver can order packets and detect gaps.
func PacketTick(wire []byte, ecc int) int {
	_, t, _, ok := parsePacket(wire, ecc)
	if !ok {
		return -1
	}
	return t
}
