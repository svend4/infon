package raydir

import (
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// build.go turns a walker's "place this here" into a one-object scene spec, so
// people can co-create the shared world: a placement becomes an ordinary Region
// (carried, ack-healed and persisted like any authored region) — meaning, not
// pixels, and the same sanitiser guards it.

// placeKinds maps a placement keyword to a scene-graph (kind, name).
var placeKinds = map[string]struct{ kind, name string }{
	"box": {"box", ""}, "sphere": {"sphere", ""}, "pyramid": {"pyramid", ""},
	"cylinder": {"cylinder", ""}, "tree": {"tree", ""}, "house": {"house", ""},
	"crystal": {"mesh", "crystal"}, "rock": {"mesh", "rock"},
	"mandelbulb": {"fractal", "mandelbulb"}, "menger": {"fractal", "menger"},
	"sierpinski": {"fractal", "sierpinski"}, "mandala": {"fractal", "mandala"},
	"melt": {"fractal", "melt"}, "escher": {"fractal", "escher"},
}

// PlaceKinds lists the placement keywords a walker can build, for help text.
func PlaceKinds() []string {
	out := make([]string, 0, len(placeKinds))
	for k := range placeKinds {
		out = append(out, k)
	}
	return out
}

// PlacementSpec builds a one-object scene spec for a placed item of the given kind,
// tinted in the builder's colour and sitting on the ground at the region origin.
// ok is false for an unknown kind. The result is meant to be a Region whose At is
// the world position in front of the builder.
func PlacementSpec(kind string, colour raytrace.Vec3) (brain.SceneSpec, bool) {
	k, ok := placeKinds[strings.ToLower(strings.TrimSpace(kind))]
	if !ok {
		return brain.SceneSpec{}, false
	}
	o := brain.ObjSpec{
		Kind: k.kind, Name: k.name, X: 0, Y: 1, Z: 0, R: 1,
		Color: [3]float64{colour.X, colour.Y, colour.Z},
	}
	if k.kind == "fractal" {
		o.Y, o.R, o.Reflect = 1.3, 1.3, 0.2
	}
	return brain.SceneSpec{Objects: []brain.ObjSpec{o}}, true
}
