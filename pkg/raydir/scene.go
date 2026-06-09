// scene.go bridges a tvcp-ai/1 brain's scene authoring (game "rayscene") to the
// renderer: it turns a brain.SceneSpec (the protocol's JSON scene graph of objects
// with full materials, a light and a sky) into a renderable raytrace.Scene. So
// "describe a 3-D scene" becomes pixels through the same path tracer used
// everywhere else. AuthorScene drives any brain.Brain; RefSceneBrain delegates to
// the protocol reference author, so the bridge is testable offline.
package raydir

import (
	"encoding/json"
	"math"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func vec3(a [3]float64) raytrace.Vec3 { return raytrace.Vec3{X: a[0], Y: a[1], Z: a[2]} }

func objMaterial(o brain.ObjSpec) raytrace.Material {
	m := raytrace.Material{
		Color:   vec3(o.Color),
		Emit:    vec3(o.Emit),
		Glass:   o.Glass,
		Metal:   o.Metal,
		Reflect: o.Reflect,
		Rough:   o.Rough,
	}
	if o.Tex != "" { // a named texture overrides the flat colour where known
		if t := TextureFor(o.Tex); t != nil {
			m.Tex = t
		}
	}
	return m
}

// objectsFromSpec turns one ObjSpec into renderable objects, offset by `at`. A box
// becomes 12 triangles; a sphere one analytic sphere; a plane the shared floor
// (only when includePlane is set). Untrusted specs from a live brain are sanitised
// here: an unknown kind or a non-finite coordinate drops the object, and sizes,
// emission and colours are clamped — so a sloppy or adversarial model cannot crash
// the renderer or corrupt the world.
func objectsFromSpec(o brain.ObjSpec, at raytrace.Vec3, includePlane bool) []raytrace.Object {
	if !validObj(o) {
		return nil
	}
	o = clampObj(o)
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
	case "mesh":
		m := meshFor(o.Name)
		if m == nil {
			return nil // unknown model
		}
		xf := raytrace.Translate(specCenter(o, at)).Mul(raytrace.ScaleUniform(specScale(o)))
		if specHasMaterial(o) { // author tinted/retextured this placement
			return []raytrace.Object{raytrace.NewInstanceMat(m, xf, objMaterial(o))}
		}
		return []raytrace.Object{raytrace.NewInstance(m, xf)}
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

// specHasMaterial reports whether the spec set any material field, i.e. whether a
// "mesh" placement wants to override the model's own baked material with its own.
func specHasMaterial(o brain.ObjSpec) bool {
	return o.Color != [3]float64{} || o.Emit != [3]float64{} ||
		o.Glass != 0 || o.Metal != 0 || o.Reflect != 0 || o.Rough != 0 ||
		(o.Tex != "" && TextureFor(o.Tex) != nil)
}

// maxRegionObjects caps how many objects one authored scene may contribute, so a
// runaway model can't flood the renderer.
const maxRegionObjects = 256

var knownKinds = map[string]bool{
	"": true, "sphere": true, "box": true, "pyramid": true,
	"cylinder": true, "tree": true, "house": true, "plane": true, "mesh": true,
}

func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) && math.Abs(f) < 1e6 }

// validObj rejects an object with an unknown kind or a non-finite coordinate.
func validObj(o brain.ObjSpec) bool {
	if !knownKinds[o.Kind] {
		return false
	}
	if o.Kind == "mesh" && meshFor(o.Name) == nil {
		return false // unknown named model: drop it
	}
	return finite(o.X) && finite(o.Y) && finite(o.Z) && finite(o.R) &&
		finite(o.S[0]) && finite(o.S[1]) && finite(o.S[2])
}

// clampObj clamps sizes, emission, colour and material weights into sane ranges.
func clampObj(o brain.ObjSpec) brain.ObjSpec {
	o.R = clampf(o.R, 0, 100)
	for i := range o.S {
		o.S[i] = clampf(o.S[i], 0, 100)
	}
	for i := range o.Emit {
		o.Emit[i] = clampf(o.Emit[i], 0, 300)
	}
	for i := range o.Color {
		o.Color[i] = clampf(o.Color[i], 0, 1)
	}
	o.Glass = clampf(o.Glass, 0, 5)
	o.Metal = clampf(o.Metal, 0, 1)
	o.Reflect = clampf(o.Reflect, 0, 1)
	o.Rough = clampf(o.Rough, 0, 1)
	return o
}

// hasRenderable reports whether the spec has at least one valid, non-plane object
// (so an authored region is never empty/black).
func hasRenderable(spec brain.SceneSpec) bool {
	for _, o := range spec.Objects {
		if o.Kind != "plane" && validObj(o) {
			return true
		}
	}
	return false
}

// fallbackSpec is a minimal safe region used when a brain returns unusable output,
// so the world keeps growing visibly instead of stalling.
func fallbackSpec() brain.SceneSpec {
	return brain.SceneSpec{
		Objects: []brain.ObjSpec{
			{X: 0, Y: 1, Z: 0, R: 1, Color: [3]float64{0.6, 0.6, 0.65}, Rough: 0.4},
			{X: 0, Y: 6, Z: -1, R: 0.6, Emit: [3]float64{16, 16, 15}},
		},
	}
}

// BuildScene turns a brain.SceneSpec into a renderable raytrace.Scene.
func BuildScene(spec brain.SceneSpec) *raytrace.Scene {
	s := &raytrace.Scene{
		Light:     vec3(spec.Light),
		SkyTop:    vec3(spec.SkyTop),
		SkyBottom: vec3(spec.SkyBot),
	}
	for i, o := range spec.Objects {
		if i >= maxRegionObjects {
			break
		}
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

// SceneContext carries continuity hints to the director so authored regions
// connect into a coherent place rather than independent islands: the region index,
// the walking heading (world direction from the previous region to this one), and
// the previous region's spec. All optional — a zero context just authors fresh.
type SceneContext struct {
	Index   int
	Heading raytrace.Vec3
	Prev    *brain.SceneSpec
}

// AuthorScene asks a brain to author a scene for the prompt (game "rayscene"). It
// is robust to a live model: a transport error is returned, but unusable model
// output (bad JSON, or nothing renderable) becomes a safe fallback region so the
// shared world never stalls or breaks.
func AuthorScene(b brain.Brain, prompt string) (*raytrace.Scene, brain.SceneSpec, error) {
	return AuthorSceneCtx(b, prompt, SceneContext{})
}

// AuthorSceneCtx is AuthorScene with continuity context (index, heading, previous
// region) passed to the director, so it can extend the world coherently.
func AuthorSceneCtx(b brain.Brain, prompt string, ctx SceneContext) (*raytrace.Scene, brain.SceneSpec, error) {
	state, _ := json.Marshal(struct {
		Prompt  string           `json:"prompt"`
		Index   int              `json:"index"`
		Heading [3]float64       `json:"heading"`
		Prev    *brain.SceneSpec `json:"prev,omitempty"`
	}{prompt, ctx.Index, [3]float64{ctx.Heading.X, ctx.Heading.Y, ctx.Heading.Z}, ctx.Prev})
	resp, err := b.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "rayscene", State: state})
	if err != nil {
		return nil, brain.SceneSpec{}, err
	}
	var spec brain.SceneSpec
	if json.Unmarshal(resp.Ray, &spec) != nil || !hasRenderable(spec) {
		spec = fallbackSpec()
	}
	return BuildScene(spec), spec, nil
}

// RefSceneBrain is a deterministic reference scene author: it delegates to the
// protocol's reference brain (game "rayscene"), so the same scene the conformance
// battery exercises is what the bridge renders.
type RefSceneBrain struct{}

// Decide implements brain.Brain for game "rayscene".
func (RefSceneBrain) Decide(req brain.Request) (brain.Response, error) {
	return brain.Reference(req), nil
}
