package device

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"
)

func TestGenerativeSourceImplementsCamera(t *testing.T) {
	// Also enforced at compile time in generative.go; this documents intent.
	var _ Camera = NewGenerativeSource(64, 48, 15, PlasmaGenerator{})
}

func TestGenerativeSourceLifecycle(t *testing.T) {
	src := NewGenerativeSource(64, 48, 15, PlasmaGenerator{})

	if src.IsOpen() {
		t.Fatal("source should start closed")
	}

	// Reading before Open must fail.
	if _, err := src.Read(); err != ErrCameraNotOpen {
		t.Fatalf("Read before Open = %v, want ErrCameraNotOpen", err)
	}

	if err := src.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !src.IsOpen() {
		t.Fatal("source should be open after Open")
	}

	// Double Open must report busy.
	if err := src.Open(); err != ErrCameraBusy {
		t.Fatalf("double Open = %v, want ErrCameraBusy", err)
	}

	img, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if img == nil {
		t.Fatal("Read returned nil image")
	}
	if got := img.Bounds(); got.Dx() != 64 || got.Dy() != 48 {
		t.Fatalf("frame bounds = %dx%d, want 64x48", got.Dx(), got.Dy())
	}

	if err := src.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := src.Read(); err != ErrCameraNotOpen {
		t.Fatalf("Read after Close = %v, want ErrCameraNotOpen", err)
	}
}

func TestGenerativeSourceRequiresGenerator(t *testing.T) {
	src := NewGenerativeSource(32, 32, 15, nil)
	if err := src.Open(); err == nil {
		t.Fatal("Open with nil generator should fail")
	}
}

func TestProceduralGenerators(t *testing.T) {
	for _, name := range []string{"plasma", "ripple"} {
		gen, err := NewProceduralGenerator(name)
		if err != nil {
			t.Fatalf("NewProceduralGenerator(%q): %v", name, err)
		}
		if gen.Name() != name {
			t.Errorf("Name() = %q, want %q", gen.Name(), name)
		}
		img, err := gen.Generate(80, 60, 0, 250*time.Millisecond)
		if err != nil {
			t.Fatalf("%s.Generate: %v", name, err)
		}
		if b := img.Bounds(); b.Dx() != 80 || b.Dy() != 60 {
			t.Fatalf("%s bounds = %dx%d, want 80x60", name, b.Dx(), b.Dy())
		}
	}
}

func TestNewProceduralGeneratorUnknown(t *testing.T) {
	if _, err := NewProceduralGenerator("does-not-exist"); err == nil {
		t.Fatal("expected error for unknown generator")
	}
}

func TestNeuralGeneratorPlaceholder(t *testing.T) {
	gen := NewNeuralGenerator(nil, "anything")
	if gen.HasBackend() {
		t.Fatal("generator should report no backend")
	}
	// With no backend it must still return a usable frame, not an error.
	img, err := gen.Generate(48, 36, 0, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 48 || b.Dy() != 36 {
		t.Fatalf("placeholder bounds = %dx%d, want 48x36", b.Dx(), b.Dy())
	}
}

// fakeBackend is a deterministic stand-in for a real model: it fills the frame
// with a fixed color so tests can detect when async generation has landed.
type fakeBackend struct{ c color.RGBA }

func (fakeBackend) Name() string { return "fake" }

func (f fakeBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, f.c)
		}
	}
	return img, nil
}

func TestNeuralGeneratorAsyncBackend(t *testing.T) {
	want := color.RGBA{R: 200, G: 10, B: 10, A: 255}
	gen := NewNeuralGenerator(fakeBackend{c: want}, "a red square")
	gen.Interval = 0 // regenerate eagerly for the test

	// The first Generate kicks an async generation and returns a placeholder.
	// Poll until the backend's frame has landed.
	deadline := time.Now().Add(2 * time.Second)
	for {
		img, err := gen.Generate(32, 32, 0, 0)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		r, g, b, _ := img.At(16, 16).RGBA()
		if uint8(r>>8) == want.R && uint8(g>>8) == want.G && uint8(b>>8) == want.B {
			break // backend frame is now being served
		}
		if time.Now().After(deadline) {
			t.Fatal("backend frame never became visible")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := gen.LastError(); err != nil {
		t.Fatalf("unexpected backend error: %v", err)
	}
}
