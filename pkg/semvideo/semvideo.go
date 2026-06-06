// Package semvideo closes the loop between TVCP's video path and the symbolic
// substrate: it sends a "video call" not as H.264 frames but as a stream of
// compact pseudo-image (marks) specs — a keyframe followed by semframe P-frame
// deltas — each wrapped in Reed-Solomon parity (pkg/glyphqr). The result is a
// few hundred bytes per second of MEANING that survives packet loss, the
// audit's "video in bytes / semantic codec" made real.
package semvideo

import (
	"encoding/json"
	"errors"

	"github.com/svend4/infon/pkg/glyphqr"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semframe"
)

// Frame kinds on the wire.
const (
	KeyFrame = byte('K')
	PFrame   = byte('D')
)

func wrap(kind byte, raw []byte, ecc int) []byte {
	return append([]byte{kind}, glyphqr.RSEncodeBytes(raw, ecc)...)
}

// Encode turns a sequence of marks specs into RS-protected wire packets: the
// first is a keyframe (full spec); each later frame is a semframe delta against
// the previous one, or a fresh keyframe if the grid shape changed.
func Encode(specs []pseudo.Spec, ecc int) ([][]byte, error) {
	var out [][]byte
	var prev pseudo.Spec
	have := false
	for _, s := range specs {
		if s.Marks == nil {
			return nil, errors.New("semvideo: only marks specs are supported")
		}
		if !have {
			b, _ := json.Marshal(s)
			out = append(out, wrap(KeyFrame, b, ecc))
			prev, have = s, true
			continue
		}
		if d, ok := semframe.DiffMarks(prev, s); ok {
			b, _ := json.Marshal(d)
			out = append(out, wrap(PFrame, b, ecc))
		} else {
			b, _ := json.Marshal(s)
			out = append(out, wrap(KeyFrame, b, ecc))
		}
		prev = s
	}
	return out, nil
}

// Decode reconstructs the spec sequence, RS-correcting every packet and applying
// deltas to the running frame.
func Decode(packets [][]byte, ecc int) ([]pseudo.Spec, error) {
	var out []pseudo.Spec
	var cur pseudo.Spec
	have := false
	for _, p := range packets {
		if len(p) < 1 {
			return nil, errors.New("semvideo: empty packet")
		}
		raw, err := glyphqr.RSDecodeBytes(p[1:], ecc)
		if err != nil {
			return nil, err
		}
		switch p[0] {
		case KeyFrame:
			var s pseudo.Spec
			if json.Unmarshal(raw, &s) != nil || s.Marks == nil {
				return nil, errors.New("semvideo: bad keyframe")
			}
			cur, have = s, true
		case PFrame:
			if !have {
				return nil, errors.New("semvideo: P-frame before keyframe")
			}
			var d semframe.MarksDelta
			if json.Unmarshal(raw, &d) != nil {
				return nil, errors.New("semvideo: bad delta")
			}
			cur = semframe.ApplyMarks(cur, d)
		default:
			return nil, errors.New("semvideo: unknown frame kind")
		}
		out = append(out, cur)
	}
	return out, nil
}

// Bytes is the total wire size of a packet stream.
func Bytes(packets [][]byte) int {
	n := 0
	for _, p := range packets {
		n += len(p)
	}
	return n
}
