package semvideo

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/glyphqr"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semframe"
)

// A Packet is one frame on a real (lossy) transport: a source frame index, a
// kind, and an RS-protected payload. Wire/ParseWire give the bytes to send.
type Packet struct {
	Idx     uint16
	Kind    byte
	Payload []byte // RS-wrapped JSON
}

func (p Packet) Wire() []byte {
	return append([]byte{byte(p.Idx >> 8), byte(p.Idx), p.Kind}, p.Payload...)
}

func ParseWire(b []byte) (Packet, bool) {
	if len(b) < 3 {
		return Packet{}, false
	}
	return Packet{Idx: uint16(b[0])<<8 | uint16(b[1]), Kind: b[2], Payload: b[3:]}, true
}

// EncodeFrames packetizes a marks-spec sequence with a Group-Of-Pictures: a
// keyframe every gop frames (gop<=0 = only the first), P-frame deltas in
// between, each RS-protected. Periodic keyframes bound desync after packet loss.
func EncodeFrames(specs []pseudo.Spec, ecc, gop int) []Packet {
	var out []Packet
	var prev pseudo.Spec
	have := false
	for i, s := range specs {
		if s.Marks == nil {
			continue
		}
		key := !have || (gop > 0 && i%gop == 0)
		if !key {
			if d, ok := semframe.DiffMarks(prev, s); ok {
				b, _ := json.Marshal(d)
				out = append(out, Packet{uint16(i), PFrame, glyphqr.RSEncodeBytes(b, ecc)})
				prev = s
				continue
			}
			// shape changed -> fall through and emit a keyframe
		}
		b, _ := json.Marshal(s)
		out = append(out, Packet{uint16(i), KeyFrame, glyphqr.RSEncodeBytes(b, ecc)})
		prev, have = s, true
	}
	return out
}

// Receiver reconstructs a frame timeline from arriving packets. It FREEZES (keeps
// the last good frame) on loss or an undecodable keyframe, and RE-SYNCS at the
// next decodable keyframe — exactly how lossy video survives.
type Receiver struct {
	ecc      int
	cur      pseudo.Spec
	last     pseudo.Spec
	synced   bool
	haveLast bool
}

func NewReceiver(ecc int) *Receiver { return &Receiver{ecc: ecc} }

// MarkLoss tells the receiver a packet was dropped: it desyncs (ignores deltas)
// and freezes until the next keyframe.
func (r *Receiver) MarkLoss() { r.synced = false }

// Apply ingests one arrived packet (RS-correcting it). Returns true if the frame
// advanced cleanly (a usable keyframe or applicable delta).
func (r *Receiver) Apply(p Packet) bool {
	raw, err := glyphqr.RSDecodeBytes(p.Payload, r.ecc)
	if err != nil {
		if p.Kind == KeyFrame {
			r.synced = false
		}
		return false
	}
	switch p.Kind {
	case KeyFrame:
		var s pseudo.Spec
		if json.Unmarshal(raw, &s) != nil || s.Marks == nil {
			r.synced = false
			return false
		}
		r.cur, r.last, r.synced, r.haveLast = s, s, true, true
		return true
	case PFrame:
		if !r.synced {
			return false
		}
		var d semframe.MarksDelta
		if json.Unmarshal(raw, &d) != nil {
			return false
		}
		r.cur = semframe.ApplyMarks(r.cur, d)
		r.last, r.haveLast = r.cur, true
		return true
	}
	return false
}

// Frame returns the current displayable frame (the last good one) and whether
// any frame has been received yet.
func (r *Receiver) Frame() (pseudo.Spec, bool) { return r.last, r.haveLast }

// MarksIoU is the semantic distortion metric: the sub-cell coverage
// intersection-over-union between two marks specs of the same shape (1.0 =
// identical meaning, 0.0 = disjoint). It measures how much MEANING survived,
// not how many pixels matched.
func MarksIoU(a, b pseudo.Spec) float64 {
	if a.Marks == nil || b.Marks == nil {
		return 0
	}
	ar, br := a.Marks.Rows, b.Marks.Rows
	if len(ar) != len(br) || len(ar) == 0 {
		return 0
	}
	S := glyphset.Sub
	inter, uni := 0, 0
	for y := range ar {
		if len(ar[y]) != len(br[y]) {
			return 0
		}
		for x := range ar[y] {
			for sy := 0; sy < S; sy++ {
				for sx := 0; sx < S; sx++ {
					pa := ar[y][x] != "" && glyphset.Coverage(ar[y][x], sx, sy, S)
					pb := br[y][x] != "" && glyphset.Coverage(br[y][x], sx, sy, S)
					if pa && pb {
						inter++
					}
					if pa || pb {
						uni++
					}
				}
			}
		}
	}
	if uni == 0 {
		return 1
	}
	return float64(inter) / float64(uni)
}
