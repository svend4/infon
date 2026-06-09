// scene.go lets a tvcp-ai/1 brain author a whole ray-traced scene, not just move
// spheres: it defines a JSON scene graph (objects with full materials, a light and
// a sky) and the bridge that turns a brain's response into a renderable
// raytrace.Scene. So "describe a 3-D scene" becomes pixels through the same path
// tracer used everywhere else. AuthorScene drives any brain.Brain; RefSceneBrain
// is a deterministic reference author that exercises the full material system
// (diffuse, mirror, metal, glass, emissive) so the bridge is testable offline.
package raydir

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// SceneSpec is the brain-authored scene graph.
type SceneSpec struct {
	Objects []ObjSpec  `json:"objects"`
	Light   [3]float64 `json:"light"`
	SkyTop  [3]float64 `json:"skyTop"`
	SkyBot  [3]float64 `json:"skyBottom"`
}

// ObjSpec is one object: a sphere (default) or an infinite plane, with a material
// drawn from the path tracer's full set (color, emission, glass IOR, metalness,
// mirror reflectance, roughness).
type ObjSpec struct {
	Kind    string     `json:"kind"` // "sphere" (default) or "plane"
	X       float64    `json:"x"`
	Y       float64    `json:"y"`
	Z       float64    `json:"z"`
	R       float64    `json:"r"` // sphere radius (ignored for plane)
	Color   [3]float64 `json:"color"`
	Emit    [3]float64 `json:"emit"`
	Glass   float64    `json:"glass"`
	Metal   float64    `json:"metal"`
	Reflect float64    `json:"reflect"`
	Rough   float64    `json:"rough"`
}

func vec3(a [3]float64) raytrace.Vec3 { return raytrace.Vec3{X: a[0], Y: a[1], Z: a[2]} }

func (o ObjSpec) material() raytrace.Material {
	return raytrace.Material{
		Color:   raytrace.Vec3{X: o.Color[0], Y: o.Color[1], Z: o.Color[2]},
		Emit:    raytrace.Vec3{X: o.Emit[0], Y: o.Emit[1], Z: o.Emit[2]},
		Glass:   o.Glass,
		Metal:   o.Metal,
		Reflect: o.Reflect,
		Rough:   o.Rough,
	}
}

// BuildScene turns a SceneSpec into a renderable raytrace.Scene.
func BuildScene(spec SceneSpec) *raytrace.Scene {
	s := &raytrace.Scene{
		Light:     vec3(spec.Light),
		SkyTop:    vec3(spec.SkyTop),
		SkyBottom: vec3(spec.SkyBot),
	}
	for _, o := range spec.Objects {
		switch o.Kind {
		case "plane":
			c := raytrace.Vec3{X: o.Color[0], Y: o.Color[1], Z: o.Color[2]}
			s.Objects = append(s.Objects, raytrace.Plane{Y: o.Y, Size: 1, C1: c, C2: c.Scale(0.6)})
		default:
			r := o.R
			if r <= 0 {
				r = 1
			}
			s.Objects = append(s.Objects, raytrace.Sphere{
				Center: raytrace.Vec3{X: o.X, Y: o.Y, Z: o.Z}, Radius: r, Mat: o.material(),
			})
		}
	}
	return s
}

// DecodeSceneSpec parses a SceneSpec JSON blob into a renderable scene.
func DecodeSceneSpec(data []byte) (*raytrace.Scene, SceneSpec, error) {
	var spec SceneSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, SceneSpec{}, err
	}
	return BuildScene(spec), spec, nil
}

// AuthorScene asks a brain to author a scene for the prompt (game "rayscene") and
// decodes the response into a renderable scene.
func AuthorScene(b brain.Brain, prompt string) (*raytrace.Scene, SceneSpec, error) {
	state, _ := json.Marshal(struct {
		Prompt string `json:"prompt"`
	}{prompt})
	resp, err := b.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "rayscene", State: state})
	if err != nil {
		return nil, SceneSpec{}, err
	}
	return DecodeSceneSpec(resp.Ray)
}

// RefSceneBrain is a deterministic reference scene author: it returns a small
// scene exercising the full material system, lightly varied by prompt keywords
// (so behaviour is reproducible without a real model).
type RefSceneBrain struct{}

// Decide implements brain.Brain for game "rayscene".
func (RefSceneBrain) Decide(req brain.Request) (brain.Response, error) {
	spec := SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: [3]float64{0.4, 0.55, 0.85}, SkyBot: [3]float64{0.85, 0.88, 0.95},
		Objects: []ObjSpec{
			{Kind: "plane", Y: 0, Color: [3]float64{0.8, 0.8, 0.8}},
			{X: -1.4, Y: 1, Z: 0, R: 1, Color: [3]float64{0.85, 0.25, 0.25}, Rough: 0.3},    // red diffuse
			{X: 1.4, Y: 1, Z: 0.3, R: 1, Color: [3]float64{0.95, 0.95, 0.97}, Reflect: 0.9}, // mirror
			{X: 0, Y: 0.6, Z: 2, R: 0.6, Glass: 1.5},                                        // glass
			{X: 0, Y: 6, Z: -1, R: 0.7, Emit: [3]float64{18, 18, 17}},                       // light
		},
	}
	// Independent keyword nudges so the author visibly responds to the prompt
	// (attributes combine: "gold night" gives both).
	var p struct {
		Prompt string `json:"prompt"`
	}
	_ = json.Unmarshal(req.State, &p)
	if contains(p.Prompt, "gold") {
		spec.Objects[1].Color = [3]float64{1.0, 0.78, 0.34}
		spec.Objects[1].Metal = 1
		spec.Objects[1].Rough = 0.15
	}
	if contains(p.Prompt, "night") {
		spec.SkyTop = [3]float64{0.02, 0.03, 0.06}
		spec.SkyBot = [3]float64{0.04, 0.04, 0.07}
		spec.Objects[4].Emit = [3]float64{30, 30, 28} // brighter key light at night
	}
	data, _ := json.Marshal(spec)
	return brain.Response{Protocol: brain.Protocol, Kind: "move", Ray: data, Reasoning: "reference scene author"}, nil
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
