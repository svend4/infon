package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestPoseRoundTrip(t *testing.T) {
	p := Pose{Pos: raytrace.Vec3{X: 1.5, Y: -2.25, Z: 7}, Yaw: 0.9, Pitch: -0.3}
	got, err := DecodePose(p.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("round trip = %+v, want %+v", got, p)
	}
	if len(p.Encode()) != 40 {
		t.Errorf("pose should encode to 40 bytes, got %d", len(p.Encode()))
	}
	if _, err := DecodePose([]byte{1, 2, 3}); err == nil {
		t.Error("short payload should error")
	}
}

func TestPoseOf(t *testing.T) {
	c := FlyCam{Pos: raytrace.Vec3{X: 3, Y: 2, Z: 1}, Yaw: 0.4, Pitch: 0.1}
	if p := PoseOf(c); p.Pos != c.Pos || p.Yaw != c.Yaw || p.Pitch != c.Pitch {
		t.Errorf("PoseOf = %+v, want camera fields", p)
	}
}

// The avatar's head sits at the pose and its gaze marker is ahead along the yaw.
func TestAvatarSpheres(t *testing.T) {
	p := Pose{Pos: raytrace.Vec3{X: 0, Y: 2, Z: 5}, Yaw: 0} // yaw 0 -> faces +Z
	objs := AvatarSpheres(p, raytrace.Vec3{X: 0.2, Y: 0.6, Z: 1})
	if len(objs) < 2 {
		t.Fatalf("avatar should be at least 2 objects, got %d", len(objs))
	}
	head := objs[0].(raytrace.Sphere)
	nose := objs[1].(raytrace.Sphere)
	if head.Center != p.Pos {
		t.Errorf("head at %v, want %v", head.Center, p.Pos)
	}
	if head.Mat.Emit.LenSq() == 0 {
		t.Error("avatar should glow so it is visible")
	}
	if nose.Center.Z <= p.Pos.Z || math.Abs(nose.Center.X-p.Pos.X) > 1e-9 {
		t.Errorf("gaze marker should be ahead along +Z, at %v", nose.Center)
	}
}
