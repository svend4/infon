package semvideo

import (
	"encoding/json"
	"errors"

	"github.com/svend4/infon/pkg/glyphqr"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semframe"
)

// bframe.go adds bidirectionally-predicted "frames of meaning" (B-frames) to the
// stream. A B-frame is a delta from the NEARER of its two surrounding anchors —
// the past anchor or the future anchor — whichever needs fewer changed cells.
//
// As in MPEG, this splits coding order from display order: the closing anchor of
// each group of pictures (GOP) must be transmitted BEFORE the B-frames that may
// predict from it. BiEncode therefore returns a display-order permutation
// alongside the packets — the semantic analogue of DTS vs PTS.
//
// The point here is NOT smaller bytes (our P-frames are already distance-1
// minimal deltas) but RESILIENCE: B-frames are non-reference and disposable.
// Lose one and only that display frame is gone; the anchors and every other
// frame still decode. Dropping all B-frames yields a valid lower-frame-rate
// video — temporal scalability — whereas dropping a P-frame in the keyframe/
// P-frame stream corrupts everything after it.

// BFrame is the wire kind for a bidirectional frame; its packet header is
// {BFrame, dir} where dir selects the reference anchor.
const BFrame = byte('B')

const (
	fromPast   = byte('L') // predict from the previous (left) anchor
	fromFuture = byte('R') // predict from the next (right) anchor
)

func wrapHdr(hdr, raw []byte, ecc int) []byte {
	out := append([]byte{}, hdr...)
	return append(out, glyphqr.RSEncodeBytes(raw, ecc)...)
}

func sameShape(a, b pseudo.Spec) bool {
	if a.Marks == nil || b.Marks == nil {
		return false
	}
	if len(a.Marks.Rows) != len(b.Marks.Rows) {
		return false
	}
	for y := range a.Marks.Rows {
		if len(a.Marks.Rows[y]) != len(b.Marks.Rows[y]) {
			return false
		}
	}
	return true
}

// BiEncode encodes a constant-shape marks sequence as a B-frame GOP stream:
// anchors every gop frames (the first a keyframe, the rest P-frames against the
// previous anchor), and every frame strictly between two anchors as a B-frame
// predicted from the nearer anchor. It returns the RS-protected packets in CODING
// order plus display[i], the display index of packets[i], so BiDecode can restore
// display order. gop < 2 is treated as 2.
func BiEncode(specs []pseudo.Spec, gop, ecc int) (packets [][]byte, display []int, err error) {
	if gop < 2 {
		gop = 2
	}
	n := len(specs)
	if n == 0 {
		return nil, nil, nil
	}
	for i, s := range specs {
		if s.Marks == nil {
			return nil, nil, errors.New("semvideo: only marks specs are supported")
		}
		if i > 0 && !sameShape(specs[0], s) {
			return nil, nil, errors.New("semvideo: B-frames require a constant grid shape")
		}
	}

	var anchors []int
	for i := 0; i < n; i += gop {
		anchors = append(anchors, i)
	}
	if anchors[len(anchors)-1] != n-1 {
		anchors = append(anchors, n-1)
	}

	// first anchor -> keyframe
	kb, _ := json.Marshal(specs[anchors[0]])
	packets = append(packets, wrapHdr([]byte{KeyFrame}, kb, ecc))
	display = append(display, anchors[0])

	for k := 1; k < len(anchors); k++ {
		a, b := anchors[k-1], anchors[k]
		// closing anchor b as a P-frame against the previous anchor a
		d, ok := semframe.DiffMarks(specs[a], specs[b])
		if !ok {
			return nil, nil, errors.New("semvideo: anchor diff failed")
		}
		db, _ := json.Marshal(d)
		packets = append(packets, wrapHdr([]byte{PFrame}, db, ecc))
		display = append(display, b)
		// B-frames strictly between a and b, each from the nearer anchor
		for j := a + 1; j < b; j++ {
			dl, okl := semframe.DiffMarks(specs[a], specs[j])
			dr, okr := semframe.DiffMarks(specs[b], specs[j])
			if !okl && !okr {
				return nil, nil, errors.New("semvideo: B-frame diff failed")
			}
			dir, chosen := fromPast, dl
			if !okl || (okr && len(dr.Cells) < len(dl.Cells)) {
				dir, chosen = fromFuture, dr
			}
			bb, _ := json.Marshal(chosen)
			packets = append(packets, wrapHdr([]byte{BFrame, dir}, bb, ecc))
			display = append(display, j)
		}
	}
	return packets, display, nil
}

// BiDecode reconstructs the display-ordered spec sequence from a B-frame stream.
// packets are in coding order; display[i] is the display index of packets[i]. It
// tolerates a sparse stream: if some display indices are absent (dropped
// B-frames), the corresponding output slots are left zero and the rest still
// decode — pass a display slice whose max+1 equals the intended length.
func BiDecode(packets [][]byte, display []int, ecc int) ([]pseudo.Spec, error) {
	if len(packets) != len(display) {
		return nil, errors.New("semvideo: packets/display length mismatch")
	}
	maxIdx := -1
	for _, d := range display {
		if d > maxIdx {
			maxIdx = d
		}
	}
	out := make([]pseudo.Spec, maxIdx+1)
	var leftAnchor, rightAnchor pseudo.Spec
	haveKey := false
	for i, p := range packets {
		if len(p) < 1 {
			return nil, errors.New("semvideo: empty packet")
		}
		switch p[0] {
		case KeyFrame:
			raw, err := glyphqr.RSDecodeBytes(p[1:], ecc)
			if err != nil {
				return nil, err
			}
			var s pseudo.Spec
			if json.Unmarshal(raw, &s) != nil || s.Marks == nil {
				return nil, errors.New("semvideo: bad keyframe")
			}
			rightAnchor, haveKey = s, true
			out[display[i]] = s
		case PFrame:
			if !haveKey {
				return nil, errors.New("semvideo: P-frame before keyframe")
			}
			raw, err := glyphqr.RSDecodeBytes(p[1:], ecc)
			if err != nil {
				return nil, err
			}
			var d semframe.MarksDelta
			if json.Unmarshal(raw, &d) != nil {
				return nil, errors.New("semvideo: bad delta")
			}
			leftAnchor = rightAnchor
			rightAnchor = semframe.ApplyMarks(leftAnchor, d)
			out[display[i]] = rightAnchor
		case BFrame:
			if !haveKey || len(p) < 2 {
				return nil, errors.New("semvideo: bad B-frame")
			}
			dir := p[1]
			raw, err := glyphqr.RSDecodeBytes(p[2:], ecc)
			if err != nil {
				return nil, err
			}
			var d semframe.MarksDelta
			if json.Unmarshal(raw, &d) != nil {
				return nil, errors.New("semvideo: bad delta")
			}
			ref := leftAnchor
			if dir == fromFuture {
				ref = rightAnchor
			}
			out[display[i]] = semframe.ApplyMarks(ref, d)
		default:
			return nil, errors.New("semvideo: unknown frame kind")
		}
	}
	return out, nil
}

// DropBFrames returns the sub-stream with every B-frame removed (anchors only),
// keeping coding/display alignment. Decoding it yields a valid lower-frame-rate
// video — proof that B-frames are disposable.
func DropBFrames(packets [][]byte, display []int) ([][]byte, []int) {
	var p [][]byte
	var d []int
	for i := range packets {
		if len(packets[i]) > 0 && packets[i][0] == BFrame {
			continue
		}
		p = append(p, packets[i])
		d = append(d, display[i])
	}
	return p, d
}
