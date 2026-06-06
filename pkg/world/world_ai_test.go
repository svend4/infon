package world_test

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/world"
)

func TestApplyEffects(t *testing.T) {
	s := world.Init()
	if up := world.Apply(s, world.Directives{Rise: 1}); up.Relief <= s.Relief {
		t.Error("rise should raise the relief")
	}
	if dn := world.Apply(s, world.Directives{Fold: -1}); dn.Fold[0] >= s.Fold[0] {
		t.Error("fold -1 should lower the fold angle")
	}
	if c := world.Apply(s, world.Directives{Camera: 1}); c.Cam == s.Cam {
		t.Error("camera +1 should move the camera")
	}
	if sp := world.Apply(s, world.Directives{Spin: 1}); sp.Scene[0] == s.Scene[0] {
		t.Error("spin should move the orbit")
	}
}

func TestBrainControllerLocal(t *testing.T) {
	c := &world.BrainController{B: brain.Local{}}
	d := c.Decide(world.Init())
	for _, v := range []int{d.Fold, d.Rise, d.Spin, d.Camera} {
		if v < -1 || v > 1 {
			t.Fatalf("directive out of range: %+v", d)
		}
	}
	states, dirs := world.LoopC(c, world.Init(), 6)
	if len(states) != 6 || len(dirs) != 6 {
		t.Fatalf("loop returned %d states / %d dirs", len(states), len(dirs))
	}
}

func TestDirByte(t *testing.T) {
	d := world.Directives{Fold: 1, Rise: -1, Spin: 0, Camera: 1}
	if world.DirByte(d) == world.DirByte(world.Directives{}) {
		t.Error("distinct directives should pack distinctly")
	}
}
