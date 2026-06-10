package raydir

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/brain"
)

// scenedelta.go crosses semframe's "semantic P-frame" idea with the rayscene world:
// a call sends not pixels, nor even a full scene each frame, but the DELTA — the
// objects that moved or changed since the last frame, plus any sky/light change.
// One sphere orbiting is a few dozen bytes; the receiver reconstructs the whole
// scene and ray-traces it locally. It is "meaning, not pixels" taken to its codec:
// scene3d and the compact rayscene wire already lived next door; this joins them.

// ObjChange is one object that changed (or is new) at a position in the scene.
type ObjChange struct {
	Index int           `json:"i"`
	Obj   brain.ObjSpec `json:"o"`
}

// SceneDelta is a rayscene P-frame: the objects changed since the previous scene,
// the new object count (so adds/removes reconstruct), and any sky/light change.
type SceneDelta struct {
	Count   int         `json:"n"`
	Changes []ObjChange `json:"c,omitempty"`
	SkyTop  *[3]float64 `json:"st,omitempty"`
	SkyBot  *[3]float64 `json:"sb,omitempty"`
	Light   *[3]float64 `json:"l,omitempty"`
	Name    string      `json:"nm,omitempty"`
}

// DiffScene computes the delta from prev to next. The bool is whether the delta is
// actually smaller than re-sending the whole scene (a keyframe) — false means "send
// a keyframe instead", as a real stream would on a scene cut.
func DiffScene(prev, next brain.SceneSpec) (SceneDelta, bool) {
	d := SceneDelta{Count: len(next.Objects), Name: next.Name}
	for i := range next.Objects {
		if i >= len(prev.Objects) || next.Objects[i] != prev.Objects[i] {
			d.Changes = append(d.Changes, ObjChange{Index: i, Obj: next.Objects[i]})
		}
	}
	if next.SkyTop != prev.SkyTop {
		st := next.SkyTop
		d.SkyTop = &st
	}
	if next.SkyBot != prev.SkyBot {
		sb := next.SkyBot
		d.SkyBot = &sb
	}
	if next.Light != prev.Light {
		l := next.Light
		d.Light = &l
	}
	return d, SceneBytes(d) < SceneBytes(next)
}

// ApplyScene reconstructs the next scene from prev and a delta.
func ApplyScene(prev brain.SceneSpec, d SceneDelta) brain.SceneSpec {
	out := prev
	out.Name = d.Name
	objs := make([]brain.ObjSpec, d.Count)
	for i := 0; i < d.Count && i < len(prev.Objects); i++ {
		objs[i] = prev.Objects[i]
	}
	for _, c := range d.Changes {
		if c.Index >= 0 && c.Index < d.Count {
			objs[c.Index] = c.Obj
		}
	}
	out.Objects = objs
	if d.SkyTop != nil {
		out.SkyTop = *d.SkyTop
	}
	if d.SkyBot != nil {
		out.SkyBot = *d.SkyBot
	}
	if d.Light != nil {
		out.Light = *d.Light
	}
	return out
}

// SceneBytes is the wire size of a scene or delta (compact JSON, omitempty).
func SceneBytes(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return len(b)
}

// SceneStream encodes a sequence of scenes as a keyframe followed by P-frame deltas
// (falling back to a keyframe when a delta would be larger), tracking the bytes it
// would put on the wire — so a moving world streams for a trickle.
type SceneStream struct {
	prev   brain.SceneSpec
	has    bool
	Bytes  int // total wire bytes emitted
	Frames int // total frames
	Keys   int // keyframes emitted
}

// Scene is the receiver's current reconstruction (prev applied through the deltas).
func (s *SceneStream) Scene() brain.SceneSpec { return s.prev }

// Push feeds the next scene and returns whether it was sent as a keyframe (true) or
// a delta (false), plus the bytes that frame put on the wire.
func (s *SceneStream) Push(scene brain.SceneSpec) (keyframe bool, frameBytes int) {
	s.Frames++
	if !s.has {
		frameBytes = SceneBytes(scene)
		s.prev, s.has, s.Bytes, s.Keys = scene, true, s.Bytes+frameBytes, s.Keys+1
		return true, frameBytes
	}
	d, ok := DiffScene(s.prev, scene)
	if !ok { // a cut: a keyframe is cheaper
		frameBytes = SceneBytes(scene)
		s.prev, s.Bytes, s.Keys = scene, s.Bytes+frameBytes, s.Keys+1
		return true, frameBytes
	}
	frameBytes = SceneBytes(d)
	s.prev, s.Bytes = ApplyScene(s.prev, d), s.Bytes+frameBytes
	return false, frameBytes
}
