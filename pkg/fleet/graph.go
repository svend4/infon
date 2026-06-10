package fleet

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

// Bridges groups units that share the same dominant concern (and are at least at
// WATCH) by that signal — the "co-occurrence" relationship that info150's portal
// surfaces across siblings none can see alone. Two robots both overheating are a
// shared cause (a hot room), not two coincidences. Only causes shared by two or
// more units are returned; each value is the unit names, sorted.
func Bridges(as []Assessment) map[string][]string {
	by := map[string][]string{}
	for _, a := range as {
		if a.Level >= LevelWatch && a.Worst != "" {
			by[a.Worst] = append(by[a.Worst], a.Unit)
		}
	}
	for k, v := range by {
		if len(v) < 2 {
			delete(by, k)
			continue
		}
		sort.Strings(v)
	}
	return by
}

// FleetGraph builds the relationship graph over assessed units and returns the
// existing *raydir.BubbleGraph, so the fleet draws with the same BubbleMap / SVG
// used for the dream worlds (and the same Q6 bubble layout). Units bridge when
// they share a dominant concern (see Bridges); the most severe unit is home (the
// anchor of the diagram). Adding order is by severity then name, so the layout is
// deterministic.
func FleetGraph(as []Assessment) *raydir.BubbleGraph {
	g := raydir.NewBubbleGraph()
	if len(as) == 0 {
		return g
	}
	srt := append([]Assessment(nil), as...)
	sort.Slice(srt, func(i, j int) bool {
		if srt[i].Severity != srt[j].Severity {
			return srt[i].Severity > srt[j].Severity
		}
		return srt[i].Unit < srt[j].Unit
	})
	id := map[string]int{}
	for _, a := range srt {
		id[a.Unit] = g.Add(fmt.Sprintf("%s %s", a.Unit, a.Level), raytrace.Vec3{}, brain.SceneSpec{})
	}
	for _, units := range Bridges(as) {
		for i := 0; i < len(units); i++ {
			for j := i + 1; j < len(units); j++ {
				g.Link(id[units[i]], id[units[j]])
			}
		}
	}
	g.Layout(220)
	return g
}

// SceneFromAssessments authors a rayscene that shows the fleet: each unit is a
// small machine (a metal body topped by a status light) coloured green (calm) to
// red (critical) by severity, and a unit at WARN or worse has its light *emit* —
// the alarm halo that the VFX pass (bloom + dream) blooms into a glow. This is the
// engine's reaction made visible, in the spirit of triage4 turning an assessment
// into a handoff. Authored independently of equipment_brain.py: they meet at the
// rayscene data, not in code.
func SceneFromAssessments(as []Assessment) brain.SceneSpec {
	spec := brain.SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: [3]float64{0.16, 0.18, 0.26}, SkyBot: [3]float64{0.4, 0.42, 0.5},
		Objects: []brain.ObjSpec{{Kind: "plane", Color: [3]float64{0.3, 0.31, 0.34}}},
	}
	n := len(as)
	worst := LevelOK
	for i, a := range as {
		x := float64(i)*2.2 - float64(n-1)*1.1
		col := severityColor(a.Severity)
		spec.Objects = append(spec.Objects, brain.ObjSpec{
			Kind: "box", X: x, Y: 0.6, Z: 2, S: [3]float64{0.5, 0.6, 0.5},
			Color: [3]float64{0.5, 0.52, 0.55}, Metal: 0.6, Rough: 0.3,
		})
		light := brain.ObjSpec{X: x, Y: 1.5, Z: 2, R: 0.32, Color: col}
		if a.Level >= LevelWarn {
			s := 2.0 + 6.0*a.Severity
			light.Emit = [3]float64{col[0] * s, col[1] * s, col[2] * s}
		}
		spec.Objects = append(spec.Objects, light)
		if a.Level > worst {
			worst = a.Level
		}
	}
	spec.Objects = append(spec.Objects, brain.ObjSpec{Y: 6, Z: -1, R: 0.7, Emit: [3]float64{14, 14, 13}})
	spec.Name = fmt.Sprintf("fleet: %d units, worst %s", n, worst)
	return spec
}

// severityColor maps a severity 0..1 to green (calm) -> amber -> red (critical).
func severityColor(s float64) [3]float64 {
	s = clamp01(s)
	if s < 0.5 {
		t := s / 0.5
		return [3]float64{0.2 + 0.8*t, 0.8, 0.2}
	}
	t := (s - 0.5) / 0.5
	return [3]float64{1.0, 0.8 * (1 - t), 0.15}
}

// ReactRequest packages the fleet's state as a tvcp-ai/1 "rayscene" request, so
// any brain that speaks the protocol can author the reaction scene — the same
// bridge yijing_brain uses, pointed at a fleet instead of a prompt.
func ReactRequest(as []Assessment) brain.Request {
	type us struct {
		Unit     string  `json:"unit"`
		Severity float64 `json:"severity"`
		Level    string  `json:"level"`
		Worst    string  `json:"worst"`
	}
	units := make([]us, 0, len(as))
	for _, a := range as {
		units = append(units, us{a.Unit, a.Severity, a.Level.String(), a.Worst})
	}
	state, _ := json.Marshal(struct {
		Prompt string `json:"prompt"`
		Fleet  []us   `json:"fleet"`
	}{"fleet status", units})
	return brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "rayscene", State: state}
}
