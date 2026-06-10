package fleet

// DemoScenario is a deterministic six-step run of a small Hyundai-style logistics
// fleet: two units overheat (a shared thermal cause -> a bridge), one starts to
// vibrate, the rest stay nominal. It is the shared fixture for the rayfleet
// showcase and the sim-to-real gates (gates.go).
func DemoScenario() []Reading {
	type unit struct {
		name  string
		base  []Signal
		spike string  // a signal that climbs in the last frames ("" = steady)
		rate  float64 // how fast it climbs
	}
	sig := func(name string, v, w float64) Signal { return Signal{Name: name, Value: v, Weight: w} }
	units := []unit{
		{"amr-1", []Signal{sig("vibration", 0.15, 1), sig("thermal", 0.2, 1), sig("poseDrift", 0.1, 0.8), sig("battery", 0.3, 0.6)}, "thermal", 0.3},
		{"amr-2", []Signal{sig("vibration", 0.2, 1), sig("thermal", 0.25, 1), sig("poseDrift", 0.12, 0.8), sig("battery", 0.35, 0.6)}, "vibration", 0.3},
		{"parkbot", []Signal{sig("vibration", 0.18, 1), sig("thermal", 0.5, 1), sig("poseDrift", 0.15, 0.8), sig("battery", 0.4, 0.6)}, "thermal", 0.3},
		{"dal-e", []Signal{sig("vibration", 0.1, 1), sig("thermal", 0.15, 1), sig("poseDrift", 0.08, 0.8), sig("battery", 0.6, 0.6)}, "", 0},
		{"atlas", []Signal{sig("vibration", 0.22, 1), sig("thermal", 0.3, 1), sig("poseDrift", 0.2, 0.8), sig("battery", 0.25, 0.6)}, "", 0},
	}
	var out []Reading
	for t := 0; t < 6; t++ {
		for _, u := range units {
			sigs := make([]Signal, len(u.base))
			copy(sigs, u.base)
			if u.spike != "" && t >= 4 {
				for i := range sigs {
					if sigs[i].Name == u.spike {
						sigs[i].Value += u.rate * float64(t-3)
					}
				}
			}
			out = append(out, Reading{Unit: u.name, T: float64(t), Signals: sigs})
		}
	}
	return out
}
