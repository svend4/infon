package aisource

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/pseudo"
)

// End-to-end: PseudoBackend -> HTTP -> reference brain -> pseudo-image spec ->
// rendered raster, for every format. Proves a modelless backend paints pixels.
func TestPseudoBackendAllFormats(t *testing.T) {
	srv := httptest.NewServer(brain.Handler())
	defer srv.Close()

	formats := []pseudo.Format{
		"", pseudo.FormatGrid, pseudo.FormatPixels, pseudo.FormatGlyphs,
		pseudo.FormatSigils, pseudo.FormatVector, pseudo.FormatSketch,
	}
	for _, f := range formats {
		b := NewPseudoBackend(srv.URL, f)
		img, err := b.Generate(context.Background(), "a calm harbor at dawn", 320, 144)
		if err != nil {
			t.Fatalf("format %q: %v", f, err)
		}
		if img == nil {
			t.Fatalf("format %q: nil image", f)
		}
		if bnds := img.Bounds(); bnds.Dx() != 320 || bnds.Dy() != 144 {
			t.Fatalf("format %q: bounds = %v, want 320x144", f, bnds)
		}
	}
}
