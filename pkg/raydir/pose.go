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

// AvatarSpheres depicts a remote participant at the pose: a glowing head plus a
// small marker ahead of it showing which way they face.
func AvatarSpheres(p Pose, colour raytrace.Vec3) []raytrace.Object {
	fwd := raytrace.Vec3{X: math.Sin(p.Yaw), Y: 0, Z: math.Cos(p.Yaw)}
	return []raytrace.Object{
		raytrace.Sphere{Center: p.Pos, Radius: 0.45, Mat: raytrace.Material{Color: colour, Emit: colour.Scale(2)}},
		raytrace.Sphere{Center: p.Pos.Add(fwd.Scale(0.55)), Radius: 0.15, Mat: raytrace.Material{Color: colour, Emit: colour}},
	}
}
