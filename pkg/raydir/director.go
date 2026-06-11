package raydir

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand"

	"github.com/svend4/infon/pkg/brain"
)

// director.go is the two halves of meaning-driven direction. The forward half is a
// LEARNING director: it does not know what an audience likes, so it proposes a
// world, watches an engagement signal (how long the viewer lingers — dwell — blended
// with mood), and climbs toward worlds that hold attention, converging on the
// viewer's taste. The inverse half is the READER: given a finished scene it recovers
// the six Q6 coordinates that would author it, and from them the hexagram and the
// mood the scene carries — meaning read back out of a world.
//
// Both turn on one continuous control space: SceneVector, the six Q6 floats (the
// same axes the Python pro2 director, yijing_brain v2, uses), so a vector means the
// same thing on both sides of tvcp-ai/1. SceneSpec authors the world; ReadScene is
// its exact inverse.

// Axis indices into a SceneVector.
const (
	AxFog = iota
	AxWarm
	AxDensity
	AxSun
	AxScale
	AxGlow
)

// SceneVector is six continuous Q6 coordinates (each 0..1) that bend a whole world:
// fog, palette warmth, entity density, day/night (sun), object scale and glow. It is
// the bridge between the symbolic hexagram (the sign of each axis) and a concrete
// rayscene.
type SceneVector [6]float64

// Hexagram is the hexagram implied by the vector's signs (>= 0.5 -> yang), bit 0 the
// bottom line — the same rule as the model's signs_to_hexagram.
func (v SceneVector) Hexagram() Hexagram {
	var h Hexagram
	for i := 0; i < 6; i++ {
		h.Lines[i] = v[i] >= 0.5
	}
	return h
}

// VectorFromHexagram places a vector at the centre of a hexagram's orthant (yang ->
// 0.75, yin -> 0.25): a canonical continuous scene for a discrete hexagram.
func VectorFromHexagram(h Hexagram) SceneVector {
	var v SceneVector
	for i := 0; i < 6; i++ {
		if h.Lines[i] {
			v[i] = 0.75
		} else {
			v[i] = 0.25
		}
	}
	return v
}

// VectorFromPrompt derives six Q6 coordinates from a prompt by hashing it: sha256 of
// the prompt, its first six bytes scaled to 0..1. This is the deterministic fallback
// the pro2 director uses when no embedding model is present, and it matches
// yijing_brain.vector_from_prompt exactly — so the offline Go author and the Python
// adapter agree on the world a prompt makes.
func VectorFromPrompt(prompt string) SceneVector {
	d := sha256.Sum256([]byte(prompt))
	var v SceneVector
	for i := 0; i < 6; i++ {
		v[i] = float64(d[i]) / 255.0
	}
	return v
}

// SceneSpec authors a rayscene whose every parameter is bent by the six floats — the
// Go twin of yijing_brain.scene_from_vector, rounded identically so Go and the model
// produce the same world from the same vector.
func (v SceneVector) SceneSpec() brain.SceneSpec {
	fog, warm, dens, sun, scale, glow := v[AxFog], v[AxWarm], v[AxDensity], v[AxSun], v[AxScale], v[AxGlow]
	day := sun
	cool := [3]float64{0.28 + 0.16*day, 0.44 + 0.18*day, 0.68 + 0.28*day}
	warmc := [3]float64{0.12 + 0.7*day, 0.10 + 0.32*day, 0.07 + 0.16*day}
	top := lerp3(cool, warmc, warm)
	var bot [3]float64
	for i := range bot {
		bot[i] = math.Min(1, top[i]+0.2)
	}
	g := (top[0] + top[1] + top[2]) / 3
	top = lerp3(top, [3]float64{g, g, g}, fog*0.65) // fog desaturates toward grey
	bot = lerp3(bot, [3]float64{g + 0.05, g + 0.05, g + 0.05}, fog*0.65)
	ground := lerp3([3]float64{0.30, 0.34, 0.30}, [3]float64{0.50, 0.42, 0.32}, warm)

	objs := []brain.ObjSpec{{Kind: "plane", Color: round3v(clamp01v(ground))}}
	base := lerp3([3]float64{0.40, 0.60, 0.90}, [3]float64{0.92, 0.45, 0.25}, warm)
	n := 2 + int(math.Round(dens*8)) // 2..10 forms
	for i := 0; i < n; i++ {
		ang := float64(i) / float64(n) * 2 * math.Pi
		r := round3(0.4 + scale*0.8)
		o := brain.ObjSpec{
			Kind:  "sphere",
			X:     round3(math.Cos(ang) * (2.0 + 1.5*scale)), // a ring centred well ahead (z=6)
			Y:     r,
			Z:     round3(6 + math.Sin(ang)*(1.8+1.4*scale)),
			R:     r,
			Rough: round3(0.2 + 0.5*(1-glow)),
		}
		off := -0.1
		if i%2 != 0 {
			off = 0.1
		}
		for k := 0; k < 3; k++ {
			o.Color[k] = round3(clamp01(base[k] + off))
		}
		if glow > 0.5 && i%2 == 0 { // glow lights some forms
			k := (glow - 0.5) * 2 * 6
			for j := 0; j < 3; j++ {
				o.Emit[j] = round3(o.Color[j] * k)
			}
		}
		objs = append(objs, o)
	}
	sunY := round3(5 + sun*4)
	sunK := round3(6 + 9*day) // 6..15: present at night, bright by day (avoids exposure crush)
	objs = append(objs, brain.ObjSpec{X: 0, Y: sunY, Z: -1, R: 0.7, Emit: [3]float64{sunK, sunK, round3(sunK * 0.95)}})

	return brain.SceneSpec{
		Objects: objs,
		Light:   [3]float64{6, round3(4 + 8*sun), -4},
		SkyTop:  round3v(clamp01v(top)),
		SkyBot:  round3v(clamp01v(bot)),
		Name:    fmt.Sprintf("Q6 vector (hex %06b)", v.Hexagram().Number()),
	}
}

// ReadScene is the reader: the exact inverse of SceneVector.SceneSpec — it recovers
// the six Q6 coordinates from a scene the builder authored. Density survives only to
// the nearest form (1/8), the rest to rounding; the signs (the hexagram) are robust.
func ReadScene(spec brain.SceneSpec) SceneVector {
	var v SceneVector
	// warmth from the ground plane: ground.R = 0.30 + 0.20*warm
	for _, o := range spec.Objects {
		if o.Kind == "plane" {
			v[AxWarm] = clampf((o.Color[0]-0.30)/0.20, 0, 1)
			break
		}
	}
	// sun from the key light's height: light.Y = 4 + 8*sun
	v[AxSun] = clampf((spec.Light[1]-4)/8, 0, 1)
	// density, scale, glow from the ring forms (kind=="sphere"; the sun has no kind)
	nRing := 0
	var r, rough float64
	have := false
	for _, o := range spec.Objects {
		if o.Kind == "sphere" {
			nRing++
			if !have {
				r, rough, have = o.R, o.Rough, true
			}
		}
	}
	v[AxDensity] = clampf((float64(nRing)-2)/8, 0, 1)
	if have {
		v[AxScale] = clampf((r-0.4)/0.8, 0, 1)      // r = 0.4 + 0.8*scale
		v[AxGlow] = clampf(1-(rough-0.2)/0.5, 0, 1) // rough = 0.2 + 0.5*(1-glow)
	}
	// fog from how far the sky has desaturated toward grey, given recovered warm/sun
	day := v[AxSun]
	cool := [3]float64{0.28 + 0.16*day, 0.44 + 0.18*day, 0.68 + 0.28*day}
	warmc := [3]float64{0.12 + 0.7*day, 0.10 + 0.32*day, 0.07 + 0.16*day}
	baseTop := lerp3(cool, warmc, v[AxWarm])
	g := (baseTop[0] + baseTop[1] + baseTop[2]) / 3
	bi, bd := 0, 0.0
	for i := 0; i < 3; i++ { // solve on the channel furthest from grey (most signal)
		if d := math.Abs(baseTop[i] - g); d > bd {
			bi, bd = i, d
		}
	}
	if bd > 1e-6 {
		f := (baseTop[bi] - spec.SkyTop[bi]) / (baseTop[bi] - g) // f = fog*0.65
		v[AxFog] = clampf(f/0.65, 0, 1)
	}
	return v
}

// Mood names the feeling a scene's vector carries, from a valence/arousal reading of
// the axes: warm, sunny and glowing read positive; fog negative; many large glowing
// forms read as high energy.
func (v SceneVector) Mood() string {
	valence := 0.45*v[AxWarm] + 0.35*v[AxSun] + 0.25*v[AxGlow] - 0.35*v[AxFog]
	arousal := 0.45*v[AxDensity] + 0.30*v[AxGlow] + 0.25*v[AxScale]
	switch {
	case valence >= 0.55 && arousal >= 0.5:
		return "joyful"
	case valence >= 0.55:
		return "serene"
	case valence < 0.35 && arousal >= 0.5:
		return "ominous"
	case valence < 0.35:
		return "somber"
	default:
		return "neutral"
	}
}

// ViewerModel is a stand-in audience: it lingers (high engagement) on scenes near its
// latent taste and drifts off others. In a live call this scalar is the measured
// dwell time blended with the call's mood (raydir.Mood); here it is a smooth function
// of taste so the learning loop is reproducible and testable.
type ViewerModel struct {
	Pref  SceneVector
	Sigma float64 // taste tolerance (smaller = pickier); default 0.35
}

// Engagement is the dwell-like signal in (0,1], peaking at 1 when the scene matches
// the viewer's taste exactly.
func (vm ViewerModel) Engagement(v SceneVector) float64 {
	d2 := 0.0
	for i := range v {
		d := v[i] - vm.Pref[i]
		d2 += d * d
	}
	s := vm.Sigma
	if s <= 0 {
		s = 0.35
	}
	return math.Exp(-d2 / (2 * s * s))
}

// LearningDirector steers a world toward what holds an audience. It never sees the
// viewer's taste; it keeps a current best world, proposes a nearby variation each
// round, keeps it when engagement rises, and slowly narrows its exploration — an
// online (1+1) hill-climb over the six Q6 axes that converges on the audience.
type LearningDirector struct {
	v    SceneVector
	best float64
	step float64
	rng  *rand.Rand
}

// NewLearningDirector starts neutral (all axes at 0.5) with a wide exploration step.
func NewLearningDirector(seed int64) *LearningDirector {
	return &LearningDirector{
		v:    SceneVector{0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
		best: math.Inf(-1),
		step: 0.35,
		rng:  rand.New(rand.NewSource(seed)),
	}
}

// Round proposes a candidate near the current best, scores it with engage, keeps it
// if better, and anneals the exploration step. It returns the candidate and its
// engagement so a caller can trace the climb.
func (d *LearningDirector) Round(engage func(SceneVector) float64) (SceneVector, float64) {
	if math.IsInf(d.best, -1) {
		d.best = engage(d.v)
	}
	cand := d.v
	for i := range cand {
		cand[i] = clampf(cand[i]+d.rng.NormFloat64()*d.step, 0, 1)
	}
	score := engage(cand)
	if score > d.best {
		d.v, d.best = cand, score
	}
	d.step *= 0.97
	if d.step < 0.03 {
		d.step = 0.03
	}
	return cand, score
}

// Best is the director's current chosen world.
func (d *LearningDirector) Best() SceneVector { return d.v }

// Engagement is the engagement of the chosen world.
func (d *LearningDirector) Engagement() float64 { return d.best }

// --- small numeric helpers (kept local to the director/reader) ---

func round3(x float64) float64 { return math.Round(x*1000) / 1000 }

func lerp3(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{a[0]*(1-t) + b[0]*t, a[1]*(1-t) + b[1]*t, a[2]*(1-t) + b[2]*t}
}

func clamp01v(a [3]float64) [3]float64 {
	return [3]float64{clamp01(a[0]), clamp01(a[1]), clamp01(a[2])}
}

func round3v(a [3]float64) [3]float64 {
	return [3]float64{round3(a[0]), round3(a[1]), round3(a[2])}
}
