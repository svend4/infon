package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestAvatarFace(t *testing.T) {
	// center, right, left, top keypoints
	pts := [][2]float32{{0.5, 0.5}, {1, 0.5}, {0, 0.5}, {0.5, 0}}
	head := raytrace.Vec3{Y: 1.6}
	objs := AvatarFace(Pose{Pos: head, Yaw: 0}, pts, raytrace.Vec3{X: 0.5, Y: 0.8, Z: 1})
	if len(objs) != 1+len(pts) {
		t.Fatalf("expected head + one sphere per keypoint = %d, got %d", 1+len(pts), len(objs))
	}
	at := func(i int) raytrace.Vec3 { return objs[i].(raytrace.Sphere).Center }
	// at yaw 0, right is +x: the x=1 keypoint sits right of the x=0 one
	if at(2).X <= at(3).X {
		t.Errorf("keypoint x=1 should be right of x=0 (%.2f vs %.2f)", at(2).X, at(3).X)
	}
	// the top keypoint (y=0) sits above the centre (y=0.5)
	if at(4).Y <= at(1).Y {
		t.Errorf("keypoint y=0 should be above y=0.5 (%.2f vs %.2f)", at(4).Y, at(1).Y)
	}
	// features are on the front (fwd = +z at yaw 0), ahead of the head plane
	if at(1).Z <= head.Z {
		t.Errorf("features should be in front of the head (z %.2f > %.2f)", at(1).Z, head.Z)
	}
}
