package raydir

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

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
	n := 0
	for _, o := range spec.Objects {
		if o.Kind == "plane" {
			continue // a single shared floor for the whole world
		}
		rad := o.R
		if rad <= 0 {
			rad = 1
		}
		w.props = append(w.props, raytrace.Sphere{
			Center: raytrace.Vec3{X: o.X + at.X, Y: o.Y + at.Y, Z: o.Z + at.Z},
			Radius: rad, Mat: objMaterial(o),
		})
		n++
	}
	return n
}

// Has reports whether a region index has already been applied.
func (w *World) Has(index int) bool { return w.seen[index] }

// AddRegion applies a region received from the director host (idempotent).
func (w *World) AddRegion(r Region) int { return w.applyRegion(r.Index, r.At, r.Spec) }

// AuthorRegion (host side) asks the brain to author region `index` at `at`, applies
// it locally, and returns the Region to broadcast to peers.
func (w *World) AuthorRegion(b brain.Brain, prompt string, index int, at raytrace.Vec3) (Region, int, error) {
	_, spec, err := AuthorScene(b, prompt)
	if err != nil {
		return Region{}, 0, err
	}
	return Region{Index: index, At: at, Spec: spec}, w.applyRegion(index, at, spec), nil
}
