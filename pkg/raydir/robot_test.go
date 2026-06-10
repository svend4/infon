package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestRobotPatrolsWaypoints(t *testing.T) {
	a := raytrace.Vec3{X: 0, Z: 0}
	b := raytrace.Vec3{X: 6, Z: 0}
	r := NewRobot(a, []raytrace.Vec3{b, a}, 0.2)
	nearB, backNearA := false, false
	for i := 0; i < 400; i++ {
		r.Step(0.05)
		if dx, dz := r.Pos.X-b.X, r.Pos.Z-b.Z; dx*dx+dz*dz < 0.5*0.5 {
			nearB = true
		}
		if nearB {
			if dx, dz := r.Pos.X-a.X, r.Pos.Z-a.Z; dx*dx+dz*dz < 0.5*0.5 {
				backNearA = true
			}
		}
	}
	if !nearB || !backNearA {
		t.Errorf("robot should reach waypoint B then loop back near A (nearB=%v back=%v, pos=%v)", nearB, backNearA, r.Pos)
	}
	if r.Pos.Y != 0.5 {
		t.Errorf("robot should ride on the ground, Y=%.2f", r.Pos.Y)
	}
}

func TestRobotBeaconReddensWithStatus(t *testing.T) {
	calm := robotStatusColor(0.1)
	fault := robotStatusColor(0.9)
	if fault.X <= calm.X || fault.Y >= calm.Y {
		t.Errorf("a worse status should be redder (more R, less G): calm=%v fault=%v", calm, fault)
	}
	// the beacon (third object) must emit, and brighter when worse
	emitOf := func(s float64) raytrace.Vec3 {
		objs := NewRobot(raytrace.Vec3{}, nil, s).Objects()
		return objs[len(objs)-1].(raytrace.Sphere).Mat.Emit
	}
	if emitOf(0.9).LenSq() <= emitOf(0.1).LenSq() {
		t.Error("a worse status should have a brighter beacon")
	}
}

func TestRobotGoToOverridesPatrol(t *testing.T) {
	r := NewRobot(raytrace.Vec3{}, nil, 0.2) // no patrol; goal-driven
	if !r.AtGoal() {
		t.Fatal("a robot with no command should report AtGoal")
	}
	dest := raytrace.Vec3{X: 6, Z: 0}
	r.GoTo(dest)
	if r.AtGoal() {
		t.Fatal("after GoTo the robot should have an active goal")
	}
	for i := 0; i < 200 && !r.AtGoal(); i++ {
		r.Step(0.05)
	}
	if !r.AtGoal() {
		t.Fatal("robot should reach and clear its commanded goal")
	}
	if dx, dz := r.Pos.X-dest.X, r.Pos.Z-dest.Z; dx*dx+dz*dz > 0.5*0.5 {
		t.Errorf("robot should stop near its goal, ended at %v", r.Pos)
	}
}

func TestRobotObjectsCount(t *testing.T) {
	objs := NewRobot(raytrace.Vec3{}, nil, 0.5).Objects()
	if len(objs) != 3 {
		t.Fatalf("a robot should be body+head+beacon = 3 objects, got %d", len(objs))
	}
}

func TestWorldRobotsRenderAndAnimate(t *testing.T) {
	w := NewWorld()
	base := len(w.Scene().Objects)
	if w.HasRobots() {
		t.Fatal("a fresh world has no robots")
	}
	w.SpawnRobot(NewRobot(raytrace.Vec3{X: 2, Z: 4}, []raytrace.Vec3{{X: 5, Z: 4}}, 0.8))
	if !w.HasRobots() {
		t.Error("HasRobots should be true after spawning")
	}
	if !w.HasAnimated() {
		t.Error("a world with a robot is animated (it moves)")
	}
	if got := len(w.Scene().Objects); got <= base {
		t.Errorf("SceneWith should include the robot's objects (%d <= %d)", got, base)
	}
	w.StepRobots(0.1) // must not panic and should move the robot toward its waypoint
	if len(w.Robots()) != 1 {
		t.Errorf("expected 1 robot, got %d", len(w.Robots()))
	}
}
