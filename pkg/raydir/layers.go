package raydir

import "github.com/svend4/infon/pkg/raytrace"

// layers.go stacks the dream world vertically, as the shamanic maps in the hackers'
// letters do: an upper world (airy, bright — the realm of possibility), the middle
// world we walk, and a lower world (dark, cold, a night city of lamps — the realm
// of the past) reached by falling down a dark, ribbed tunnel. Choosing a layer
// repaints the sky to its mood; the descent tunnel is buildable geometry.

// Layer is which level of the world you are on.
type Layer int

const (
	LayerMiddle Layer = iota // the everyday ground
	LayerUpper               // airy and bright
	LayerLower               // dark, cold, a night city
)

// LayerSky returns the sky colours (top, bottom) for a layer.
func LayerSky(l Layer) (top, bottom raytrace.Vec3) {
	switch l {
	case LayerUpper:
		return raytrace.Vec3{X: 0.55, Y: 0.72, Z: 0.95}, raytrace.Vec3{X: 0.95, Y: 0.97, Z: 1.0}
	case LayerLower:
		return raytrace.Vec3{X: 0.02, Y: 0.03, Z: 0.06}, raytrace.Vec3{X: 0.05, Y: 0.05, Z: 0.09}
	default:
		return raytrace.Vec3{X: 0.45, Y: 0.6, Z: 0.85}, raytrace.Vec3{X: 0.85, Y: 0.88, Z: 0.95}
	}
}

// SetLayer moves the world to a layer, repainting its sky to match.
func (w *World) SetLayer(l Layer) {
	w.layer = l
	w.SkyTop, w.SkyBottom = LayerSky(l)
}

// Layer returns the world's current layer.
func (w *World) Layer() Layer { return w.layer }

// DescentTunnel builds a dark, ribbed vertical shaft from `at` down by `depth`: a
// stack of thin rings of alternating radius (the parallel grooves the dreamers
// describe), open in the middle so you fall through into the lower world.
func DescentTunnel(at raytrace.Vec3, depth float64) []raytrace.Object {
	mat := raytrace.Material{Color: raytrace.Vec3{X: 0.12, Y: 0.12, Z: 0.14}, Rough: 0.8}
	var out []raytrace.Object
	step := 0.6
	for y := 0.0; y < depth; y += step {
		r := 3.0
		if int(y/step)%2 == 0 { // alternate radius -> grooves
			r = 2.7
		}
		out = append(out, cylinderObjects(raytrace.Vec3{X: at.X, Y: at.Y - y, Z: at.Z}, r, 0.16, mat, 18)...)
	}
	return out
}
