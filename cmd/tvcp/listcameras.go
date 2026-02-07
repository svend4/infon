package main

import (
	"fmt"
	"os"

	"github.com/svend4/infon/internal/device"
)

func runListCameras() {
	fmt.Println("🎥 Available Cameras")
	fmt.Println()

	cameras, err := device.ListCameras()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing cameras: %v\n", err)
		if err == device.ErrNoCamera {
			fmt.Println("No cameras found. Make sure:")
			fmt.Println("  1. A camera is connected")
			fmt.Println("  2. You have permission to access /dev/video* devices")
			fmt.Println("  3. Camera drivers are loaded (v4l2)")
			fmt.Println()
			fmt.Println("You can still use test cameras:")
			fmt.Println("  ./bin/tvcp preview bounce")
			fmt.Println("  ./bin/tvcp call <host:port> gradient")
		}
		os.Exit(1)
	}

	if len(cameras) == 0 {
		fmt.Println("No cameras found.")
		fmt.Println()
		fmt.Println("You can use test cameras instead:")
		fmt.Println("  Patterns: bounce, gradient, noise, colorbar")
		return
	}

	fmt.Printf("Found %d camera(s):\n\n", len(cameras))
	for _, cam := range cameras {
		fmt.Printf("  [%d] %s\n", cam.ID, cam.Name)
		fmt.Printf("      Path: %s\n", cam.Path)
		if cam.Available {
			fmt.Printf("      Status: Available ✓\n")
		} else {
			fmt.Printf("      Status: Busy ✗\n")
		}
		if len(cam.Resolutions) > 0 {
			fmt.Printf("      Common resolutions:\n")
			for _, res := range cam.Resolutions {
				fmt.Printf("        - %dx%d\n", res.Width, res.Height)
			}
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  To use camera 0:")
	fmt.Println("  ./bin/tvcp preview --camera 0")
	fmt.Println()
	fmt.Println("  To use test pattern:")
	fmt.Println("  ./bin/tvcp preview bounce")
}
