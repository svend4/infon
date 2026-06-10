package fleet

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/raydir"
)

// one signal of the given value (weight 1) — severity then equals that value.
func one(name string, v float64) []Signal { return []Signal{{Name: name, Value: v}} }

func TestAssessLevelsByMagnitude(t *testing.T) {
	e := NewMonitoringEngine()
	low := e.Assess(Reading{Unit: "a", Signals: []Signal{{"x", 0.05, 1}, {"y", 0.05, 1}}})
	if low.Level != LevelOK {
		t.Errorf("all-quiet should be OK, got %s (sev %.2f)", low.Level, low.Severity)
	}
	hi := e.Assess(Reading{Unit: "b", Signals: []Signal{{"x", 0.95, 1}, {"y", 0.95, 1}}})
	if hi.Level != LevelCritical {
		t.Errorf("all-alarm should be CRITICAL, got %s (sev %.2f)", hi.Level, hi.Severity)
	}
}

// Soft-OR: one critical signal among quiet ones must raise severity above a unit
// that is uniformly moderate — a single bad reading still alarms.
func TestSoftOR(t *testing.T) {
	e := NewMonitoringEngine()
	spike := e.Assess(Reading{Unit: "spike", Signals: []Signal{{"a", 1.0, 1}, {"b", 0, 1}, {"c", 0, 1}}})
	flat := e.Assess(Reading{Unit: "flat", Signals: []Signal{{"a", 0.3, 1}, {"b", 0.3, 1}, {"c", 0.3, 1}}})
	if spike.Severity <= flat.Severity {
		t.Errorf("one critical signal (sev %.2f) should exceed uniform moderate (sev %.2f)", spike.Severity, flat.Severity)
	}
	if spike.Worst != "a" {
		t.Errorf("worst signal should be the spiking one, got %q", spike.Worst)
	}
}

func TestWorstByContribution(t *testing.T) {
	e := NewMonitoringEngine()
	a := e.Assess(Reading{Unit: "u", Signals: []Signal{
		{"low_heavy", 0.4, 3},  // contribution 1.2
		{"high_light", 0.9, 1}, // contribution 0.9
	}})
	if a.Worst != "low_heavy" {
		t.Errorf("worst should be highest value*weight, got %q", a.Worst)
	}
}

// Deterioration: the same final severity escalates further when it is reached by
// climbing from a quiet baseline than when it is steady — the trend kick.
func TestDeteriorationEscalates(t *testing.T) {
	flatEng := NewMonitoringEngine()
	var flat Assessment
	for i := 0; i < 5; i++ { // steady at 0.5
		flat = flatEng.Assess(Reading{Unit: "f", Signals: one("v", 0.5)})
	}
	climbEng := NewMonitoringEngine()
	for i := 0; i < 4; i++ { // quiet, settle baseline near 0.2
		climbEng.Assess(Reading{Unit: "c", Signals: one("v", 0.2)})
	}
	climb := climbEng.Assess(Reading{Unit: "c", Signals: one("v", 0.5)}) // same 0.5, but rising

	if flat.Severity != climb.Severity {
		t.Fatalf("test setup: severities should match (%.3f vs %.3f)", flat.Severity, climb.Severity)
	}
	if climb.Level <= flat.Level {
		t.Errorf("a rising unit (%s) should out-rank a steady one (%s) at equal severity", climb.Level, flat.Level)
	}
	if climb.Trend <= 0 {
		t.Errorf("climbing unit should have positive trend, got %.3f", climb.Trend)
	}
	if !strings.Contains(climb.Cue, "rising") {
		t.Errorf("rising unit's cue should say rising, got %q", climb.Cue)
	}
}

// A unit that is high on first sight is not reported as "rising" — there is no
// history to deteriorate from (baseline starts at the current value).
func TestFirstSightNotRising(t *testing.T) {
	e := NewMonitoringEngine()
	a := e.Assess(Reading{Unit: "new", Signals: one("v", 0.9)})
	if a.Trend != 0 {
		t.Errorf("first sight should have zero trend, got %.3f", a.Trend)
	}
	if strings.Contains(a.Cue, "rising") {
		t.Errorf("first-sight cue should not say rising, got %q", a.Cue)
	}
	if a.Level != LevelCritical {
		t.Errorf("first-sight 0.9 should still be CRITICAL by magnitude, got %s", a.Level)
	}
}

func TestCueIsShort(t *testing.T) {
	e := NewMonitoringEngine()
	for _, v := range []float64{0.0, 0.4, 0.7, 1.0} {
		a := e.Assess(Reading{Unit: "longunitname-parkbot-07", Signals: one("vibration", v)})
		if len(a.Cue) > 80 {
			t.Errorf("cue too long (%d): %q", len(a.Cue), a.Cue)
		}
	}
}

func TestEmptyReading(t *testing.T) {
	e := NewMonitoringEngine()
	a := e.Assess(Reading{Unit: "ghost"})
	if a.Level != LevelOK || !strings.Contains(a.Cue, "no signal") {
		t.Errorf("empty reading should be OK/no signal, got %s %q", a.Level, a.Cue)
	}
}

func TestBridgesCoOccurrence(t *testing.T) {
	as := []Assessment{
		{Unit: "amr-1", Worst: "thermal", Level: LevelWarn},
		{Unit: "parkbot", Worst: "thermal", Level: LevelCritical},
		{Unit: "amr-2", Worst: "vibration", Level: LevelWarn},
		{Unit: "dal-e", Worst: "thermal", Level: LevelOK}, // OK: excluded
	}
	br := Bridges(as)
	if len(br) != 1 {
		t.Fatalf("expected one shared cause, got %v", br)
	}
	got := br["thermal"]
	if len(got) != 2 || got[0] != "amr-1" || got[1] != "parkbot" {
		t.Errorf("thermal bridge should be [amr-1 parkbot], got %v", got)
	}
	if _, ok := br["vibration"]; ok {
		t.Error("a lone vibration unit should not form a bridge")
	}
}

func TestFleetGraphLinksSharedCause(t *testing.T) {
	as := []Assessment{
		{Unit: "amr-1", Worst: "thermal", Level: LevelWarn, Severity: 0.6},
		{Unit: "parkbot", Worst: "thermal", Level: LevelCritical, Severity: 0.8},
		{Unit: "amr-2", Worst: "vibration", Level: LevelWarn, Severity: 0.5},
	}
	g := FleetGraph(as)
	if g.Len() != 3 {
		t.Fatalf("expected 3 bubbles, got %d", g.Len())
	}
	// home is the most severe unit (parkbot)
	home, _ := g.Get(g.Home())
	if !strings.HasPrefix(home.Name, "parkbot") {
		t.Errorf("home should be the most severe unit, got %q", home.Name)
	}
	// the two thermal units bridge; the vibration one is isolated.
	byName := map[string]int{}
	for _, b := range g.Bubbles() {
		byName[strings.Fields(b.Name)[0]] = b.ID
	}
	linked := func(a, b int) bool {
		for _, n := range g.Neighbors(a) {
			if n == b {
				return true
			}
		}
		return false
	}
	if !linked(byName["amr-1"], byName["parkbot"]) {
		t.Error("amr-1 and parkbot share thermal and should bridge")
	}
	if len(g.Neighbors(byName["amr-2"])) != 0 {
		t.Errorf("amr-2 has a unique cause and should be isolated, got %v", g.Neighbors(byName["amr-2"]))
	}
}

func TestSceneFromAssessmentsValidAndRenders(t *testing.T) {
	as := []Assessment{
		{Unit: "ok", Severity: 0.1, Level: LevelOK},
		{Unit: "crit", Severity: 0.85, Level: LevelCritical},
	}
	spec := SceneFromAssessments(as)
	if len(spec.Objects) != 2+2*len(as) { // plane + (body+light)*n + sun
		t.Fatalf("expected %d objects, got %d", 2+2*len(as), len(spec.Objects))
	}
	if !strings.Contains(spec.Name, "worst CRITICAL") {
		t.Errorf("name should report the worst level, got %q", spec.Name)
	}
	emits := 0
	for _, o := range spec.Objects {
		for _, c := range o.Color {
			if c < 0 || c > 1 {
				t.Errorf("colour out of range: %v", o.Color)
			}
		}
		if o.Emit != [3]float64{} {
			emits++
		}
	}
	if emits < 2 { // the critical unit's alarm light + the sun
		t.Errorf("a critical fleet should have an emitting alarm light plus the sun, got %d emitters", emits)
	}
	// the authored scene must actually render (tiny, cheap).
	scene := raydir.BuildScene(spec)
	if scene == nil || len(scene.Objects) == 0 {
		t.Fatal("scene did not build")
	}
}

func TestReactRequestWellFormed(t *testing.T) {
	as := []Assessment{{Unit: "amr-1", Severity: 0.6, Level: LevelWarn, Worst: "thermal"}}
	req := ReactRequest(as)
	if req.Kind != "move" || req.Game != "rayscene" || req.Protocol == "" {
		t.Errorf("react request envelope wrong: %+v", req)
	}
	if !strings.Contains(string(req.State), "amr-1") || !strings.Contains(string(req.State), "thermal") {
		t.Errorf("state should carry the fleet, got %s", req.State)
	}
}
