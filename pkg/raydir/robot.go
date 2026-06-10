package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// robot.go puts working robots into the shared world — the camp/logistics machines
// of Part 1 as first-class world entities, alongside the flock, sprites and flyer.
// A robot patrols a loop of waypoints (a delivery round) carrying a status that
// colours its beacon green (calm) to red (fault), so the same world the walker
// explores (rayexplore) or shares (raymeet) is also a yard of moving machines.
// Local and deterministic — offline and tested.

// Robot is a machine that patrols a loop of waypoints, its beacon coloured by a
// status severity (0..1).
type Robot struct {
	Pos    raytrace.Vec3
	Yaw    float64
	Status float64 // 0..1; colours the beacon green->red
	speed  float64
	wps    []raytrace.Vec3
	wp     int
	goal   *raytrace.Vec3 // a commanded destination (overrides the patrol until reached)
	t      float64
}

// NewRobot makes a robot at `at` that patrols the given waypoints (looping); with
// none it idles in place. status (0..1) sets the beacon colour.
func NewRobot(at raytrace.Vec3, waypoints []raytrace.Vec3, status float64) *Robot {
	s := status
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	return &Robot{Pos: at, Status: s, speed: 2.2, wps: waypoints}
}

// Step advances the robot by dt seconds toward its commanded goal if it has one,
// else along its patrol loop; it stays on the ground. On reaching a commanded goal
// the goal clears (AtGoal becomes true), so a driver can issue the next one.
func (r *Robot) Step(dt float64) {
	if r.speed <= 0 {
		r.speed = 2.2
	}
	r.t += dt
	target, has := r.Pos, false
	if r.goal != nil {
		target, has = *r.goal, true
	} else if len(r.wps) > 0 {
		target, has = r.wps[r.wp%len(r.wps)], true
	}
	if has {
		flat := raytrace.Vec3{X: target.X - r.Pos.X, Z: target.Z - r.Pos.Z}
		dist := flat.Len()
		if dist < 0.4 {
			if r.goal != nil {
				r.goal = nil // reached the commanded destination
			} else {
				r.wp = (r.wp + 1) % len(r.wps)
			}
		} else {
			dir := flat.Scale(1 / dist)
			step := r.speed * dt
			if step > dist {
				step = dist
			}
			r.Pos.X += dir.X * step
			r.Pos.Z += dir.Z * step
			r.Yaw = math.Atan2(dir.X, dir.Z)
		}
	}
	r.Pos.Y = 0.5 // body centre rides on the ground
}

// GoTo commands the robot toward p, overriding its patrol until it arrives.
func (r *Robot) GoTo(p raytrace.Vec3) { r.goal = &p }

// AtGoal reports whether the robot has no active commanded goal (so it is ready
// for the next one). A patrolling robot with no command is always "at goal".
func (r *Robot) AtGoal() bool { return r.goal == nil }

// Pose is the robot's pose, for rendering.
func (r *Robot) Pose() Pose { return Pose{Pos: r.Pos, Yaw: r.Yaw} }

// Objects renders the robot: a metal body, a head, and an emissive status beacon
// that brightens and reddens as its status worsens.
func (r *Robot) Objects() []raytrace.Object {
	c := robotStatusColor(r.Status)
	body := raytrace.Material{Color: raytrace.Vec3{X: 0.55, Y: 0.57, Z: 0.6}, Metal: 0.5, Rough: 0.3}
	head := raytrace.Material{Color: raytrace.Vec3{X: 0.5, Y: 0.52, Z: 0.56}, Metal: 0.4, Rough: 0.35}
	beacon := raytrace.Material{Color: c, Emit: c.Scale(0.8 + 2.2*r.Status), Rough: 0.4}
	return []raytrace.Object{
		raytrace.Sphere{Center: r.Pos, Radius: 0.45, Mat: body},
		raytrace.Sphere{Center: r.Pos.Add(raytrace.Vec3{Y: 0.6}), Radius: 0.28, Mat: head},
		raytrace.Sphere{Center: r.Pos.Add(raytrace.Vec3{Y: 1.02}), Radius: 0.2, Mat: beacon},
	}
}

// robotStatusColor maps a status 0..1 to green (calm) -> amber -> red (fault).
func robotStatusColor(s float64) raytrace.Vec3 {
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	if s < 0.5 {
		t := s / 0.5
		return raytrace.Vec3{X: 0.2 + 0.8*t, Y: 0.8, Z: 0.2}
	}
	t := (s - 0.5) / 0.5
	return raytrace.Vec3{X: 1.0, Y: 0.8 * (1 - t), Z: 0.15}
}

// SpawnRobot adds a patrolling robot to the world.
func (w *World) SpawnRobot(r *Robot) { w.robots = append(w.robots, r) }

// StepRobots advances every robot by dt seconds.
func (w *World) StepRobots(dt float64) {
	for _, r := range w.robots {
		r.Step(dt)
	}
}

// HasRobots reports whether the world has any robots.
func (w *World) HasRobots() bool { return len(w.robots) > 0 }

// Robots returns the world's robots (for a status read-out).
func (w *World) Robots() []*Robot { return w.robots }
