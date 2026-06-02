// Command semdistdemo prints the rate-distortion-of-MEANING benchmark
// (pkg/semdist): how many wire bytes buy how much surviving meaning (MarksIoU),
// as the keyframe interval is varied under packet loss and as Reed-Solomon parity
// is varied under byte corruption. It also renders the GOP curve to a PNG.
//
//	go run ./cmd/semdistdemo
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/semdist"
)

func main() {
	out := flag.String("out", "_bench", "output directory")
	flag.Parse()
	os.MkdirAll(*out, 0o755)

	s := semdist.Scene(24, 12, 8)
	gop := semdist.GopCurve(s, []int{1, 2, 4, 8, 16}, 6, 0.25, 7)
	ecc := semdist.EccCurve(s, []int{2, 4, 8, 16}, 3, 7)

	fmt.Println("rate-distortion of MEANING (24-frame scene)")
	fmt.Println("\nA) keyframe interval under 25% packet loss:")
	fmt.Printf("   %4s %8s %7s %8s\n", "gop", "bytes", "IoU", "exact")
	for _, p := range gop {
		fmt.Printf("   %4d %8d %7.3f %5d/%d\n", p.Knob, p.Bytes, p.IoU, p.Exact, p.Frames)
	}
	fmt.Println("\nB) Reed-Solomon parity under 3-byte/packet corruption:")
	fmt.Printf("   %4s %8s %7s %8s\n", "ecc", "bytes", "IoU", "exact")
	for _, p := range ecc {
		fmt.Printf("   %4d %8d %7.3f %5d/%d\n", p.Knob, p.Bytes, p.IoU, p.Exact, p.Frames)
	}

	path := fmt.Sprintf("%s/semdist.png", *out)
	if err := chart(path, gop); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\nwrote %s (bytes -> meaning preserved, by keyframe interval)\n", path)
}

func chart(path string, pts []semdist.Point) error {
	const W, H, m = 520, 320, 48
	c := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(c, c.Bounds(), &image.Uniform{color.RGBA{16, 16, 24, 255}}, image.Point{}, draw.Src)
	ink := color.RGBA{210, 214, 224, 255}
	axis := color.RGBA{70, 74, 92, 255}
	dot := color.RGBA{90, 200, 200, 255}

	minB, maxB := pts[0].Bytes, pts[0].Bytes
	for _, p := range pts {
		if p.Bytes < minB {
			minB = p.Bytes
		}
		if p.Bytes > maxB {
			maxB = p.Bytes
		}
	}
	if maxB == minB {
		maxB = minB + 1
	}
	px := func(b int) int { return m + (W-2*m)*(b-minB)/(maxB-minB) }
	py := func(iou float64) int { return (H - m) - int(float64(H-2*m)*iou) } // IoU 0..1

	// axes
	fillRect(c, m, H-m, W-2*m, 2, axis)
	fillRect(c, m, m, 2, H-2*m, axis)
	microfont.Draw(c, m, 14, 2, "MEANING VS BYTES", color.RGBA{240, 200, 90, 255})
	microfont.Draw(c, 6, m-2, 1, "IOU1", ink)
	microfont.Draw(c, 6, H-m-6, 1, "IOU0", ink)
	microfont.Draw(c, W-m-30, H-m+8, 1, "BYTES", ink)

	// polyline + dots
	prevX, prevY := -1, -1
	for _, p := range pts {
		x, y := px(p.Bytes), py(p.IoU)
		if prevX >= 0 {
			line(c, prevX, prevY, x, y, color.RGBA{60, 120, 120, 255})
		}
		fillRect(c, x-3, y-3, 6, 6, dot)
		microfont.Draw(c, x-6, y-14, 1, fmt.Sprintf("G%d", p.Knob), ink)
		prevX, prevY = x, y
	}
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	return png.Encode(fp, c)
}

func fillRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{col}, image.Point{}, draw.Src)
}

func line(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	for {
		fillRect(img, x0, y0, 2, 2, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
func sign(a int) int {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}
