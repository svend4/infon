package raysource

import (
	"testing"

	"github.com/svend4/infon/internal/device"
)

func TestRayGeneratorProducesFrame(t *testing.T) {
	g := New(nil, 1)
	if g.Name() != "raytrace" {
		t.Errorf("name = %q", g.Name())
	}
	img, err := g.Generate(40, 30, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 40 || img.Bounds().Dy() != 30 {
		t.Fatalf("frame = %v, want 40x30", img.Bounds())
	}
	// Same elapsed time => same camera => identical pixels (deterministic).
	img2, _ := g.Generate(40, 30, 7, 0)
	r1, _, _, _ := img.At(20, 15).RGBA()
	r2, _, _, _ := img2.At(20, 15).RGBA()
	if r1 != r2 {
		t.Error("frames at the same elapsed time should be identical")
	}
}

func TestRayGeneratorIsACamera(t *testing.T) {
	src := device.NewGenerativeSource(48, 32, 15, New(nil, 1))
	if err := src.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = src.Close() }()
	img, err := src.Read()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 48 || img.Bounds().Dy() != 32 {
		t.Fatalf("camera frame = %v, want 48x32", img.Bounds())
	}
}
