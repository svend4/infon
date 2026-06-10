package raydir

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// Pose is a participant's placement in the shared world: where they stand and
// which way they look. In the shared-world experience it is the ONLY thing sent
// over the network — the world itself is derived identically on both sides from a
// common script, so peers exchange meaning (a 40-byte pose), never geometry or
// pixels.
type Pose struct {
	Pos        raytrace.Vec3
	Yaw, Pitch float64
}

// PoseOf reads the pose of a fly camera.
func PoseOf(c FlyCam) Pose { return Pose{Pos: c.Pos, Yaw: c.Yaw, Pitch: c.Pitch} }

// Encode packs the pose into 40 bytes (five big-endian float64s).
func (p Pose) Encode() []byte {
	b := make([]byte, 40)
	put := func(i int, v float64) { binary.BigEndian.PutUint64(b[i:], math.Float64bits(v)) }
	put(0, p.Pos.X)
	put(8, p.Pos.Y)
	put(16, p.Pos.Z)
	put(24, p.Yaw)
	put(32, p.Pitch)
	return b
}

// DecodePose unpacks a pose produced by Encode.
func DecodePose(b []byte) (Pose, error) {
	if len(b) < 40 {
		return Pose{}, errors.New("raydir: pose payload too short")
	}
	g := func(i int) float64 { return math.Float64frombits(binary.BigEndian.Uint64(b[i:])) }
	return Pose{Pos: raytrace.Vec3{X: g(0), Y: g(8), Z: g(16)}, Yaw: g(24), Pitch: g(32)}, nil
}

// PoseSet maps participant IDs to their poses. In a group, the hub (host)
// broadcasts the whole set so everyone sees everyone; a guest sends a one-entry
// set (itself). Encoding is count(2) + N×(id(4) + 40-byte pose).
type PoseSet map[uint32]Pose

// Encode serialises the pose set.
func (s PoseSet) Encode() []byte {
	b := make([]byte, 2, 2+len(s)*44)
	binary.BigEndian.PutUint16(b[0:], uint16(len(s)))
	for id, p := range s {
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], id)
		b = append(b, hdr[:]...)
		b = append(b, p.Encode()...)
	}
	return b
}

// DecodePoseSet parses a pose set produced by Encode.
func DecodePoseSet(b []byte) (PoseSet, error) {
	if len(b) < 2 {
		return nil, errors.New("raydir: pose set too short")
	}
	n := int(binary.BigEndian.Uint16(b[0:]))
	off := 2
	s := make(PoseSet, n)
	for i := 0; i < n; i++ {
		if off+44 > len(b) {
			return nil, errors.New("raydir: truncated pose set")
		}
		id := binary.BigEndian.Uint32(b[off:])
		p, err := DecodePose(b[off+4 : off+44])
		if err != nil {
			return nil, err
		}
		s[id] = p
		off += 44
	}
	return s, nil
}

// avatarPalette gives participants distinct, bright colours.
var avatarPalette = []raytrace.Vec3{
	{X: 0.25, Y: 0.85, Z: 1.0}, {X: 1.0, Y: 0.5, Z: 0.2}, {X: 0.55, Y: 1.0, Z: 0.3},
	{X: 1.0, Y: 0.35, Z: 0.7}, {X: 0.8, Y: 0.7, Z: 1.0}, {X: 1.0, Y: 0.9, Z: 0.3},
}

// AvatarColor maps a participant ID to a distinct avatar colour.
func AvatarColor(id uint32) raytrace.Vec3 { return avatarPalette[id%uint32(len(avatarPalette))] }

// AvatarSpheres depicts a remote participant at the pose: a glowing head plus a
// small marker ahead of it showing which way they face.
func AvatarSpheres(p Pose, colour raytrace.Vec3) []raytrace.Object {
	fwd := raytrace.Vec3{X: math.Sin(p.Yaw), Y: 0, Z: math.Cos(p.Yaw)}
	return []raytrace.Object{
		raytrace.Sphere{Center: p.Pos, Radius: 0.45, Mat: raytrace.Material{Color: colour, Emit: colour.Scale(2)}},
		raytrace.Sphere{Center: p.Pos.Add(fwd.Scale(0.55)), Radius: 0.15, Mat: raytrace.Material{Color: colour, Emit: colour}},
	}
}
