package raydir

import "testing"

func feedAudio(loud func(i int) float64, frames int) *Mood {
	m := NewMood()
	for i := 0; i < frames; i++ {
		m.ObserveAudio(loud(i), 0.05)
	}
	return m
}

func TestVoiceMoodCalmRestlessCurious(t *testing.T) {
	calm := feedAudio(func(int) float64 { return 0.2 }, 120) // quiet, steady
	if calm.Name() != "calm" {
		t.Errorf("a quiet voice should read calm, got %s", calm.Name())
	}
	restless := feedAudio(func(int) float64 { return 0.9 }, 120) // loud, steady
	if restless.Name() != "restless" {
		t.Errorf("a loud steady voice should read restless, got %s", restless.Name())
	}
	curious := feedAudio(func(i int) float64 { // loud and swinging (expressive prosody)
		if i%2 == 0 {
			return 0.3
		}
		return 0.8
	}, 120)
	if curious.Name() != "curious" {
		t.Errorf("an expressive, swinging voice should read curious, got %s", curious.Name())
	}
}

func TestVoiceLoudnessRange(t *testing.T) {
	if voiceLoudness(nil) != 0 {
		t.Error("silence should be 0 loudness")
	}
	quiet := make([]int16, 256)
	for i := range quiet {
		quiet[i] = 200
	}
	loud := make([]int16, 256)
	for i := range loud {
		loud[i] = 12000
	}
	if voiceLoudness(loud) <= voiceLoudness(quiet) {
		t.Error("a louder frame should read louder")
	}
	if v := voiceLoudness(loud); v < 0 || v > 1 {
		t.Errorf("loudness must be 0..1, got %.3f", v)
	}
}

func TestWorldObserveVoiceNeedsSensing(t *testing.T) {
	w := NewWorld()
	w.ObserveVoice(make([]int16, 64), 0.05) // no panic, no-op without sensing
	w.SetMoodSensing(true)
	loud := make([]int16, 256)
	for i := range loud {
		loud[i] = 14000
	}
	for i := 0; i < 120; i++ {
		w.ObserveVoice(loud, 0.05)
	}
	if w.MoodName() != "restless" {
		t.Errorf("a loud voice should make the world restless, got %s", w.MoodName())
	}
}
