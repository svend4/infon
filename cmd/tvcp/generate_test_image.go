package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// generateTestImage creates a test pattern image
func generateTestImage(filename string, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a colorful test pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a gradient pattern
			r := uint8(float64(x) / float64(width) * 255)
			g := uint8(float64(y) / float64(height) * 255)
			b := uint8(math.Sin(float64(x+y)/20.0)*127 + 128)

			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Add some geometric shapes
	// Circle in the center
	cx, cy := width/2, height/2
	radius := width / 4
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy < radius*radius {
				img.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	// Save to file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// runGenerate generates a test image
func runGenerate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp generate <output.png>")
		os.Exit(1)
	}

	filename := os.Args[2]
	width := 640
	height := 480

	fmt.Printf("Generating test image: %dx%d\n", width, height)
	if err := generateTestImage(filename, width, height); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Test image saved to: %s\n", filename)
	fmt.Println("\nNow try:")
	fmt.Printf("  tvcp demo %s\n", filename)
}
