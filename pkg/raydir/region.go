package raydir

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"sort"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// Region is one authored chunk of the shared world: its index, where it sits, and
// the brain's scene description for it. A director host broadcasts Regions so a
// guest reconstructs each identical region locally — meaning (a scene graph), not
// pixels — which is what lets the shared world stay in sync even with a live,
// non-deterministic AI director.
type Region struct {
	Index int
	At    raytrace.Vec3
	Spec  brain.SceneSpec
}

// Encode packs a region as index(4) + position(24) + SceneSpec JSON.
func (r Region) Encode() []byte {
	head := make([]byte, 28)
	binary.BigEndian.PutUint32(head[0:], uint32(r.Index))
	binary.BigEndian.PutUint64(head[4:], math.Float64bits(r.At.X))
	binary.BigEndian.PutUint64(head[12:], math.Float64bits(r.At.Y))
	binary.BigEndian.PutUint64(head[20:], math.Float64bits(r.At.Z))
	body, _ := json.Marshal(r.Spec)
	return append(head, body...)
}

// DecodeRegion unpacks a region produced by Encode.
func DecodeRegion(b []byte) (Region, error) {
	if len(b) < 28 {
		return Region{}, errors.New("raydir: region payload too short")
	}
	r := Region{
		Index: int(binary.BigEndian.Uint32(b[0:])),
		At: raytrace.Vec3{
			X: math.Float64frombits(binary.BigEndian.Uint64(b[4:])),
			Y: math.Float64frombits(binary.BigEndian.Uint64(b[12:])),
			Z: math.Float64frombits(binary.BigEndian.Uint64(b[20:])),
		},
	}
	if err := json.Unmarshal(b[28:], &r.Spec); err != nil {
		return Region{}, err
	}
	return r, nil
}

// applyRegion adds a region's props exactly once (idempotent by index), so a
// guest can re-apply re-broadcast regions safely.
func (w *World) applyRegion(index int, at raytrace.Vec3, spec brain.SceneSpec) int {
	if w.seen == nil {
		w.seen = map[int]bool{}
	}
	if w.seen[index] {
		return 0
	}
	w.seen[index] = true
	w.chunks = len(w.seen)
	w.landmarks = append(w.landmarks, Landmark{Index: index, At: at, Name: regionName(spec, index)})
	n := 0
	for i, o := range spec.Objects {
		if i >= maxRegionObjects { // cap so a runaway model can't flood the world
			break
		}
		if isAnimated(o.Anim) && validObj(o) {
			// a moving object: keep its spec and re-place it each frame (see SceneWith).
			w.animated = append(w.animated, animObj{spec: o, at: at, seed: len(w.animated)*97 + index})
			n += len(objectsFromSpec(o, at, false))
			continue
		}
		objs := objectsFromSpec(o, at, false) // one shared floor: planes are skipped
		w.props = append(w.props, objs...)
		n += len(objs)
	}
	return n
}

// Has reports whether a region index has already been applied.
func (w *World) Has(index int) bool { return w.seen[index] }

// Known returns the sorted region indices this world has applied — what a guest
// acknowledges so the host can fill only the gaps.
func (w *World) Known() []int {
	out := make([]int, 0, len(w.seen))
	for i := range w.seen {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}

// EncodeAck packs a list of region indices a peer already has (count + uint32s).
func EncodeAck(indices []int) []byte {
	b := make([]byte, 2, 2+len(indices)*4)
	binary.BigEndian.PutUint16(b[0:], uint16(len(indices)))
	for _, i := range indices {
		var h [4]byte
		binary.BigEndian.PutUint32(h[:], uint32(i))
		b = append(b, h[:]...)
	}
	return b
}

// DecodeAck parses an ack produced by EncodeAck.
func DecodeAck(b []byte) ([]int, error) {
	if len(b) < 2 {
		return nil, errors.New("raydir: ack too short")
	}
	n := int(binary.BigEndian.Uint16(b[0:]))
	off := 2
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if off+4 > len(b) {
			return nil, errors.New("raydir: truncated ack")
		}
		out = append(out, int(binary.BigEndian.Uint32(b[off:])))
		off += 4
	}
	return out, nil
}

// MissingRegions returns the regions whose index is not in `have` — what the host
// re-sends to a guest in response to its ack, instead of blindly re-broadcasting
// everything to everyone.
func MissingRegions(have []int, regions []Region) []Region {
	hs := make(map[int]bool, len(have))
	for _, i := range have {
		hs[i] = true
	}
	var out []Region
	for _, r := range regions {
		if !hs[r.Index] {
			out = append(out, r)
		}
	}
	return out
}

// AddRegion applies a region received from the director host (idempotent).
func (w *World) AddRegion(r Region) int { return w.applyRegion(r.Index, r.At, r.Spec) }

// AuthorRegion (host side) asks the brain to author region `index` at `at`, applies
// it locally, and returns the Region to broadcast to peers.
func (w *World) AuthorRegion(b brain.Brain, prompt string, index int, at raytrace.Vec3) (Region, int, error) {
	_, spec, err := AuthorSceneCtx(b, prompt, w.context(index, at))
	if err != nil {
		return Region{}, 0, err
	}
	w.lastSpec, w.lastAt = &spec, at
	return Region{Index: index, At: at, Spec: spec}, w.applyRegion(index, at, spec), nil
}
