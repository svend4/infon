package raydir

import "math"

// mood.go lets the world read how you move and shift its tone to match. It is not
// a director that reads your mind — just a small, honest reading of behaviour: do
// you linger (slow, still), press on (fast, straight), or wander and look around
// (lots of turning)? From that it picks a mood — calm, restless, or curious — and
// biases the prompt handed to the director, so a world built for a dawdler is
// quiet and intimate while one built for a sprinter is vast and open. Smoothed
// (exponential moving averages), so a single twitch doesn't swing the tone.

// mood thresholds (on the smoothed estimates).
const (
	moodTurnBusy  = 0.25 // radians/second of turning above which you're "looking around"
	moodSpeedFast = 0.9  // units/second of travel above which you're "pressing on"
	moodEMA       = 0.1  // smoothing factor for the running estimates
)

// Mood tracks how the walker moves and classifies it.
type Mood struct {
	speedEMA float64 // smoothed walking speed (units/second)
	turnEMA  float64 // smoothed turn rate (radians/second)
	last     Pose
	has      bool
}

// NewMood returns a fresh mood tracker.
func NewMood() *Mood { return &Mood{} }

// angDelta is the smallest absolute angle between two yaws (handles wraparound).
func angDelta(a, b float64) float64 {
	d := math.Mod(a-b, 2*math.Pi)
	if d < -math.Pi {
		d += 2 * math.Pi
	} else if d > math.Pi {
		d -= 2 * math.Pi
	}
	return math.Abs(d)
}

// Observe folds a new pose (dt seconds after the previous) into the smoothed
// estimates of how fast you move and how much you turn.
func (m *Mood) Observe(p Pose, dt float64) {
	if dt <= 0 {
		return
	}
	if !m.has {
		m.last, m.has = p, true
		return
	}
	speed := p.Pos.Sub(m.last.Pos).Len() / dt
	turn := angDelta(p.Yaw, m.last.Yaw) / dt
	m.speedEMA = m.speedEMA*(1-moodEMA) + speed*moodEMA
	m.turnEMA = m.turnEMA*(1-moodEMA) + turn*moodEMA
	m.last = p
}

// Name classifies the current behaviour: "curious" (lots of looking around),
// "restless" (pressing forward), or "calm" (lingering).
func (m *Mood) Name() string {
	switch {
	case m.turnEMA > moodTurnBusy:
		return "curious"
	case m.speedEMA > moodSpeedFast:
		return "restless"
	default:
		return "calm"
	}
}

// Prompt returns a phrase biasing the director toward the current mood's tone.
// These words are understood by the reference author (and any live model).
func (m *Mood) Prompt() string {
	switch m.Name() {
	case "curious":
		return "strange, varied and surprising"
	case "restless":
		return "vast, grand and open"
	default:
		return "quiet, still and intimate"
	}
}
