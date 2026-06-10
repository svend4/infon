package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// Lingering (slow, no turning) reads as calm.
func TestMoodCalm(t *testing.T) {
	m := NewMood()
	p := Pose{}
	for i := 0; i < 80; i++ {
		p.Pos.Z += 0.02 // a slow shuffle forward
		m.Observe(p, 0.1)
	}
	if got := m.Name(); got != "calm" {
		t.Errorf("a dawdler should read calm, got %q (speed %.2f turn %.2f)", got, m.speedEMA, m.turnEMA)
	}
}

// Pressing forward fast and straight reads as restless.
func TestMoodRestless(t *testing.T) {
	m := NewMood()
	p := Pose{}
	for i := 0; i < 80; i++ {
		p.Pos.Z += 0.3 // 3 units/second
		m.Observe(p, 0.1)
	}
	if got := m.Name(); got != "restless" {
		t.Errorf("a sprinter should read restless, got %q (speed %.2f turn %.2f)", got, m.speedEMA, m.turnEMA)
	}
}

// Turning a lot (looking around) reads as curious, even if not moving fast.
func TestMoodCurious(t *testing.T) {
	m := NewMood()
	p := Pose{}
	for i := 0; i < 80; i++ {
		p.Yaw += 0.1 // ~1 rad/second of turning
		p.Pos.Z += 0.02
		m.Observe(p, 0.1)
	}
	if got := m.Name(); got != "curious" {
		t.Errorf("a wanderer should read curious, got %q (speed %.2f turn %.2f)", got, m.speedEMA, m.turnEMA)
	}
}

// The three moods bias the director toward three distinct tones.
func TestMoodPromptsDiffer(t *testing.T) {
	got := map[string]bool{}
	for _, turnSpeed := range [][2]float64{{0, 0.02}, {0, 0.3}, {0.1, 0.02}} {
		m := NewMood()
		p := Pose{}
		for i := 0; i < 80; i++ {
			p.Yaw += turnSpeed[0]
			p.Pos.Z += turnSpeed[1]
			m.Observe(p, 0.1)
		}
		got[m.Prompt()] = true
	}
	if len(got) != 3 {
		t.Errorf("expected three distinct mood prompts, got %d: %v", len(got), got)
	}
}

// angDelta measures the smallest angle and handles wraparound.
func TestAngDelta(t *testing.T) {
	if d := angDelta(0.1, -0.1); math.Abs(d-0.2) > 1e-9 {
		t.Errorf("angDelta(0.1,-0.1) = %.4f, want 0.2", d)
	}
	if d := angDelta(3.0, -3.0); math.Abs(d-(2*math.Pi-6)) > 1e-9 { // wraps the short way
		t.Errorf("angDelta across the wrap = %.4f, want %.4f", d, 2*math.Pi-6)
	}
}

// The world biases its grow prompts by mood when sensing is on, and leaves them
// alone when it's off.
func TestWorldBiasPrompt(t *testing.T) {
	w := NewWorld()
	if w.BiasPrompt("a quiet field") != "a quiet field" {
		t.Error("with mood sensing off the prompt should be unchanged")
	}
	w.SetMoodSensing(true)
	if got := w.BiasPrompt("a field"); got == "a field" || len(got) <= len("a field") {
		t.Errorf("with mood sensing on the prompt should gain a tone, got %q", got)
	}
	if w.MoodName() == "" {
		t.Error("mood sensing on should report a mood name")
	}
}

// The reference director actually builds differently for different moods: the
// three biased prompts yield three different scenes.
func TestMoodShiftsAuthoredScene(t *testing.T) {
	b := brain.Local{}
	count := func(tone string) int {
		_, spec, _ := AuthorScene(b, "a field, "+tone)
		return len(spec.Objects)
	}
	calm := count("quiet, still and intimate")
	restless := count("vast, grand and open")
	curious := count("strange, varied and surprising")
	plain := func() int { _, s, _ := AuthorScene(b, "a field"); return len(s.Objects) }()
	if calm <= plain || restless <= plain || curious <= plain {
		t.Errorf("each mood should add to the scene: plain %d, calm %d, restless %d, curious %d", plain, calm, restless, curious)
	}
	if calm == restless && restless == curious {
		t.Error("the three moods should not all build the same number of objects")
	}
	// the grand/open mood should render (a sanity build through the scene graph):
	// look toward the distant pyramid landmark and expect a hit.
	_, spec, _ := AuthorScene(b, "a field, vast, grand and open")
	if _, ok := nearestHit(buildWorld(spec), raytrace.Ray{Origin: raytrace.Vec3{Y: 4, Z: -30}, Dir: raytrace.Vec3{Z: 1}}); !ok {
		t.Error("a restless scene should contain its grand landmark")
	}
}

// buildWorld renders a spec into a scene for hit-testing.
func buildWorld(spec brain.SceneSpec) *raytrace.Scene {
	s := BuildScene(spec)
	s.BuildBVH()
	return s
}
