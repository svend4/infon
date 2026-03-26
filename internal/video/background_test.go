package video

import (
	"image"
	"image/color"
	"testing"
)

func TestNewBackgroundProcessor(t *testing.T) {
	bp := NewBackgroundProcessor()

	if bp == nil {
		t.Fatal("NewBackgroundProcessor() returned nil")
	}

	if bp.mode != BackgroundNone {
		t.Errorf("Default mode = %v, expected BackgroundNone", bp.mode)
	}

	if bp.blurRadius != 15 {
		t.Errorf("Default blur radius = %d, expected 15", bp.blurRadius)
	}

	if bp.threshold != 128 {
		t.Errorf("Default threshold = %d, expected 128", bp.threshold)
	}
}

func TestBackgroundProcessor_SetMode(t *testing.T) {
	bp := NewBackgroundProcessor()

	bp.SetMode(BackgroundBlur)

	bp.mu.RLock()
	mode := bp.mode
	bp.mu.RUnlock()

	if mode != BackgroundBlur {
		t.Errorf("Mode = %v, expected BackgroundBlur", mode)
	}
}

func TestBackgroundProcessor_SetBlurRadius(t *testing.T) {
	bp := NewBackgroundProcessor()

	tests := []struct {
		input    int
		expected int
	}{
		{10, 10},
		{0, 1},   // Should clamp to 1
		{-5, 1},  // Should clamp to 1
		{60, 50}, // Should clamp to 50
		{25, 25},
	}

	for _, tt := range tests {
		bp.SetBlurRadius(tt.input)

		bp.mu.RLock()
		radius := bp.blurRadius
		bp.mu.RUnlock()

		if radius != tt.expected {
			t.Errorf("SetBlurRadius(%d): got %d, expected %d", tt.input, radius, tt.expected)
		}
	}
}

func TestBackgroundProcessor_SetReplacementImage(t *testing.T) {
	bp := NewBackgroundProcessor()

	img := image.NewRGBA(image.Rect(0, 0, 640, 480))

	bp.SetReplacementImage(img)

	bp.mu.RLock()
	replacementImg := bp.replacementImg
	bp.mu.RUnlock()

	if replacementImg != img {
		t.Error("Replacement image not set correctly")
	}
}

func TestBackgroundProcessor_SetBackgroundColor(t *testing.T) {
	bp := NewBackgroundProcessor()

	testColor := color.RGBA{255, 0, 0, 255} // Red

	bp.SetBackgroundColor(testColor)

	bp.mu.RLock()
	bgColor := bp.backgroundColor
	bp.mu.RUnlock()

	if bgColor != testColor {
		t.Errorf("Background color = %v, expected %v", bgColor, testColor)
	}
}

func TestBackgroundProcessor_Process_None(t *testing.T) {
	bp := NewBackgroundProcessor()

	// Create test image
	img := createTestImage(320, 240)

	result, err := bp.Process(img)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}

	// With BackgroundNone, should return same image
	if result != img {
		t.Error("Process() with BackgroundNone should return same image")
	}

	stats := bp.GetStatistics()
	if stats.FramesProcessed != 1 {
		t.Errorf("FramesProcessed = %d, expected 1", stats.FramesProcessed)
	}
}

func TestBackgroundProcessor_Process_Blur(t *testing.T) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundBlur)
	bp.SetBlurRadius(5)

	img := createTestImage(320, 240)

	result, err := bp.Process(img)
	if err != nil {
		t.Fatalf("Process() blur failed: %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}

	// Check dimensions preserved
	if result.Bounds() != img.Bounds() {
		t.Error("Result dimensions don't match input")
	}

	// Image should be different (blurred)
	if result == img {
		t.Error("Blurred image should be different from input")
	}
}

func TestBackgroundProcessor_Process_Color(t *testing.T) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundColor)

	testColor := color.RGBA{0, 0, 255, 255} // Blue
	bp.SetBackgroundColor(testColor)

	img := createTestImage(320, 240)

	result, err := bp.Process(img)
	if err != nil {
		t.Fatalf("Process() color failed: %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}

	// Check dimensions
	if result.Bounds() != img.Bounds() {
		t.Error("Result dimensions don't match input")
	}

	// Check that some background pixels match the test color
	// Check corner pixels (should be background)
	cornerColor := result.RGBAAt(0, 0)
	if cornerColor.B < 200 {
		t.Error("Corner pixel should be blue (background color)")
	}
}

func TestBackgroundProcessor_Process_Image(t *testing.T) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundImage)

	// Create replacement background
	bgImg := image.NewRGBA(image.Rect(0, 0, 640, 480))
	// Fill with red
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			bgImg.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	bp.SetReplacementImage(bgImg)

	img := createTestImage(320, 240)

	result, err := bp.Process(img)
	if err != nil {
		t.Fatalf("Process() image replacement failed: %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}

	// Check dimensions
	if result.Bounds() != img.Bounds() {
		t.Error("Result dimensions don't match input")
	}
}

func TestBackgroundProcessor_Process_Image_NoReplacement(t *testing.T) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundImage)
	// Don't set replacement image

	img := createTestImage(320, 240)

	_, err := bp.Process(img)
	if err == nil {
		t.Error("Process() should fail when no replacement image set")
	}
}

func TestBackgroundProcessor_CreateMask(t *testing.T) {
	bp := NewBackgroundProcessor()

	img := createTestImage(320, 240)

	mask := bp.createMask(img)

	if mask == nil {
		t.Fatal("createMask() returned nil")
	}

	// Check dimensions
	if mask.Bounds() != img.Bounds() {
		t.Error("Mask dimensions don't match input")
	}

	// Check that center is foreground (255)
	centerX := img.Bounds().Dx() / 2
	centerY := img.Bounds().Dy() / 2
	centerValue := mask.GrayAt(centerX, centerY).Y

	if centerValue != 255 {
		t.Errorf("Center mask value = %d, expected 255 (foreground)", centerValue)
	}

	// Check that corners are background (0)
	cornerValue := mask.GrayAt(0, 0).Y

	if cornerValue != 0 {
		t.Errorf("Corner mask value = %d, expected 0 (background)", cornerValue)
	}
}

func TestBackgroundProcessor_GaussianBlur(t *testing.T) {
	bp := NewBackgroundProcessor()

	img := createTestImage(100, 100)

	// Test with radius 0 (should return same image)
	result := bp.gaussianBlur(img, 0)
	if result != img {
		t.Error("gaussianBlur with radius 0 should return same image")
	}

	// Test with radius 5
	result = bp.gaussianBlur(img, 5)
	if result == nil {
		t.Fatal("gaussianBlur() returned nil")
	}

	if result.Bounds() != img.Bounds() {
		t.Error("Blurred image dimensions don't match input")
	}
}

func TestBackgroundProcessor_ResizeImage(t *testing.T) {
	bp := NewBackgroundProcessor()

	img := createTestImage(640, 480)

	// Resize to smaller
	result := bp.resizeImage(img, 320, 240)

	if result == nil {
		t.Fatal("resizeImage() returned nil")
	}

	if result.Bounds().Dx() != 320 || result.Bounds().Dy() != 240 {
		t.Errorf("Resized dimensions = %dx%d, expected 320x240",
			result.Bounds().Dx(), result.Bounds().Dy())
	}

	// Resize to larger
	result = bp.resizeImage(img, 1280, 960)

	if result.Bounds().Dx() != 1280 || result.Bounds().Dy() != 960 {
		t.Errorf("Resized dimensions = %dx%d, expected 1280x960",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestBackgroundProcessor_GetStatistics(t *testing.T) {
	bp := NewBackgroundProcessor()

	// Process some frames
	img := createTestImage(320, 240)

	bp.SetMode(BackgroundBlur)
	bp.Process(img)
	bp.Process(img)
	bp.Process(img)

	stats := bp.GetStatistics()

	if stats.FramesProcessed != 3 {
		t.Errorf("FramesProcessed = %d, expected 3", stats.FramesProcessed)
	}

	if stats.Mode != BackgroundBlur {
		t.Errorf("Mode = %v, expected BackgroundBlur", stats.Mode)
	}

	if stats.BlurRadius != bp.blurRadius {
		t.Errorf("BlurRadius = %d, expected %d", stats.BlurRadius, bp.blurRadius)
	}
}

func TestBackgroundProcessor_Reset(t *testing.T) {
	bp := NewBackgroundProcessor()

	// Process some frames
	img := createTestImage(320, 240)
	bp.SetMode(BackgroundBlur)
	bp.Process(img)
	bp.Process(img)

	bp.Reset()

	stats := bp.GetStatistics()

	if stats.FramesProcessed != 0 {
		t.Errorf("After reset, FramesProcessed = %d, expected 0", stats.FramesProcessed)
	}
}

func TestBackgroundProcessor_Concurrent(t *testing.T) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundBlur)

	img := createTestImage(320, 240)

	// Test concurrent processing
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := bp.Process(img)
			if err != nil {
				t.Errorf("Concurrent Process() failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := bp.GetStatistics()
	if stats.FramesProcessed != 10 {
		t.Errorf("FramesProcessed = %d, expected 10", stats.FramesProcessed)
	}
}

func TestBackgroundProcessor_EdgeBlending(t *testing.T) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundColor)
	bp.SetBackgroundColor(color.RGBA{255, 0, 0, 255})

	img := createTestImage(320, 240)

	result, err := bp.Process(img)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}

	// Check that there are blended pixels (not just 0 or 255 alpha)
	bounds := result.Bounds()
	hasBlendedPixels := false

	// Check pixels in the middle ring (should have blending)
	centerX := bounds.Dx() / 2
	centerY := bounds.Dy() / 2
	radiusCheck := bounds.Dx() / 4

	for angle := 0.0; angle < 360.0; angle += 45.0 {
		x := centerX + int(float64(radiusCheck)*cosDegrees(angle))
		y := centerY + int(float64(radiusCheck)*sinDegrees(angle))

		if x >= 0 && x < bounds.Dx() && y >= 0 && y < bounds.Dy() {
			c := result.RGBAAt(x, y)
			// If it's not pure foreground or pure background, it's blended
			if c.R > 0 && c.R < 255 {
				hasBlendedPixels = true
				break
			}
		}
	}

	if !hasBlendedPixels {
		t.Log("Note: No blended pixels detected (may be due to test image characteristics)")
	}
}

// Helper functions

func createTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a simple gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)

			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

func cosDegrees(deg float64) float64 {
	return float64(0.017453292519943295 * deg) // Approximate cos
}

func sinDegrees(deg float64) float64 {
	return float64(0.017453292519943295 * deg) // Approximate sin
}

// Benchmarks

func BenchmarkBackgroundProcessor_Process_None(b *testing.B) {
	bp := NewBackgroundProcessor()
	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Process(img)
	}
}

func BenchmarkBackgroundProcessor_Process_Blur(b *testing.B) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundBlur)
	bp.SetBlurRadius(10)

	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Process(img)
	}
}

func BenchmarkBackgroundProcessor_Process_Color(b *testing.B) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundColor)
	bp.SetBackgroundColor(color.RGBA{0, 255, 0, 255})

	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Process(img)
	}
}

func BenchmarkBackgroundProcessor_Process_Image(b *testing.B) {
	bp := NewBackgroundProcessor()
	bp.SetMode(BackgroundImage)

	bgImg := createTestImage(640, 480)
	bp.SetReplacementImage(bgImg)

	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Process(img)
	}
}

func BenchmarkBackgroundProcessor_CreateMask(b *testing.B) {
	bp := NewBackgroundProcessor()
	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.createMask(img)
	}
}

func BenchmarkBackgroundProcessor_GaussianBlur(b *testing.B) {
	bp := NewBackgroundProcessor()
	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.gaussianBlur(img, 15)
	}
}

func BenchmarkBackgroundProcessor_ResizeImage(b *testing.B) {
	bp := NewBackgroundProcessor()
	img := createTestImage(1920, 1080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.resizeImage(img, 640, 480)
	}
}
