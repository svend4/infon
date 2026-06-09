package brain

import (
	"encoding/json"
	"math"
	"strings"
)

// SceneSpec is the tvcp-ai/1 "rayscene" payload: a brain-authored 3-D scene graph
// (objects with materials, a light and a sky). It is transport- and renderer-
// agnostic JSON; a renderer (see pkg/raydir) turns it into pixels. This is how a
// brain authors a whole 3-D world from a prompt, not just moves pieces.
type SceneSpec struct {
	Objects []ObjSpec  `json:"objects"`
	Light   [3]float64 `json:"light"`
	SkyTop  [3]float64 `json:"skyTop"`
	SkyBot  [3]float64 `json:"skyBottom"`
}

// ObjSpec is one object in a SceneSpec: a sphere (default), a box, or an infinite
// plane, with a material from the renderer's full set (colour, emission, glass
// index, metalness, mirror reflectance, roughness). For a box, S is the
// half-extents (falling back to a cube of half-size R); for a sphere, R is the
// radius.
type ObjSpec struct {
	Kind    string     `json:"kind"`
	Name    string     `json:"name"` // for kind "mesh": the named model to instance
	Tex     string     `json:"tex"`  // optional named surface texture (checker, marble, wood, stone, clouds, or a loaded image)
	X       float64    `json:"x"`
	Y       float64    `json:"y"`
	Z       float64    `json:"z"`
	R       float64    `json:"r"`
	S       [3]float64 `json:"s"` // box half-extents (x,y,z)
	Color   [3]float64 `json:"color"`
	Emit    [3]float64 `json:"emit"`
	Glass   float64    `json:"glass"`
	Metal   float64    `json:"metal"`
	Reflect float64    `json:"reflect"`
	Rough   float64    `json:"rough"`
}

// refRayScene is the reference scenographer for game "rayscene": it authors a
// small scene exercising the full material set, lightly varied by prompt keyword
// ("gold", "night"), so behaviour is reproducible without a real model.
func refRayScene(req Request) Response {
	var p struct {
		Prompt  string     `json:"prompt"`
		Index   int        `json:"index"`
		Heading [3]float64 `json:"heading"`
		Prev    *SceneSpec `json:"prev"`
	}
	_ = json.Unmarshal(req.State, &p)
	spec := SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: [3]float64{0.4, 0.55, 0.85}, SkyBot: [3]float64{0.85, 0.88, 0.95},
		Objects: []ObjSpec{
			{Kind: "plane", Color: [3]float64{0.8, 0.8, 0.8}},
			{X: -1.4, Y: 1, R: 1, Color: [3]float64{0.85, 0.25, 0.25}, Rough: 0.3},
			{X: 1.4, Y: 1, Z: 0.3, R: 1, Color: [3]float64{0.95, 0.95, 0.97}, Reflect: 0.9},
			{Y: 0.6, Z: 2, R: 0.6, Glass: 1.5},
			{Y: 6, Z: -1, R: 0.7, Emit: [3]float64{18, 18, 17}},
		},
	}
	// Continuity: a region connects to the place that came before it. Inherit the
	// previous region's sky (so adjacent regions share a sky, before any keyword
	// override), and lay a path of stepping stones leading back toward it along the
	// reverse of the walking heading — turning independent chunks into one journey.
	if p.Prev != nil {
		spec.SkyTop, spec.SkyBot = p.Prev.SkyTop, p.Prev.SkyBot
	}
	if hl := math.Hypot(p.Heading[0], p.Heading[2]); hl > 1e-6 {
		ux, uz := p.Heading[0]/hl, p.Heading[2]/hl
		for k := 1; k <= 4; k++ {
			d := 2.0 * float64(k)
			spec.Objects = append(spec.Objects, ObjSpec{
				Kind: "cylinder", X: -ux * d, Z: -uz * d, R: 0.4,
				S: [3]float64{0.4, 0.05, 0.4}, Color: [3]float64{0.6, 0.6, 0.62}, Rough: 0.6,
			})
		}
	}
	if strings.Contains(p.Prompt, "gold") {
		spec.Objects[1].Color = [3]float64{1, 0.78, 0.34}
		spec.Objects[1].Metal = 1
		spec.Objects[1].Rough = 0.15
	}
	if strings.Contains(p.Prompt, "night") {
		spec.SkyTop = [3]float64{0.02, 0.03, 0.06}
		spec.SkyBot = [3]float64{0.04, 0.04, 0.07}
	}
	if hasAny(p.Prompt, "box", "tower", "wall", "build", "pillar", "monolith") {
		// a recognizable structure, not just spheres.
		spec.Objects = append(spec.Objects, ObjSpec{
			Kind: "box", X: 0, Y: 1.5, Z: -1, S: [3]float64{0.8, 1.5, 0.8},
			Color: [3]float64{0.6, 0.6, 0.65}, Rough: 0.4,
		})
	}
	if hasAny(p.Prompt, "tree", "forest", "wood", "park") {
		spec.Objects = append(spec.Objects, ObjSpec{Kind: "tree", X: -3, Y: 0, Z: -1, R: 1.2})
	}
	if hasAny(p.Prompt, "house", "home", "village", "town", "cabin") {
		spec.Objects = append(spec.Objects, ObjSpec{Kind: "house", X: 3, Y: 0, Z: -1, R: 1.1})
	}
	if hasAny(p.Prompt, "crystal", "gem", "cave") {
		spec.Objects = append(spec.Objects, ObjSpec{Kind: "mesh", Name: "crystal", X: -3, Y: 1, Z: 1, R: 1})
	}
	if hasAny(p.Prompt, "rock", "stone", "boulder") {
		spec.Objects = append(spec.Objects, ObjSpec{Kind: "mesh", Name: "rock", X: 3, Y: 0.5, Z: 1, R: 1})
	}
	for _, tx := range []string{"marble", "wood", "checker", "stone", "clouds"} {
		if strings.Contains(p.Prompt, tx) { // texture the diffuse sphere on request
			spec.Objects[1].Tex = tx
			break
		}
	}
	data, _ := json.Marshal(spec)
	return Response{Protocol: Protocol, Kind: "move", Ray: data, Reasoning: "reference scene author"}
}

func hasAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
