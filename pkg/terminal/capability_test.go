package terminal

import "testing"

// setEnv sets the detection-relevant env for one test and clears the rest, so
// each case is isolated.
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for _, k := range []string{"TERM", "COLORTERM", "TERM_PROGRAM", "LANG", "LC_ALL", "LC_CTYPE"} {
		t.Setenv(k, "")
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestDetectTrueColorFromColorterm(t *testing.T) {
	setEnv(t, map[string]string{"COLORTERM": "truecolor"})
	if !DetectCapability().TrueColor {
		t.Error("COLORTERM=truecolor should yield TrueColor")
	}
}

func TestDetectPlainXterm(t *testing.T) {
	setEnv(t, map[string]string{"TERM": "xterm"})
	c := DetectCapability()
	if c.TrueColor || c.Unicode13 || c.SupportsPixelProtocol() {
		t.Errorf("plain xterm should advertise nothing fancy, got %+v", c)
	}
	if c.BestBlitMode() != "quadrant" {
		t.Errorf("plain xterm best mode = %q, want quadrant", c.BestBlitMode())
	}
}

func TestDetectKittyIsRich(t *testing.T) {
	setEnv(t, map[string]string{"TERM": "xterm-kitty", "COLORTERM": "truecolor", "LANG": "en_US.UTF-8"})
	c := DetectCapability()
	if !c.TrueColor {
		t.Error("kitty should be truecolor")
	}
	if !c.Unicode13 {
		t.Error("kitty + UTF-8 should imply sextant-capable")
	}
	if !c.Kitty || !c.SupportsPixelProtocol() {
		t.Error("kitty should support a pixel protocol")
	}
	if c.BestBlitMode() != "sextant" {
		t.Errorf("kitty best mode = %q, want sextant", c.BestBlitMode())
	}
}

func TestBestBlitModeTiers(t *testing.T) {
	// TrueColor but no Unicode13 -> halfblock.
	setEnv(t, map[string]string{"COLORTERM": "truecolor", "TERM": "xterm-256color"})
	if m := DetectCapability().BestBlitMode(); m != "halfblock" {
		t.Errorf("truecolor-only best mode = %q, want halfblock", m)
	}
}

func TestUnicode13RequiresUTF8(t *testing.T) {
	// wezterm without a UTF-8 locale should NOT claim sextant support.
	setEnv(t, map[string]string{"TERM_PROGRAM": "wezterm", "LANG": "C"})
	if DetectCapability().Unicode13 {
		t.Error("non-UTF-8 locale must not advertise Unicode13")
	}
}

func TestSixelDetection(t *testing.T) {
	setEnv(t, map[string]string{"TERM": "foot"})
	if !DetectCapability().Sixel {
		t.Error("foot should be detected as Sixel-capable")
	}
}
