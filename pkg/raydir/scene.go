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
		return boxObjects(specCenter(o, at), specHalf(o), objMaterial(o))
	case "pyramid":
		return pyramidObjects(specCenter(o, at), specHalf(o), objMaterial(o))
	case "cylinder":
		h := specHalf(o)
		return cylinderObjects(specCenter(o, at), h.X, h.Y, objMaterial(o), 16)
	case "tree":
		return treeObjects(specCenter(o, at), specScale(o), objMaterial(o))
	case "house":
		return houseObjects(specCenter(o, at), specScale(o), objMaterial(o))
	default:
		return []raytrace.Object{raytrace.Sphere{
			Center: specCenter(o, at), Radius: specScale(o), Mat: objMaterial(o),
		}}
	}
}

// specCenter is the object's world position (its declared point plus the region
// offset).
func specCenter(o brain.ObjSpec, at raytrace.Vec3) raytrace.Vec3 {
	return raytrace.Vec3{X: o.X + at.X, Y: o.Y + at.Y, Z: o.Z + at.Z}
}

// specHalf is the half-extents for boxes/pyramids/cylinders: S if given, else a
// cube/square of half-size R (default 1).
func specHalf(o brain.ObjSpec) raytrace.Vec3 {
	h := raytrace.Vec3{X: o.S[0], Y: o.S[1], Z: o.S[2]}
	if h.LenSq() == 0 {
		r := specScale(o)
		h = raytrace.Vec3{X: r, Y: r, Z: r}
	}
	return h
}

// specScale is R, defaulting to 1 (sphere radius / composite scale).
func specScale(o brain.ObjSpec) float64 {
	if o.R <= 0 {
		return 1
	}
	return o.R
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
