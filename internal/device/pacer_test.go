package device

import (
	"testing"
	"time"
)

func TestPacerFirstFrameAlwaysRenders(t *testing.T) {
	p := NewPacer(15, 0)
	if !p.ShouldRender(time.Now()) {
		t.Error("first call should always render")
	}
	if p.Rendered != 1 {
		t.Errorf("rendered = %d, want 1", p.Rendered)
	}
}

func TestPacerHoldsUntilDue(t *testing.T) {
	start := time.Unix(0, 0)
	p := NewPacer(10, 0)  // 100ms interval
	p.ShouldRender(start) // arm at t=0, next due t=100ms

	if p.ShouldRender(start.Add(50 * time.Millisecond)) {
		t.Error("should not render before the interval elapses")
	}
	if !p.ShouldRender(start.Add(100 * time.Millisecond)) {
		t.Error("should render exactly at the due time")
	}
}

func TestPacerSkipsMissedSlots(t *testing.T) {
	start := time.Unix(0, 0)
	p := NewPacer(10, 0) // 100ms interval
	p.ShouldRender(start)

	// Jump 350ms ahead: slots at 100/200/300 were due; we render once and skip
	// the fully-missed ones rather than firing a catch-up burst.
	if !p.ShouldRender(start.Add(350 * time.Millisecond)) {
		t.Fatal("should render when well past due")
	}
	if p.Skipped == 0 {
		t.Errorf("expected skipped slots, got %d", p.Skipped)
	}
	if p.Rendered != 2 {
		t.Errorf("rendered = %d, want 2 (initial + this one)", p.Rendered)
	}
}

func TestPacerWidensUnderLoad(t *testing.T) {
	p := NewPacer(30, 0) // 33.3ms target
	base := p.EffectiveFPS()
	if base < 29 || base > 31 {
		t.Errorf("base FPS = %.1f, want ~30", base)
	}
	// Report sustained work of 100ms (3x the target period).
	for i := 0; i < 20; i++ {
		p.Observe(100 * time.Millisecond)
	}
	loaded := p.EffectiveFPS()
	if loaded >= base {
		t.Errorf("effective FPS should drop under load: base=%.1f loaded=%.1f", base, loaded)
	}
	if loaded > 11 {
		t.Errorf("under 100ms work, effective FPS should approach ~10, got %.1f", loaded)
	}
}

func TestPacerRespectsMaxFPS(t *testing.T) {
	// target 1000 fps but capped at 60.
	p := NewPacer(1000, 60)
	if fps := p.EffectiveFPS(); fps > 61 {
		t.Errorf("effective FPS = %.1f, should be capped at ~60", fps)
	}
}

func TestPacerDefaults(t *testing.T) {
	p := NewPacer(0, 0) // invalid -> default 15
	if fps := p.EffectiveFPS(); fps < 14 || fps > 16 {
		t.Errorf("default FPS = %.1f, want ~15", fps)
	}
}

func TestPacerObserveSmoothing(t *testing.T) {
	p := NewPacer(15, 0)
	p.Observe(10 * time.Millisecond)
	first := p.emaWork
	p.Observe(200 * time.Millisecond)
	// EMA should move toward the new value but not jump fully to it.
	if p.emaWork <= first || p.emaWork >= 200*time.Millisecond {
		t.Errorf("EMA not smoothing: first=%v after=%v", first, p.emaWork)
	}
}
