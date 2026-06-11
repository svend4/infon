package raydir

import (
	"encoding/binary"
	"errors"

	"github.com/svend4/infon/internal/avatar"
	"github.com/svend4/infon/pkg/raytrace"
)

// presence.go is what a participant puts on the wire each frame to BE somewhere in a
// shared world: who they are, where they stand and look (a 40-byte pose), and their
// face (compact keypoints). The world itself is rebuilt locally and identically by
// every peer from a common script, so this presence — meaning, not pixels — is all
// that crosses the network. It joins the pose wire format with the avatar face
// keypoints so a real call can be rendered INSIDE the world, networked.

// Presence is one participant's networked self: id, pose and face keypoints.
type Presence struct {
	ID   uint32
	Pose Pose
	Face avatar.Keypoints
}

// Encode packs a presence: id(4) + pose(40) + faceLen(2) + face. The face uses the
// avatar package's compact keypoint frame (~282 B for 68 points, resolution-free).
func (p Presence) Encode() []byte {
	face := p.Face.Encode()
	b := make([]byte, 0, 4+40+2+len(face))
	var id [4]byte
	binary.BigEndian.PutUint32(id[:], p.ID)
	b = append(b, id[:]...)
	b = append(b, p.Pose.Encode()...)
	var fl [2]byte
	binary.BigEndian.PutUint16(fl[:], uint16(len(face)))
	b = append(b, fl[:]...)
	b = append(b, face...)
	return b
}

// decodePresenceAt decodes one presence at offset off, returning it and the offset
// just past it.
func decodePresenceAt(b []byte, off int) (Presence, int, error) {
	if off+46 > len(b) {
		return Presence{}, off, errors.New("raydir: presence too short")
	}
	id := binary.BigEndian.Uint32(b[off:])
	pose, err := DecodePose(b[off+4 : off+44])
	if err != nil {
		return Presence{}, off, err
	}
	fl := int(binary.BigEndian.Uint16(b[off+44:]))
	start := off + 46
	if start+fl > len(b) {
		return Presence{}, off, errors.New("raydir: truncated presence face")
	}
	face, err := avatar.DecodeKeypoints(b[start : start+fl])
	if err != nil {
		return Presence{}, off, err
	}
	return Presence{ID: id, Pose: pose, Face: face}, start + fl, nil
}

// DecodePresence parses a single presence produced by Encode.
func DecodePresence(b []byte) (Presence, error) {
	p, _, err := decodePresenceAt(b, 0)
	return p, err
}

// PresenceSet maps participant IDs to their presence. The hub broadcasts the whole
// set so everyone sees everyone; a guest sends a one-entry set (itself). Encoding is
// count(2) followed by each presence (self-delimiting via its face length).
type PresenceSet map[uint32]Presence

// Encode serialises the presence set.
func (s PresenceSet) Encode() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b[0:], uint16(len(s)))
	for _, p := range s {
		b = append(b, p.Encode()...)
	}
	return b
}

// DecodePresenceSet parses a presence set produced by Encode.
func DecodePresenceSet(b []byte) (PresenceSet, error) {
	if len(b) < 2 {
		return nil, errors.New("raydir: presence set too short")
	}
	n := int(binary.BigEndian.Uint16(b[0:]))
	off := 2
	s := make(PresenceSet, n)
	for i := 0; i < n; i++ {
		p, next, err := decodePresenceAt(b, off)
		if err != nil {
			return nil, err
		}
		s[p.ID] = p
		off = next
	}
	return s, nil
}

// Objects renders every participant in the set as an in-world face (raydir.AvatarFace)
// coloured by id, skipping the viewer's own id — the meeting drawn into the world.
func (s PresenceSet) Objects(skip uint32) []raytrace.Object {
	var out []raytrace.Object
	for id, p := range s {
		if id == skip {
			continue
		}
		out = append(out, AvatarFace(p.Pose, p.Face.Points, AvatarColor(id))...)
	}
	return out
}
