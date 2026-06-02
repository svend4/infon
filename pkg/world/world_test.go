package world_test

import (
	"testing"

	"github.com/svend4/infon/pkg/scene3d"
	"github.com/svend4/infon/pkg/world"
)

func TestStateRoundTrip(t *testing.T) {
	if world.StateBytes != 51 {
		t.Fatalf("StateBytes=%d, want 51", world.StateBytes)
	}
	s := world.State{Relief: 120, Cam: 42}
	s.Fold[0], s.Fold[6] = 10, 200
	s.Scene[0] = scene3d.Item{Piece: 1, X: 2, Y: 3, Z: 4, Yaw: 5, Color: 6}
	b := s.Bytes()
	if len(b) != world.StateBytes {
		t.Fatalf("packed %d bytes, want %d", len(b), world.StateBytes)
	}
	if world.FromBytes(b) != s {
		t.Error("round trip changed the state")
	}
}

func TestBrainLoop(t *testing.T) {
	states := world.Loop(world.RefBrain{}, world.State{}, 4)
	if len(states) != 4 {
		t.Fatalf("got %d states, want 4", len(states))
	}
	if states[0].Cam != 5 || states[3].Cam != 20 {
		t.Errorf("phase did not advance: cams %d..%d", states[0].Cam, states[3].Cam)
	}
	if states[0] == states[1] {
		t.Error("consecutive states should differ")
	}
}

func TestRender(t *testing.T) {
	s := world.RefBrain{}.Next(world.State{Cam: 30})
	if s.Render(200, 200) == nil {
		t.Fatal("nil render")
	}
}
