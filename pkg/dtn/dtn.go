// Package dtn is a delay-tolerant ("deep space") profile for the semantic stream:
// each semvideo packet is a BUNDLE that is stored and forwarded only during
// intermittent CONTACT windows (a satellite pass, a planetary line-of-sight), and
// is dropped if it waits longer than its time-to-live. Simulate schedules which
// bundles get through; the surviving keyframes still let a receiver reconstruct
// the world — the audit's "video for DTN/Bundle Protocol" as a pure model.
package dtn

import "sort"

// Contact is a window [Start, End) during which the link is up (tick units).
type Contact struct {
	Start, End int
}

// Report is the outcome of a store-and-forward schedule.
type Report struct {
	Total       int
	Delivered   int
	Expired     int
	AvgLatency  float64 // mean ticks between a bundle's birth and its delivery
	Mask        []bool  // Mask[i] = bundle i was delivered
	DeliverTime []int   // DeliverTime[i] = tick delivered, or -1
}

// Simulate moves n bundles (bundle i is born at tick i) across a schedule of
// contacts: at each contact it first expires bundles older than ttl, then
// forwards up to perPass of the oldest waiting bundles. Deterministic.
func Simulate(n int, contacts []Contact, ttl, perPass int) Report {
	cs := append([]Contact(nil), contacts...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].Start < cs[j].Start })

	mask := make([]bool, n)
	dt := make([]int, n)
	for i := range dt {
		dt[i] = -1
	}
	next := 0 // oldest not-yet-resolved bundle (born order = FIFO)
	expired, latSum, delivered := 0, 0, 0

	for _, c := range cs {
		// drop bundles too old to ride even this contact (and any later one).
		for next < n && !mask[next] && next+ttl < c.Start {
			expired++
			next++
		}
		// forward up to perPass already-born bundles.
		capLeft := perPass
		for capLeft > 0 && next < n && next <= c.Start {
			mask[next] = true
			dt[next] = c.Start
			latSum += c.Start - next
			delivered++
			next++
			capLeft--
		}
	}
	// anything still unresolved never met a contact.
	for ; next < n; next++ {
		if !mask[next] {
			expired++
		}
	}

	avg := 0.0
	if delivered > 0 {
		avg = float64(latSum) / float64(delivered)
	}
	return Report{Total: n, Delivered: delivered, Expired: expired, AvgLatency: avg, Mask: mask, DeliverTime: dt}
}
