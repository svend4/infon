package device

import "time"

// Pacer implements adaptive frame pacing (roadmap D3): it decouples a source's
// nominal FPS from how fast frames can actually be produced/rendered, and skips
// frames when the producer falls behind so playback stays smooth and on-time
// instead of drifting or backing up.
//
// Usage: call ShouldRender(now) on a fixed tick; it returns true when a frame is
// due. Report how long producing/rendering took with Observe so the pacer can
// adapt the effective interval under sustained load.
type Pacer struct {
	targetInterval time.Duration // 1/targetFPS
	minInterval    time.Duration // never render faster than this
	nextDue        time.Time
	emaWork        time.Duration // smoothed observed work time
	started        bool

	Rendered uint64
	Skipped  uint64
}

// NewPacer creates a pacer for targetFPS. maxFPS caps the fastest cadence (0 =
// no cap beyond targetFPS).
func NewPacer(targetFPS, maxFPS float64) *Pacer {
	if targetFPS <= 0 {
		targetFPS = 15
	}
	p := &Pacer{targetInterval: time.Duration(float64(time.Second) / targetFPS)}
	if maxFPS > 0 {
		p.minInterval = time.Duration(float64(time.Second) / maxFPS)
	}
	return p
}

// ShouldRender reports whether a frame is due at time now. The first call always
// returns true (and arms the schedule). When the producer is late, the schedule
// advances past missed slots (counting them as Skipped) so we don't render a
// burst of stale frames to "catch up".
func (p *Pacer) ShouldRender(now time.Time) bool {
	if !p.started {
		p.started = true
		p.nextDue = now.Add(p.effectiveInterval())
		p.Rendered++
		return true
	}
	if now.Before(p.nextDue) {
		return false
	}

	// We're at/after the due time. Advance the schedule by whole intervals; any
	// fully-missed slots are skipped rather than rendered late.
	interval := p.effectiveInterval()
	p.nextDue = p.nextDue.Add(interval)
	for !now.Before(p.nextDue) {
		p.Skipped++
		p.nextDue = p.nextDue.Add(interval)
	}
	p.Rendered++
	return true
}

// Observe records how long the last produce+render cycle took, so the pacer can
// widen the interval under sustained load (when work exceeds the target period,
// targeting ~the work time keeps cadence stable instead of perpetually late).
func (p *Pacer) Observe(work time.Duration) {
	// Exponential moving average to smooth spikes.
	if p.emaWork == 0 {
		p.emaWork = work
	} else {
		p.emaWork = (p.emaWork*7 + work) / 8
	}
}

// effectiveInterval is the target interval, widened toward the smoothed work
// time when work consistently overruns the target, and floored by minInterval.
func (p *Pacer) effectiveInterval() time.Duration {
	iv := p.targetInterval
	if p.emaWork > iv {
		iv = p.emaWork
	}
	if p.minInterval > 0 && iv < p.minInterval {
		iv = p.minInterval
	}
	return iv
}

// EffectiveFPS returns the current effective frames-per-second.
func (p *Pacer) EffectiveFPS() float64 {
	iv := p.effectiveInterval()
	if iv <= 0 {
		return 0
	}
	return float64(time.Second) / float64(iv)
}
