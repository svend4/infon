// Package fleet is a decision-support engine for watching machines — robots,
// vehicles, equipment — the way info150's triage4 watches a casualty: ingest a
// few normalised observations, fuse them into a severity, track each unit against
// its own baseline so a sudden change escalates, and emit a short operator cue.
// It is the "camera observes -> analysis -> conclusion -> reaction" loop of
// triage4-drive (drowsiness / distraction / incapacitation -> dispatcher cue),
// repointed from a driver to a robot.
//
// The engine is clean-room and pure (stdlib only), so it is fully testable; the
// graph and scene helpers in graph.go turn its output into the existing bubble
// diagram (raydir.BubbleGraph) and a rayscene, so a fleet draws with the same
// renderer as the dream worlds. The wire bridge to any tvcp-ai/1 brain is
// ReactRequest (see ai/adapters/equipment_brain.py).
package fleet

import (
	"fmt"
	"sort"
	"strings"
)

// Signal is one normalised indicator for a unit: Value in [0,1] where higher is
// more concerning (PERCLOS for a driver, vibration or temperature for a machine).
// Weight is its relative importance when the signals are fused (defaults to 1).
type Signal struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Weight float64 `json:"weight,omitempty"`
}

// Reading is a snapshot of one unit's signals at time T (seconds).
type Reading struct {
	Unit    string   `json:"unit"`
	T       float64  `json:"t,omitempty"`
	Signals []Signal `json:"signals"`
}

// Level is the triage class of an assessment, ordered OK < Watch < Warn <
// Critical (green / amber / orange / red).
type Level int

// The four triage levels, ascending in concern.
const (
	LevelOK Level = iota
	LevelWatch
	LevelWarn
	LevelCritical
)

func (l Level) String() string {
	switch l {
	case LevelWatch:
		return "WATCH"
	case LevelWarn:
		return "WARN"
	case LevelCritical:
		return "CRITICAL"
	default:
		return "OK"
	}
}

// Assessment is the engine's conclusion for one reading.
type Assessment struct {
	Unit     string
	T        float64
	Severity float64 // fused 0..1
	Level    Level
	Worst    string  // the dominant signal's name (highest value*weight)
	WorstVal float64 // its value
	Baseline float64 // the unit's running baseline severity (its own normal)
	Trend    float64 // severity - baseline (>0 = deteriorating)
	Cue      string  // short operator advisory
	Explain  string  // every signal, sorted by contribution
}

// MonitoringEngine fuses signals and tracks each unit over time. Like
// triage4-drive's DriverMonitoringEngine it weights the signals and applies a
// per-unit baseline, so a unit that always runs a little warm does not cry wolf
// while a sudden jump above a quiet unit's own normal escalates. Stateful: each
// Assess folds the reading into that unit's baseline (an EWMA) — the engine's
// temporal memory.
type MonitoringEngine struct {
	watch, warn, crit float64 // severity thresholds, ascending
	trendKick         float64 // how strongly deterioration above baseline raises the level
	alpha             float64 // EWMA factor for the baseline (0..1; higher = faster)
	baseline          map[string]float64
	seen              map[string]bool
}

// NewMonitoringEngine returns an engine with sensible default thresholds.
func NewMonitoringEngine() *MonitoringEngine {
	return &MonitoringEngine{
		watch: 0.3, warn: 0.55, crit: 0.8,
		trendKick: 0.5,
		alpha:     0.25,
		baseline:  map[string]float64{},
		seen:      map[string]bool{},
	}
}

// Assess fuses a reading into an Assessment and folds it into the unit's
// baseline. The severity is a soft-OR — half the weighted mean of the signals,
// half the single worst signal — so an overall-quiet unit with one critical
// fault is still flagged (a robot with one critical reading IS critical). The
// level is chosen from the *effective* severity, which adds a fraction of any
// positive trend (deterioration above the unit's own baseline).
func (e *MonitoringEngine) Assess(r Reading) Assessment {
	a := Assessment{Unit: r.Unit, T: r.T}
	if len(r.Signals) == 0 {
		a.Cue = r.Unit + ": no signal"
		a.Explain = "no signals reported"
		e.seen[r.Unit] = true
		return a
	}
	var sum, wsum, maxVal, worstContrib float64
	for _, s := range r.Signals {
		w := s.Weight
		if w <= 0 {
			w = 1
		}
		v := clamp01(s.Value)
		sum += v * w
		wsum += w
		if v > maxVal {
			maxVal = v
		}
		if c := v * w; c > worstContrib {
			worstContrib, a.Worst, a.WorstVal = c, s.Name, v
		}
	}
	mean := 0.0
	if wsum > 0 {
		mean = sum / wsum
	}
	a.Severity = 0.5*mean + 0.5*maxVal

	base, ok := e.baseline[r.Unit]
	if !ok {
		base = a.Severity // first sight: baseline is the current value (no false alarm)
	}
	a.Baseline = base
	a.Trend = a.Severity - base
	eff := a.Severity
	if a.Trend > 0 {
		eff += a.Trend * e.trendKick
	}
	a.Level = e.level(eff)
	a.Cue = cue(a)
	a.Explain = explain(r)

	e.baseline[r.Unit] = base*(1-e.alpha) + a.Severity*e.alpha
	e.seen[r.Unit] = true
	return a
}

func (e *MonitoringEngine) level(eff float64) Level {
	switch {
	case eff >= e.crit:
		return LevelCritical
	case eff >= e.warn:
		return LevelWarn
	case eff >= e.watch:
		return LevelWatch
	default:
		return LevelOK
	}
}

// cue is a short operator advisory (kept terse, in the spirit of triage4-drive's
// dispatcher cues and biocore's SMS cap).
func cue(a Assessment) string {
	if a.Level == LevelOK {
		return a.Unit + ": nominal"
	}
	dir := ""
	switch {
	case a.Trend > 0.08:
		dir = ", rising"
	case a.Trend < -0.08:
		dir = ", easing"
	}
	return fmt.Sprintf("%s: %s %s %.0f%%%s", a.Unit, a.Level, a.Worst, a.WorstVal*100, dir)
}

// explain lists every signal as name=value, sorted by contribution (value*weight)
// then name — the engine's reasoning, like triage4's explainability endpoint.
func explain(r Reading) string {
	type kv struct {
		name string
		v, c float64
	}
	items := make([]kv, 0, len(r.Signals))
	for _, s := range r.Signals {
		w := s.Weight
		if w <= 0 {
			w = 1
		}
		v := clamp01(s.Value)
		items = append(items, kv{s.Name, v, v * w})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].c != items[j].c {
			return items[i].c > items[j].c
		}
		return items[i].name < items[j].name
	})
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s=%.2f", it.name, it.v))
	}
	return strings.Join(parts, " ")
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
