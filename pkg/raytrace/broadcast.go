// broadcast.go turns a ray-scene animation into a loss-tolerant spectator stream:
// one self-contained Reed-Solomon packet per tick, each a full scene "keyframe",
// so a spectator can decode any packet on its own and simply freezes on a lost or
// corrupt one. It is the same channel idea as pkg/arena's semantic broadcast —
// "watch the 3-D world over a few hundred bytes per second" — applied to the ray
// tracer: many viewers, each rendering the received scene locally.
package raytrace

// EncodeBroadcast emits one independent RS packet per animation frame. Unlike the
// delta stream (EncodeAnim), every packet is a complete scene, so any single
// packet that arrives intact can be rendered with no dependence on the others.
func EncodeBroadcast(a SceneAnim, ecc int) [][]byte {
	out := make([][]byte, 0, len(a.Frames))
	for _, f := range a.Frames {
		w := f
		if len(w.Spheres) > a.N {
			w.Spheres = w.Spheres[:a.N]
		}
		out = append(out, Encode(w, ecc))
	}
	return out
}

// Spectator reconstructs the scene stream a viewer sees. Feed each received
// packet to Recv (or call Miss for a tick that never arrived); it freezes on the
// last good scene on loss/corruption and updates on a good packet.
type Spectator struct {
	ecc  int
	last SceneWire
	have bool
}

// NewSpectator makes a spectator that decodes ecc-protected scene packets.
func NewSpectator(ecc int) *Spectator { return &Spectator{ecc: ecc} }

// Recv applies one received packet and returns the current scene plus a live flag
// (true if the packet decoded and updated the view).
func (s *Spectator) Recv(wire []byte) (SceneWire, bool) {
	w, err := Decode(wire, s.ecc)
	if err != nil {
		return s.last, false
	}
	s.last, s.have = w, true
	return w, true
}

// Miss records a lost tick and returns the frozen (last good) scene.
func (s *Spectator) Miss() (SceneWire, bool) { return s.last, false }

// Have reports whether the spectator has decoded at least one packet.
func (s *Spectator) Have() bool { return s.have }
