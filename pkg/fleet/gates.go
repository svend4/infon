package fleet

// gates.go is the sim-to-real conformance harness — info150's triage4 evaluation
// gates (triage accuracy, classification F1, vitals MAE) repurposed to answer the
// Part 1 / variant-E question: does the monitoring engine, validated in
// simulation, reach the *same operational conclusions* on noisy real telemetry?
// The same scenario is run twice — once as clean "sim" readings, once perturbed by
// a sim-to-real GapModel — and the two runs' assessments are scored against gate
// thresholds. The per-unit baseline (the engine's own normal) is what lets it
// absorb a steady offset, so a modest gap still passes. Pure and deterministic.

// GapModel is the sim-to-real gap applied to each signal:
//
//	real = clamp01(sim*Scale + Bias + uniform(-Noise, +Noise))
//
// Scale defaults to 1; the noise is hashed from (unit, signal, time, Seed) so a
// gap is reproducible.
type GapModel struct {
	Scale float64
	Bias  float64
	Noise float64
	Seed  int64
}

// ApplyGap returns a perturbed copy of the readings (the "real" run).
func ApplyGap(readings []Reading, g GapModel) []Reading {
	scale := g.Scale
	if scale == 0 {
		scale = 1
	}
	out := make([]Reading, len(readings))
	for i, r := range readings {
		nr := Reading{Unit: r.Unit, T: r.T, Signals: make([]Signal, len(r.Signals))}
		for j, s := range r.Signals {
			n := 0.0
			if g.Noise != 0 {
				n = (hashUnit(r.Unit, s.Name, r.T, g.Seed)*2 - 1) * g.Noise
			}
			nr.Signals[j] = Signal{Name: s.Name, Value: clamp01(s.Value*scale + g.Bias + n), Weight: s.Weight}
		}
		out[i] = nr
	}
	return out
}

// Gate is one pass/fail criterion: Metric is compared to Thresh, passing if it is
// at least Thresh (Higher) or at most Thresh (!Higher).
type Gate struct {
	Name   string
	Metric float64
	Thresh float64
	Higher bool
	Pass   bool
}

// GateReport is the verdict of a sim-vs-real comparison.
type GateReport struct {
	Gates []Gate
	Pass  bool
}

// Compare runs the engine over the sim and the real readings and scores the four
// gates: level agreement, dominant-cause agreement, severity MAE, and bridge
// (shared-cause) agreement. Overall pass requires every gate.
func Compare(sim, real []Reading) GateReport {
	simAs := runLatest(sim)
	realAs := runLatest(real)
	gates := []Gate{
		gate("level-agreement", agreeFrac(simAs, realAs, func(a Assessment) string { return a.Level.String() }), 0.75, true),
		gate("cause-agreement", agreeFrac(simAs, realAs, func(a Assessment) string { return a.Worst }), 0.75, true),
		gate("severity-MAE", severityMAE(simAs, realAs), 0.15, false),
		gate("bridge-agreement", bridgeJaccard(simAs, realAs), 0.5, true),
	}
	all := true
	for _, g := range gates {
		if !g.Pass {
			all = false
		}
	}
	return GateReport{Gates: gates, Pass: all}
}

func gate(name string, metric, thresh float64, higher bool) Gate {
	pass := metric <= thresh
	if higher {
		pass = metric >= thresh
	}
	return Gate{Name: name, Metric: metric, Thresh: thresh, Higher: higher, Pass: pass}
}

// runLatest runs a fresh engine over the readings and returns the latest
// assessment per unit (its temporal memory accumulates across the earlier ones).
func runLatest(readings []Reading) map[string]Assessment {
	e := NewMonitoringEngine()
	out := map[string]Assessment{}
	for _, r := range readings {
		out[r.Unit] = e.Assess(r)
	}
	return out
}

// agreeFrac is the fraction of common units whose key matches between the runs.
func agreeFrac(a, b map[string]Assessment, key func(Assessment) string) float64 {
	common, match := 0, 0
	for u, av := range a {
		bv, ok := b[u]
		if !ok {
			continue
		}
		common++
		if key(av) == key(bv) {
			match++
		}
	}
	if common == 0 {
		return 1
	}
	return float64(match) / float64(common)
}

func severityMAE(a, b map[string]Assessment) float64 {
	common := 0
	sum := 0.0
	for u, av := range a {
		bv, ok := b[u]
		if !ok {
			continue
		}
		common++
		d := av.Severity - bv.Severity
		if d < 0 {
			d = -d
		}
		sum += d
	}
	if common == 0 {
		return 0
	}
	return sum / float64(common)
}

// bridgeJaccard is the Jaccard similarity of the two runs' shared-cause memberships
// ("cause|unit"); identical bridge structure (including none) scores 1.
func bridgeJaccard(a, b map[string]Assessment) float64 {
	sa := bridgeSet(a)
	sb := bridgeSet(b)
	if len(sa) == 0 && len(sb) == 0 {
		return 1
	}
	inter := 0
	for k := range sa {
		if sb[k] {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

func bridgeSet(m map[string]Assessment) map[string]bool {
	as := make([]Assessment, 0, len(m))
	for _, a := range m {
		as = append(as, a)
	}
	set := map[string]bool{}
	for cause, units := range Bridges(as) {
		for _, u := range units {
			set[cause+"|"+u] = true
		}
	}
	return set
}

// hashUnit is a deterministic uniform [0,1) from a unit, signal, time and seed.
func hashUnit(unit, sigName string, t float64, seed int64) float64 {
	h := uint64(1469598103934665603)
	mix := func(s string) {
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= 1099511628211
		}
	}
	mix(unit)
	mix(sigName)
	h ^= uint64(int64(t*1000)) * 2654435761
	h ^= uint64(seed) * 40503
	h ^= h >> 33
	return float64(h%1000000) / 1000000.0
}
