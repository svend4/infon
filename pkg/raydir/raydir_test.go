package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func initState() CamState {
	return CamState{Angle: 0, Radius: 6, Height: 2.4, Target: raytrace.Vec3{X: 0, Y: 1, Z: 0}}
}

func TestRefDirectorOrbitsAndStaysInBounds(t *testing.T) {
	s := initState()
	a0 := s.Angle
	s = RefDirector{}.Next(s)
	if math.Abs(s.Angle-(a0+dAngle)) > 1e-9 {
		t.Errorf("angle did not advance by dAngle: %v", s.Angle)
	}
	for i := 0; i < 200; i++ {
		s = RefDirector{}.Next(s)
		if s.Radius < rMin-1e-9 || s.Radius > rMax+1e-9 || s.Height < hMin-1e-9 || s.Height > hMax+1e-9 {
			t.Fatalf("state left bounds: radius=%v height=%v", s.Radius, s.Height)
		}
	}
}

func TestBrainDirectorWithLocalReference(t *testing.T) {
	// brain.Local is the built-in reference; for game:world it returns spin=1, so
	// the camera must orbit deterministically.
	d := &BrainDirector{B: brain.Local{}}
	s := initState()
	for i := 0; i < 10; i++ {
		s = d.Next(s)
	}
	if math.Abs(s.Angle-10*dAngle) > 1e-9 {
		t.Errorf("brain-flown angle = %v, want %v", s.Angle, 10*dAngle)
	}
	// Camera should look back toward the target (yaw points roughly to -Z side).
	cam := s.Camera()
	if cam.FOV <= 0 {
		t.Error("camera FOV not set")
	}
}

func TestFlyProducesCameras(t *testing.T) {
	cams := Fly(RefDirector{}, initState(), 12)
	if len(cams) != 12 {
		t.Fatalf("got %d cameras, want 12", len(cams))
	}
}
