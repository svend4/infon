package raydir

import (
	"math"
	"strings"

	"github.com/svend4/infon/pkg/raytrace"
)

// sprite.go puts dream characters in the world — the "sprites" the hackers debate:
// can you talk to them, can you tell a sprite from a real dreamer? A Sprite drifts
// about its spot and answers questions from a small dream-logic table. The tell is
// the classic one: ask a sprite whether this is a dream and it deflects or denies,
// where a lucid dreamer would simply say yes. Local and deterministic.

// SpriteColor is a dream character's pale, faintly luminous look.
var SpriteColor = raytrace.Vec3{X: 0.72, Y: 0.8, Z: 0.86}

// Sprite is a dream character you can question.
type Sprite struct {
	Name string
	Pos  raytrace.Vec3
	home raytrace.Vec3
	seed int
	t    float64
}

// NewSprite makes a sprite named `name` at `at`.
func NewSprite(name string, at raytrace.Vec3, seed int) *Sprite {
	return &Sprite{Name: name, Pos: at, home: at, seed: seed}
}

// Step drifts the sprite gently about its spot (so it feels alive), deterministic
// in its clock.
func (s *Sprite) Step(dt float64) {
	s.t += dt
	r := 0.6
	s.Pos = s.home.Add(raytrace.Vec3{
		X: math.Sin(s.t*0.5+float64(s.seed)) * r,
		Z: math.Cos(s.t*0.37+float64(s.seed)) * r,
	})
}

// Pose is the sprite's pose, for rendering an avatar.
func (s *Sprite) Pose() Pose {
	return Pose{Pos: s.Pos, Yaw: math.Sin(s.t*0.5) * 1.5}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// crypticLines are a sprite's default dream-logic replies (chosen by seed).
var crypticLines = []string{
	"I was just about to remember something.",
	"You look like someone I lost.",
	"Keep walking and it stays. Stop and it forgets.",
	"The north moved again last night.",
	"Don't follow the lights too far.",
}

// Answer replies to a question with dream logic. The tell: asked whether this is a
// dream, a sprite deflects or denies (a lucid dreamer would say yes).
func (s *Sprite) Answer(q string) string {
	ql := strings.ToLower(q)
	switch {
	case containsAny(ql, "dream", "asleep", "sleep", "real", "awake", "wake"):
		return "A dream? What a thing to ask. No — this is as real as anything." // the deflection
	case containsAny(ql, "who", "name", "you"):
		return "I'm " + s.Name + ". I think I've always been here."
	case containsAny(ql, "where", "place", "here", "this"):
		return "The edge of somewhere. It was different a moment ago."
	case containsAny(ql, "exit", "out", "leave", "door", "way"):
		return "There's no door. There was never a door."
	case containsAny(ql, "help", "lost", "find"):
		return "I can't help. I'm only passing through, like you."
	default:
		return crypticLines[s.seed%len(crypticLines)]
	}
}

// LucidTell reports whether an answer reads as a sprite's deflection rather than a
// lucid dreamer's admission — the hackers' way to tell a sprite from a dreamer.
func LucidTell(answer string) bool {
	a := strings.ToLower(answer)
	return containsAny(a, "no", "real", "what a thing") && !strings.Contains(a, "yes")
}

// Objects renders the sprite as a pale avatar with a soft glow.
func (s *Sprite) Objects() []raytrace.Object {
	return AvatarSpheres(s.Pose(), SpriteColor)
}

// AddSprite places a dream character in the world.
func (w *World) AddSprite(s *Sprite) { w.sprites = append(w.sprites, s) }

// StepSprites drifts every sprite by dt seconds.
func (w *World) StepSprites(dt float64) {
	for _, s := range w.sprites {
		s.Step(dt)
	}
}

// NearestSprite returns the closest sprite within `radius` of p, or nil.
func (w *World) NearestSprite(p raytrace.Vec3, radius float64) *Sprite {
	var best *Sprite
	bestD := radius * radius
	for _, s := range w.sprites {
		if d := s.Pos.Sub(p).LenSq(); d <= bestD {
			best, bestD = s, d
		}
	}
	return best
}

// HasSprites reports whether the world has any dream characters.
func (w *World) HasSprites() bool { return len(w.sprites) > 0 }
