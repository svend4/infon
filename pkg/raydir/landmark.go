package raydir

import (
	"fmt"
	"math"
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// landmark.go names places and draws a map. Each authored region can have a name
// (the director's, or a default); the world remembers where its regions sit, and
// Minimap draws a top-down ASCII map of those landmarks and the walkers — so a
// growing world becomes navigable, and you can fast-travel to a named place.

// Landmark is a named region position for the map and fast-travel.
type Landmark struct {
	Index int
	At    raytrace.Vec3
	Name  string
}

// Landmarks returns the world's named regions (in application order).
func (w *World) Landmarks() []Landmark { return append([]Landmark(nil), w.landmarks...) }

// regionName is a region's place name: the director's, or a default by index.
func regionName(spec brain.SceneSpec, index int) string {
	if strings.TrimSpace(spec.Name) != "" {
		return spec.Name
	}
	return fmt.Sprintf("Region %d", index)
}

// FindLandmark returns the landmark whose name contains q (case-insensitive), or
// ok=false.
func (w *World) FindLandmark(q string) (Landmark, bool) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return Landmark{}, false
	}
	for _, l := range w.landmarks {
		if strings.Contains(strings.ToLower(l.Name), q) {
			return l, true
		}
	}
	return Landmark{}, false
}

// Minimap draws a top-down (X right, Z down) ASCII map sized cols x rows: landmarks
// as '#', other walkers as 'o', you as '@', then a legend of named places. It
// auto-fits the bounds of everything shown.
func Minimap(marks []Landmark, others []raytrace.Vec3, self raytrace.Vec3, cols, rows int) string {
	if cols < 8 {
		cols = 8
	}
	if rows < 4 {
		rows = 4
	}
	minX, maxX := self.X, self.X
	minZ, maxZ := self.Z, self.Z
	acc := func(p raytrace.Vec3) {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minZ, maxZ = math.Min(minZ, p.Z), math.Max(maxZ, p.Z)
	}
	for _, l := range marks {
		acc(l.At)
	}
	for _, o := range others {
		acc(o)
	}
	// pad so points aren't on the edge and zero-extent maps don't divide by zero.
	padX := math.Max(2, (maxX-minX)*0.1)
	padZ := math.Max(2, (maxZ-minZ)*0.1)
	minX, maxX, minZ, maxZ = minX-padX, maxX+padX, minZ-padZ, maxZ+padZ

	grid := make([][]rune, rows)
	for r := range grid {
		grid[r] = make([]rune, cols)
		for c := range grid[r] {
			grid[r][c] = '.'
		}
	}
	put := func(p raytrace.Vec3, ch rune) {
		c := int((p.X - minX) / (maxX - minX) * float64(cols-1))
		r := int((p.Z - minZ) / (maxZ - minZ) * float64(rows-1))
		if c >= 0 && c < cols && r >= 0 && r < rows {
			grid[r][c] = ch
		}
	}
	for _, l := range marks {
		put(l.At, '#')
	}
	for _, o := range others {
		put(o, 'o')
	}
	put(self, '@')

	var b strings.Builder
	for r := 0; r < rows; r++ {
		b.WriteString(string(grid[r]))
		b.WriteByte('\n')
	}
	for _, l := range marks {
		fmt.Fprintf(&b, "# %s (%.0f,%.0f)\n", l.Name, l.At.X, l.At.Z)
	}
	return b.String()
}
