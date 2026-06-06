// Package world closes the loop: a State is the whole pseudo-3D scene as one
// 51-byte record (fold angles + relief height + scene items + camera yaw); a
// Brain reads a State and emits the next one. The game loop is just
// state -> Brain.Next -> state, shipped frame by frame over a deltastream and
// drawn by the single isometric renderer. RefBrain is a deterministic stand-in;
// an LLM brain (tvcp-ai) implements the same one-method interface, so "neural ->
// bytes -> 3-D frame" is one cycle. The combined renderer (path 1 body + path 2
// relief + path 3 scene, one fold.RenderCam) now lives here, reusable.
package world

import (
	"image"
	"math"

	"github.com/svend4/infon/pkg/fold"
	"github.com/svend4/infon/pkg/relief"
	"github.com/svend4/infon/pkg/scene3d"
	"github.com/svend4/infon/pkg/tangram"
	t7 "github.com/svend4/infon/pkg/tangram7"
)

// StateBytes is the packed size of a State.
const StateBytes = 7 + 1 + 7*6 + 1 // fold + relief + scene + cam

// State is the whole world: 7 fold angles (a folding body), 1 relief height (a
// rising voxel animal), 7 scene items (orbiting slabs) and a camera yaw.
type State struct {
	Fold   [7]byte
	Relief byte
	Scene  [7]scene3d.Item
	Cam    byte
}

// Bytes packs the state for the wire.
func (s State) Bytes() []byte {
	b := make([]byte, 0, StateBytes)
	b = append(b, s.Fold[:]...)
	b = append(b, s.Relief)
	for _, it := range s.Scene {
		b = append(b, it.Piece, it.X, it.Y, it.Z, it.Yaw, it.Color)
	}
	b = append(b, s.Cam)
	return b
}

// FromBytes unpacks a state (missing bytes default to zero).
func FromBytes(b []byte) State {
	var s State
	if len(b) < StateBytes {
		return s
	}
	copy(s.Fold[:], b[0:7])
	s.Relief = b[7]
	for i := 0; i < 7; i++ {
		off := 8 + i*6
		s.Scene[i] = scene3d.Item{Piece: b[off], X: b[off+1], Y: b[off+2], Z: b[off+3], Yaw: b[off+4], Color: b[off+5]}
	}
	s.Cam = b[50]
	return s
}

func shift(fs []fold.Face, dx, dy, dz float64) []fold.Face {
	out := make([]fold.Face, len(fs))
	for i, f := range fs {
		pts := make([]fold.Pt3, len(f.Pts))
		for j, p := range f.Pts {
			pts[j] = fold.Pt3{X: p.X + dx, Y: p.Y + dy, Z: p.Z + dz}
		}
		out[i] = fold.Face{Pts: pts, Color: f.Color, Shade: f.Shade}
	}
	return out
}

// Render draws the whole world in one isometric image (camera yaw applied).
func (s State) Render(pxW, pxH int) image.Image {
	cat := tangram.Cat()
	angles := make([]float64, 7)
	for i := 0; i < 7; i++ {
		angles[i] = float64(s.Fold[i]) / 255 * fold.FoldMaxAngle
	}
	var all []fold.Face
	all = append(all, shift(relief.Faces(cat, relief.Uniform(cat, s.Relief)), 0, 0, 0)...)
	all = append(all, shift(fold.FacesFold(t7.Square(), angles), 13, 2, 0)...)
	all = append(all, shift(scene3d.SceneFaces(s.Scene[:]), 23, 0, 0)...)
	return fold.RenderCam(all, pxW, pxH, 2*math.Pi*float64(s.Cam)/256)
}

// Brain emits the next world state from the current one. An LLM adapter
// implements the same interface.
type Brain interface {
	Next(State) State
}

func clamp(v float64) byte {
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}

// RefBrain is a deterministic reference brain: it advances a master phase
// (carried in Cam) and expands it into a breathing, orbiting, rotating world -
// a compact latent rendered into a full scene each tick.
type RefBrain struct{}

// Next produces the next state from the current phase (Cam).
func (RefBrain) Next(s State) State {
	cam := byte(int(s.Cam) + 5)
	ph := 2 * math.Pi * float64(cam) / 256
	var ns State
	ns.Cam = cam
	for i := 0; i < 7; i++ {
		ns.Fold[i] = clamp(20 + 110*(0.5*(1+math.Sin(ph+float64(i)*0.3))))
	}
	ns.Relief = clamp(20 + 100*(0.5*(1+math.Sin(ph))))
	for i := 0; i < 7; i++ {
		ang := float64(i)*(2*math.Pi/7) + ph
		ns.Scene[i] = scene3d.Item{
			Piece: byte(i),
			X:     clamp(128 + 70*math.Cos(ang)),
			Y:     clamp(128 + 70*math.Sin(ang)),
			Z:     clamp(120 + 90*math.Sin(ph+float64(i))),
			Yaw:   byte((i + int(cam)/32) % 8),
			Color: byte(i),
		}
	}
	return ns
}

// Loop runs the brain for n ticks and returns the sequence of states.
func Loop(b Brain, init State, n int) []State {
	out := make([]State, 0, n)
	s := init
	for i := 0; i < n; i++ {
		s = b.Next(s)
		out = append(out, s)
	}
	return out
}
