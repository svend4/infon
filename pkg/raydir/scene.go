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

// BuildScene turns a brain.SceneSpec into a renderable raytrace.Scene.
func BuildScene(spec brain.SceneSpec) *raytrace.Scene {
	s := &raytrace.Scene{
		Light:     vec3(spec.Light),
		SkyTop:    vec3(spec.SkyTop),
		SkyBottom: vec3(spec.SkyBot),
	}
	for _, o := range spec.Objects {
		switch o.Kind {
		case "plane":
			c := vec3(o.Color)
			s.Objects = append(s.Objects, raytrace.Plane{Y: o.Y, Size: 1, C1: c, C2: c.Scale(0.6)})
		default:
			r := o.R
			if r <= 0 {
				r = 1
			}
			s.Objects = append(s.Objects, raytrace.Sphere{
				Center: raytrace.Vec3{X: o.X, Y: o.Y, Z: o.Z}, Radius: r, Mat: objMaterial(o),
			})
		}
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
