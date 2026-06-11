package raydir

import (
	"testing"

	"github.com/svend4/infon/internal/avatar"
	"github.com/svend4/infon/pkg/raytrace"
)

func samplePresence(id uint32, seed int) Presence {
	return Presence{
		ID:   id,
		Pose: Pose{Pos: raytrace.Vec3{X: float64(id), Y: 1.6, Z: 3}, Yaw: 0.5, Pitch: -0.1},
		Face: avatar.Keypoints{Frame: 1, Points: DemoFace(seed)},
	}
}

func TestPresenceRoundTrip(t *testing.T) {
	p := samplePresence(7, 2)
	got, err := DecodePresence(p.Encode())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != p.ID || got.Pose != p.Pose {
		t.Errorf("id/pose mismatch: %+v vs %+v", got.Pose, p.Pose)
	}
	if len(got.Face.Points) != len(p.Face.Points) {
		t.Fatalf("face point count: %d vs %d", len(got.Face.Points), len(p.Face.Points))
	}
}

func TestPresenceSetRoundTrip(t *testing.T) {
	in := PresenceSet{}
	for i := uint32(0); i < 4; i++ {
		in[i] = samplePresence(i, int(i))
	}
	out, err := DecodePresenceSet(in.Encode())
	if err != nil {
		t.Fatalf("decode set: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("set size: %d vs %d", len(out), len(in))
	}
	for id, p := range in {
		g, ok := out[id]
		if !ok {
			t.Fatalf("missing id %d", id)
		}
		if g.Pose != p.Pose || len(g.Face.Points) != len(p.Face.Points) {
			t.Errorf("id %d mismatch", id)
		}
	}
}

func TestPresenceSetObjectsSkipsSelf(t *testing.T) {
	s := PresenceSet{}
	for i := uint32(0); i < 3; i++ {
		s[i] = samplePresence(i, int(i))
	}
	all := s.Objects(99) // skip nobody real
	mine := s.Objects(1) // skip id 1
	if len(all) == 0 {
		t.Fatal("a populated set should render faces")
	}
	if len(mine) >= len(all) {
		t.Errorf("skipping a participant should drop their face objects: %d vs %d", len(mine), len(all))
	}
}

func TestPresenceWireIsSmall(t *testing.T) {
	// presence must be a trickle: a pose plus a face, well under a UDP packet.
	if n := len(samplePresence(1, 1).Encode()); n > 600 {
		t.Errorf("presence too large for a frame: %d bytes", n)
	}
}
