package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func runDemo() {
	// Check if image path was provided
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing image path")
		fmt.Fprintln(os.Stderr, "Usage: tvcp demo <image_path>")
		fmt.Fprintln(os.Stderr, "\nThis demo displays an image in the terminal using Unicode block characters.")
		fmt.Fprintln(os.Stderr, "Supported formats: PNG, JPEG, GIF")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp demo photo.jpg")
		fmt.Fprintln(os.Stderr, "  tvcp demo test.png")
		os.Exit(1)
	}

	imagePath := os.Args[2]

	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening image: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		os.Exit(1)
	}

	bounds := img.Bounds()
	fmt.Printf("Loaded %s image: %dx%d pixels\n", format, bounds.Dx(), bounds.Dy())

	// Calculate terminal size (default to 80x24)
	termWidth := 80
	termHeight := 24

	// Try to get actual terminal size
	// In a real implementation, we'd use syscall to get actual terminal size
	// For now, use default values

	fmt.Printf("Rendering to terminal: %dx%d characters\n", termWidth, termHeight)
	fmt.Println("Rendering...")

	// Convert image to terminal frame
	frame := babe.ImageToFrame(img, termWidth, termHeight)

	// Render to terminal
	frame.RenderToTerminal()

	fmt.Println(color.Reset)
	fmt.Println("\n✓ Render complete!")
	fmt.Printf("Image resolution: %dx%d pixels\n", bounds.Dx(), bounds.Dy())
	fmt.Printf("Terminal resolution: %dx%d chars = %dx%d effective pixels\n",
		termWidth, termHeight, termWidth*2, termHeight*2)
}

// runDemoPattern demonstrates rendering patterns
func runDemoPattern() {
	frame := terminal.NewFrame(40, 20)

	// Draw a gradient
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			intensity := uint8(float64(x) / 40.0 * 255)
			c := color.RGB{R: intensity, G: intensity, B: intensity}
			frame.SetBlock(x, y, '█', c, color.Black)
		}
	}

	// Draw a box
	frame.DrawBox(5, 5, 30, 10, color.White, color.Black)

	// Draw text
	frame.DrawText(10, 8, "TVCP Demo", color.Cyan, color.Black)

	// Render
	frame.RenderToTerminal()
	fmt.Println(color.Reset)
}
