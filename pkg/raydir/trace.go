package raydir

import (
	"encoding/binary"
	"errors"
	"os"
	"sort"

	"github.com/svend4/infon/pkg/raytrace"
)

// trace.go is the world's memory of being walked: a coarse grid that counts how
// often each patch of ground has been trodden, rendered as worn earth — so paths
// emerge where people actually go. It is small, persistable (so the wear survives a
// session), and a kind of social archaeology of the shared world.

// Trace accumulates foot traffic per ground cell.
type Trace struct {
	cell   float64
	counts map[[2]int]int
	max    int
}

// NewTrace makes a trace with the given cell size (world units; default 1.5).
func NewTrace(cell float64) *Trace {
	if cell <= 0 {
		cell = 1.5
	}
	return &Trace{cell: cell, counts: map[[2]int]int{}}
}

func (t *Trace) cellOf(p raytrace.Vec3) [2]int {
	return [2]int{int(floorDiv(p.X, t.cell)), int(floorDiv(p.Z, t.cell))}
}

func floorDiv(a, b float64) float64 {
	q := a / b
	if q < 0 {
		return float64(int(q)) - 1
	}
	return float64(int(q))
}

// Tread records a step at p, deepening the path there.
func (t *Trace) Tread(p raytrace.Vec3) {
	c := t.cellOf(p)
	t.counts[c]++
	if t.counts[c] > t.max {
		t.max = t.counts[c]
	}
}

// Cells reports how many distinct ground cells have been trodden.
func (t *Trace) Cells() int { return len(t.counts) }

// Objects renders the worn patches: a flat, dark quad just above the floor for each
// well-trodden cell, darkening with traffic. Lightly trodden cells (count < 2) are
// skipped so a path emerges rather than a stain everywhere.
func (t *Trace) Objects() []raytrace.Object {
	var out []raytrace.Object
	for c, n := range t.counts {
		if n < 2 {
			continue
		}
		f := float64(n) / float64(t.max) // 0..1 wear
		shade := 0.32 - 0.18*f           // darker with traffic
		mat := raytrace.Material{Color: raytrace.Vec3{X: shade * 1.1, Y: shade, Z: shade * 0.8}, Rough: 0.9}
		cx := (float64(c[0]) + 0.5) * t.cell
		cz := (float64(c[1]) + 0.5) * t.cell
		out = append(out, boxObjects(
			raytrace.Vec3{X: cx, Y: 0.02, Z: cz},
			raytrace.Vec3{X: t.cell * 0.46, Y: 0.02, Z: t.cell * 0.46},
			mat)...)
	}
	return out
}

var (
	traceMagic = [4]byte{'R', 'T', 'R', 'C'}
	errTrace   = errors.New("raydir: malformed trace")
)

// EncodeTrace packs a trace as magic + cell size + count + [x i32][z i32][n u32].
func (t *Trace) Encode() []byte {
	keys := make([][2]int, 0, len(t.counts))
	for c := range t.counts {
		keys = append(keys, c)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	out := append([]byte(nil), traceMagic[:]...)
	var u [8]byte
	binary.BigEndian.PutUint64(u[:], uint64(int64(t.cell*1000))) // cell size in milli-units
	out = append(out, u[:]...)
	binary.BigEndian.PutUint32(u[:4], uint32(len(keys)))
	out = append(out, u[:4]...)
	for _, c := range keys {
		binary.BigEndian.PutUint32(u[:4], uint32(int32(c[0])))
		out = append(out, u[:4]...)
		binary.BigEndian.PutUint32(u[:4], uint32(int32(c[1])))
		out = append(out, u[:4]...)
		binary.BigEndian.PutUint32(u[:4], uint32(t.counts[c]))
		out = append(out, u[:4]...)
	}
	return out
}

// Save writes the trace to a file atomically (worn paths persist between sessions).
func (t *Trace) Save(path string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, t.Encode(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadTrace reads a trace previously saved with Save.
func LoadTrace(path string) (*Trace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeTrace(data)
}

// DecodeTrace parses bytes produced by Trace.Encode.
func DecodeTrace(data []byte) (*Trace, error) {
	if len(data) < 16 || [4]byte{data[0], data[1], data[2], data[3]} != traceMagic {
		return nil, errTrace
	}
	cell := float64(int64(binary.BigEndian.Uint64(data[4:12]))) / 1000
	t := NewTrace(cell)
	n := int(binary.BigEndian.Uint32(data[12:16]))
	off := 16
	for i := 0; i < n; i++ {
		if off+12 > len(data) {
			return nil, errTrace
		}
		x := int(int32(binary.BigEndian.Uint32(data[off : off+4])))
		z := int(int32(binary.BigEndian.Uint32(data[off+4 : off+8])))
		cnt := int(binary.BigEndian.Uint32(data[off+8 : off+12]))
		t.counts[[2]int{x, z}] = cnt
		if cnt > t.max {
			t.max = cnt
		}
		off += 12
	}
	return t, nil
}
