package brain

import (
	"encoding/json"
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

// ObjSpec is one object in a SceneSpec: a sphere (default) or an infinite plane,
// with a material from the renderer's full set (colour, emission, glass index,
// metalness, mirror reflectance, roughness).
type ObjSpec struct {
	Kind    string     `json:"kind"`
	X       float64    `json:"x"`
	Y       float64    `json:"y"`
	Z       float64    `json:"z"`
	R       float64    `json:"r"`
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
		Prompt string `json:"prompt"`
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
	if strings.Contains(p.Prompt, "gold") {
		spec.Objects[1].Color = [3]float64{1, 0.78, 0.34}
		spec.Objects[1].Metal = 1
		spec.Objects[1].Rough = 0.15
	}
	if strings.Contains(p.Prompt, "night") {
		spec.SkyTop = [3]float64{0.02, 0.03, 0.06}
		spec.SkyBot = [3]float64{0.04, 0.04, 0.07}
	}
	data, _ := json.Marshal(spec)
	return Response{Protocol: Protocol, Kind: "move", Ray: data, Reasoning: "reference scene author"}
}
