package raydir

// intention.go gives the walker the hackers' core art: intention (намерение). You
// hold a wish — a theme — and, the longer you hold it, the more strongly the world
// grown ahead bends toward it. A flicker changes little; a sustained intention
// manifests, the director building what you will into being. It composes with the
// mood bias (mood colours the tone; intention sets the subject).

// intentThresh and intentFull are the strengths at which an intention starts to
// colour, and fully manifests in, the world.
const (
	intentThresh = 0.33
	intentFull   = 0.66
	intentRate   = 0.25 // strength gained per second while held
)

// SetIntention sets (or changes) the held intention to a theme — a prompt
// fragment like "a forest of crystals". Changing it resets the strength.
func (w *World) SetIntention(theme string) {
	if theme != w.intent {
		w.intentStr = 0
	}
	w.intent = theme
}

// HoldIntention sustains the current intention for dt seconds, building its
// strength toward full manifestation.
func (w *World) HoldIntention(dt float64) {
	if w.intent == "" {
		return
	}
	w.intentStr = clampf(w.intentStr+dt*intentRate, 0, 1)
}

// ClearIntention lets the intention go.
func (w *World) ClearIntention() { w.intent, w.intentStr = "", 0 }

// Intention returns the held theme and its strength (0..1).
func (w *World) Intention() (string, float64) { return w.intent, w.intentStr }

// IntendPrompt bends a base director prompt by the held intention: nothing while
// it's a flicker, a colouring as it builds, and — once sustained — the theme takes
// over, so the world ahead becomes what you will.
func (w *World) IntendPrompt(base string) string {
	switch {
	case w.intent == "" || w.intentStr < intentThresh:
		return base
	case w.intentStr < intentFull:
		return base + ", " + w.intent
	default:
		return w.intent // manifested: your will is the world
	}
}
