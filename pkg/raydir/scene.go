// scene.go bridges a tvcp-ai/1 brain's scene authoring (game "rayscene") to the
// renderer: it turns a brain.SceneSpec (the protocol's JSON scene graph of objects
// with full materials, a light and a sky) into a renderable raytrace.Scene. So
// "describe a 3-D scene" becomes pixels through the same path tracer used
// everywhere else. AuthorScene drives any brain.Brain; RefSceneBrain delegates to
// the protocol reference author, so the bridge is testable offline.
package raydir

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func vec3(a [3]float64) raytrace.Vec3 { return raytrace.Vec3{X: a[0], Y: a[1], Z: a[2]} }

func objMaterial(o brain.ObjSpec) raytrace.Material {
	return raytrace.Material{
		Color:   vec3(o.Color),
		Emit:    vec3(o.Emit),
		Glass:   o.Glass,
		Metal:   o.Metal,
		Reflect: o.Reflect,
		Rough:   o.Rough,
	}
}

// objectsFromSpec turns one ObjSpec into renderable objects, offset by `at`. A box
// becomes 12 triangles; a sphere one analytic sphere; a plane the shared floor
// (only when includePlane is set).
func objectsFromSpec(o brain.ObjSpec, at raytrace.Vec3, includePlane bool) []raytrace.Object {
	switch o.Kind {
	case "plane":
		if !includePlane {
			return nil
		}
		c := vec3(o.Color)
		return []raytrace.Object{raytrace.Plane{Y: o.Y, Size: 1, C1: c, C2: c.Scale(0.6)}}
	case "box":
		half := raytrace.Vec3{X: o.S[0], Y: o.S[1], Z: o.S[2]}
		if half.LenSq() == 0 {
			r := o.R
			if r <= 0 {
				r = 1
			}
			half = raytrace.Vec3{X: r, Y: r, Z: r}
		}
		return boxObjects(raytrace.Vec3{X: o.X + at.X, Y: o.Y + at.Y, Z: o.Z + at.Z}, half, objMaterial(o))
	default:
		r := o.R
		if r <= 0 {
			r = 1
		}
		return []raytrace.Object{raytrace.Sphere{
			Center: raytrace.Vec3{X: o.X + at.X, Y: o.Y + at.Y, Z: o.Z + at.Z}, Radius: r, Mat: objMaterial(o),
		}}
	}
}

// boxObjects builds an axis-aligned cuboid (12 triangles) centred at c with the
// given half-extents.
func boxObjects(c, half raytrace.Vec3, mat raytrace.Material) []raytrace.Object {
	mk := func(sx, sy, sz float64) raytrace.Vec3 {
		return raytrace.Vec3{X: c.X + sx*half.X, Y: c.Y + sy*half.Y, Z: c.Z + sz*half.Z}
	}
	p := [8]raytrace.Vec3{
		mk(-1, -1, -1), mk(1, -1, -1), mk(1, 1, -1), mk(-1, 1, -1), // z-
		mk(-1, -1, 1), mk(1, -1, 1), mk(1, 1, 1), mk(-1, 1, 1), // z+
	}
	quad := func(a, b, cc, d int) []raytrace.Object {
		return []raytrace.Object{
			raytrace.Triangle{A: p[a], B: p[b], C: p[cc], Mat: mat},
			raytrace.Triangle{A: p[a], B: p[cc], C: p[d], Mat: mat},
		}
	}
	var t []raytrace.Object
	t = append(t, quad(0, 1, 2, 3)...) // z-
	t = append(t, quad(4, 5, 6, 7)...) // z+
	t = append(t, quad(0, 1, 5, 4)...) // y-
	t = append(t, quad(3, 2, 6, 7)...) // y+
	t = append(t, quad(1, 2, 6, 5)...) // x+
	t = append(t, quad(0, 3, 7, 4)...) // x-
	return t
}

// BuildScene turns a brain.SceneSpec into a renderable raytrace.Scene.
func BuildScene(spec brain.SceneSpec) *raytrace.Scene {
	s := &raytrace.Scene{
		Light:     vec3(spec.Light),
		SkyTop:    vec3(spec.SkyTop),
		SkyBottom: vec3(spec.SkyBot),
	}
	for _, o := range spec.Objects {
		s.Objects = append(s.Objects, objectsFromSpec(o, raytrace.Vec3{}, true)...)
	}
	return s
}

// DecodeSceneSpec parses a SceneSpec JSON blob into a renderable scene.
func DecodeSceneSpec(data []byte) (*raytrace.Scene, brain.SceneSpec, error) {
	var spec brain.SceneSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, brain.SceneSpec{}, err
	}
	return BuildScene(spec), spec, nil
}

// AuthorScene asks a brain to author a scene for the prompt (game "rayscene") and
// decodes the response into a renderable scene.
func AuthorScene(b brain.Brain, prompt string) (*raytrace.Scene, brain.SceneSpec, error) {
	state, _ := json.Marshal(struct {
		Prompt string `json:"prompt"`
	}{prompt})
	resp, err := b.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "rayscene", State: state})
	if err != nil {
		return nil, brain.SceneSpec{}, err
	}
	return DecodeSceneSpec(resp.Ray)
}

// RefSceneBrain is a deterministic reference scene author: it delegates to the
// protocol's reference brain (game "rayscene"), so the same scene the conformance
// battery exercises is what the bridge renders.
type RefSceneBrain struct{}

// Decide implements brain.Brain for game "rayscene".
func (RefSceneBrain) Decide(req brain.Request) (brain.Response, error) {
	return brain.Reference(req), nil
}
