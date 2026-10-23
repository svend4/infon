package fleet

import (
	"encoding/json"
	"math"
	"testing"
)

// Huge (but JSON-legal) or non-finite weights, and NaN values, must not turn the
// fused severity into NaN: every threshold reads NaN as "OK", so a unit with all
// signals at maximum would be reported as nominal and its baseline poisoned.
func TestQualityMiscAudit_AssessExtremeWeights(t *testing.T) {
	var rs []Reading
	if err := json.Unmarshal([]byte(`[{"unit":"bot","signals":[
		{"name":"vibration","value":1,"weight":1e308},
		{"name":"temperature","value":1,"weight":1e308}]}]`), &rs); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		r    Reading
	}{
		{"json-1e308", rs[0]},
		{"inf", Reading{Unit: "bot", Signals: []Signal{{"a", 1, math.Inf(1)}, {"b", 1, math.Inf(1)}}}},
		{"nan-weight", Reading{Unit: "bot", Signals: []Signal{{"a", 1, math.NaN()}, {"b", 1, 1}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewMonitoringEngine()
			a := e.Assess(tc.r)
			if math.IsNaN(a.Severity) || a.Severity < 0 || a.Severity > 1 {
				t.Fatalf("Severity = %v, want within 0..1", a.Severity)
			}
			if a.Level != LevelCritical {
				t.Errorf("Level = %s (cue %q) with every signal at 1.0, want CRITICAL", a.Level, a.Cue)
			}
			// The baseline must stay usable: a later quiet reading reads as OK.
			b := e.Assess(Reading{Unit: "bot", Signals: []Signal{{"a", 0, 1}}})
			if math.IsNaN(b.Baseline) || math.IsNaN(b.Severity) {
				t.Errorf("baseline poisoned after an extreme-weight reading: %+v", b)
			}
		})
	}

	// A NaN value (not expressible in JSON, but possible from Go callers) must
	// not poison the severity either.
	a := NewMonitoringEngine().Assess(Reading{Unit: "n", Signals: []Signal{{"a", math.NaN(), 1}, {"b", 0.9, 1}}})
	if math.IsNaN(a.Severity) {
		t.Errorf("Severity = NaN for a NaN signal value")
	}
}
