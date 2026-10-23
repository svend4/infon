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

// MaxRegionBytes is the largest Encode that fits one network packet: the
// transport's payload limit (network.MaxPacketSize - network.PacketHeaderSize). A
// larger region is refused by SendPacket and never reaches a peer, so authored
// regions are trimmed to it (see Fit).
const MaxRegionBytes = 1400 - 13

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

// Fit returns r trimmed so its Encode fits one packet (MaxRegionBytes). It drops
// the place name only if even an empty scene overflows, then sheds non-emissive
// objects from the end (keeping lights and the ground plane), and only if the
// lights alone overflow falls back to the longest prefix of objects that fits. A
// region that already fits is returned unchanged.
func (r Region) Fit() Region {
	if len(r.Encode()) <= MaxRegionBytes {
		return r
	}
	objs := r.Spec.Objects
	r.Spec.Objects = nil
	if len(r.Encode()) > MaxRegionBytes {
		r.Spec.Name = ""
	}
	// Shed plain objects from the end first, keeping emitters and the ground: the
	// lights carry far more of a scene's look than one more form (SceneVector lists
	// its sun last), and an overflow is usually only a few bytes.
	keep := append([]brain.ObjSpec(nil), objs...)
	r.Spec.Objects = keep
	for i := len(keep) - 1; i >= 0 && len(r.Encode()) > MaxRegionBytes; i-- {
		if keep[i].Emit == ([3]float64{}) && keep[i].Kind != "plane" {
			keep = append(keep[:i], keep[i+1:]...)
			r.Spec.Objects = keep
		}
	}
	if len(r.Encode()) <= MaxRegionBytes {
		return r
	}
	// Still too big (mostly emitters): keep the longest prefix that fits. The
	// encoding grows with every object, so binary-search it.
	k := sort.Search(len(objs), func(n int) bool {
		t := r
		t.Spec.Objects = objs[:n+1]
		return len(t.Encode()) > MaxRegionBytes
	})
	r.Spec.Objects = append([]brain.ObjSpec(nil), objs[:k]...)
	return r
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
	if !validAt(r.At) {
		return Region{}, errors.New("raydir: region position out of range")
	}
	if err := json.Unmarshal(b[28:], &r.Spec); err != nil {
		return Region{}, err
	}
	return r, nil
}

// validAt reports whether a region position is finite and in range (see finite),
// so a malformed one is never placed — nor kept by Prune, which orders by depth.
func validAt(at raytrace.Vec3) bool { return finite(at.X) && finite(at.Y) && finite(at.Z) }

// World size caps: past either, the furthest-behind regions are shed (as Prune
// would drop them), so a peer flooding new region indices can't grow memory or the
// per-frame scene without bound.
const (
	maxAppliedRegions = 1024
	maxWorldProps     = 1 << 16
)

// applyRegion adds a region's props exactly once (idempotent by index), so a
// guest can re-apply re-broadcast regions safely. The region is also remembered
// (w.applied) so far-behind ones can later be pruned (see Prune). A region at a
// non-finite or out-of-range position is ignored, and past the world caps the
// furthest-behind regions are shed.
func (w *World) applyRegion(index int, at raytrace.Vec3, spec brain.SceneSpec) int {
	if w.seen == nil {
		w.seen = map[int]bool{}
	}
	if w.seen[index] || !validAt(at) {
		return 0
	}
	w.applied = append(w.applied, Region{Index: index, At: at, Spec: spec})
	n := w.applySpec(index, at, spec)
	for len(w.applied) > 1 && (len(w.applied) > maxAppliedRegions || len(w.props)+len(w.animated) > maxWorldProps) {
		w.shed(len(w.applied)/4 + 1) // in batches, so a long walk rebuilds rarely
	}
	return n
}

// shed drops the n furthest-behind regions (lowest Z, the earliest applied on a
// tie) and rebuilds the derived state from the rest, in their applied order.
func (w *World) shed(n int) {
	order := make([]int, len(w.applied))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return w.applied[order[a]].At.Z < w.applied[order[b]].At.Z })
	drop := make(map[int]bool, n)
	for _, i := range order[:min(n, len(order))] {
		drop[i] = true
	}
	keep := make([]Region, 0, len(w.applied))
	for i, r := range w.applied {
		if !drop[i] {
			keep = append(keep, r)
		}
	}
	w.rebuild(keep)
}

// applySpec adds a region's derived state (props, movers, landmark, sound flags).
// Assumes the region is not already applied; used by applyRegion and by rebuild
// (Prune, shed).
func (w *World) applySpec(index int, at raytrace.Vec3, spec brain.SceneSpec) int {
	if w.seen == nil {
		w.seen = map[int]bool{}
	}
	w.seen[index] = true
	w.chunks = len(w.seen)
	if at.Z > w.frontierZ { // track the furthest region (the seasonal frontier)
		w.frontierZ = at.Z
	}
	w.landmarks = append(w.landmarks, Landmark{Index: index, At: at, Name: regionName(spec, index)})
	n := 0
	for i, o := range spec.Objects {
		if i >= maxRegionObjects { // cap so a runaway model can't flood the world
			break
		}
		if w.seasonal { // tint this region's foliage for the season where it sits
			o = seasonTintSpec(o, at.Z)
		}
		switch { // remember features for the soundscape (see World.Ambient)
		case o.Kind == "water":
			w.sndWater = true
		case o.Kind == "tree":
			w.sndForest++
		case o.Anim == "orbit":
			w.sndBirds = true
		case o.Kind == "fractal":
			w.sndHum = true
		}
		if validObj(o) && (isAnimated(o.Anim) || o.Kind == "water") {
			// a moving object (or water): keep its spec and rebuild it each frame from
			// the shared clock (see SceneWith), rather than baking it into static props.
			w.animated = append(w.animated, animObj{spec: o, at: at, seed: len(w.animated)*97 + index, born: w.animTime})
			n += len(objectsFromSpec(o, at, false))
			continue
		}
		objs := objectsFromSpec(o, at, false) // one shared floor: planes are skipped
		w.props = append(w.props, objs...)
		n += len(objs)
	}
	return n
}

// Prune drops regions sitting behind minZ (far back as you walk forward) and
// rebuilds the derived state from the survivors, so a long exploration keeps flat
// memory and render cost. Worn paths (the trace) are kept — the world still
// remembers. Returns how many regions were dropped. (A guest re-fetches a pruned
// region via ack gap-fill if it walks back; a host re-grows.)
func (w *World) Prune(minZ float64) int {
	if math.IsNaN(minZ) { // no depth to prune behind
		return 0
	}
	keep := make([]Region, 0, len(w.applied))
	dropped := 0
	for _, r := range w.applied {
		if r.At.Z >= minZ {
			keep = append(keep, r)
		} else {
			dropped++
		}
	}
	if dropped == 0 {
		return 0
	}
	w.rebuild(keep)
	return dropped
}

// rebuild resets the derived state and re-applies the surviving regions.
func (w *World) rebuild(keep []Region) {
	w.props = w.props[:0]
	w.animated = w.animated[:0]
	w.landmarks = w.landmarks[:0]
	w.seen = map[int]bool{}
	w.chunks = 0
	w.frontierZ = 0
	w.sndWater, w.sndForest, w.sndBirds, w.sndHum = false, 0, false, false
	w.applied = keep
	for _, r := range keep {
		w.applySpec(r.Index, r.At, r.Spec)
	}
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
	reg := Region{Index: index, At: at, Spec: spec}.Fit() // apply exactly what peers receive
	w.lastSpec, w.lastAt = &reg.Spec, at
	return reg, w.applyRegion(index, at, reg.Spec), nil
}
