package video

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"
)

// BackgroundMode represents the background effect mode
type BackgroundMode int

const (
	BackgroundNone BackgroundMode = iota
	BackgroundBlur
	BackgroundImage
	BackgroundColor
)

// BackgroundProcessor handles virtual background effects
type BackgroundProcessor struct {
	mu sync.RWMutex

	mode            BackgroundMode
	blurRadius      int
	replacementImg  *image.RGBA
	backgroundColor color.RGBA

	// Segmentation threshold
	threshold uint8

	// Statistics
	framesProcessed uint64
}

// NewBackgroundProcessor creates a new background processor
func NewBackgroundProcessor() *BackgroundProcessor {
	return &BackgroundProcessor{
		mode:            BackgroundNone,
		blurRadius:      15,
		backgroundColor: color.RGBA{0, 255, 0, 255}, // Green screen default
		threshold:       128,
	}
}

// SetMode sets the background mode
func (bp *BackgroundProcessor) SetMode(mode BackgroundMode) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.mode = mode
}

// SetBlurRadius sets the blur radius (for blur mode)
func (bp *BackgroundProcessor) SetBlurRadius(radius int) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if radius < 1 {
		radius = 1
	}
	if radius > 50 {
		radius = 50
	}

	bp.blurRadius = radius
}

// SetReplacementImage sets the replacement image (for image mode)
func (bp *BackgroundProcessor) SetReplacementImage(img *image.RGBA) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.replacementImg = img
}

// SetBackgroundColor sets the background color (for color mode)
func (bp *BackgroundProcessor) SetBackgroundColor(c color.RGBA) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.backgroundColor = c
}

// Process applies background effect to a frame
func (bp *BackgroundProcessor) Process(img *image.RGBA) (*image.RGBA, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.framesProcessed++

	if bp.mode == BackgroundNone {
		return img, nil
	}

	// Simple edge-based segmentation to detect foreground
	mask := bp.createMask(img)

	// Apply effect based on mode
	switch bp.mode {
	case BackgroundBlur:
		return bp.applyBlur(img, mask), nil
	case BackgroundImage:
		if bp.replacementImg == nil {
			return img, fmt.Errorf("no replacement image set")
		}
		return bp.applyReplacement(img, mask, bp.replacementImg), nil
	case BackgroundColor:
		return bp.applyColor(img, mask, bp.backgroundColor), nil
	default:
		return img, nil
	}
}

// createMask creates a binary mask (simple edge-based segmentation)
// In a real implementation, this would use ML models like BodyPix or MediaPipe
func (bp *BackgroundProcessor) createMask(img *image.RGBA) *image.Gray {
	bounds := img.Bounds()
	mask := image.NewGray(bounds)

	// Simple center-weighted mask (assumes person is in center)
	// This is a placeholder - real implementation would use ML segmentation
	centerX := bounds.Dx() / 2
	centerY := bounds.Dy() / 2

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Distance from center
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)

			// Simple radial gradient (person in center)
			maxDist := float64(bounds.Dx() / 2)
			if dist < maxDist*0.4 {
				mask.SetGray(x, y, color.Gray{255}) // Foreground
			} else if dist < maxDist*0.6 {
				// Feathered edge
				alpha := uint8((1.0 - (dist-maxDist*0.4)/(maxDist*0.2)) * 255)
				mask.SetGray(x, y, color.Gray{alpha})
			} else {
				mask.SetGray(x, y, color.Gray{0}) // Background
			}
		}
	}

	return mask
}

// applyBlur applies blur effect to background
func (bp *BackgroundProcessor) applyBlur(img *image.RGBA, mask *image.Gray) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	// Create blurred version of the image
	blurred := bp.gaussianBlur(img, bp.blurRadius)

	// Composite based on mask
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			maskValue := mask.GrayAt(x, y).Y

			if maskValue == 255 {
				// Foreground - use original
				result.Set(x, y, img.At(x, y))
			} else if maskValue == 0 {
				// Background - use blurred
				result.Set(x, y, blurred.At(x, y))
			} else {
				// Edge - blend
				origColor := img.RGBAAt(x, y)
				blurColor := blurred.RGBAAt(x, y)

				alpha := float64(maskValue) / 255.0
				r := uint8(float64(origColor.R)*alpha + float64(blurColor.R)*(1-alpha))
				g := uint8(float64(origColor.G)*alpha + float64(blurColor.G)*(1-alpha))
				b := uint8(float64(origColor.B)*alpha + float64(blurColor.B)*(1-alpha))

				result.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}

	return result
}

// gaussianBlur applies Gaussian blur
func (bp *BackgroundProcessor) gaussianBlur(img *image.RGBA, radius int) *image.RGBA {
	if radius < 1 {
		return img
	}

	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	// Simple box blur approximation (faster than true Gaussian)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var rSum, gSum, bSum uint32
			var count uint32

			// Sample neighborhood
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					sx := x + dx
					sy := y + dy

					if sx >= bounds.Min.X && sx < bounds.Max.X &&
						sy >= bounds.Min.Y && sy < bounds.Max.Y {
						c := img.RGBAAt(sx, sy)
						rSum += uint32(c.R)
						gSum += uint32(c.G)
						bSum += uint32(c.B)
						count++
					}
				}
			}

			if count > 0 {
				result.SetRGBA(x, y, color.RGBA{
					uint8(rSum / count),
					uint8(gSum / count),
					uint8(bSum / count),
					255,
				})
			}
		}
	}

	return result
}

// applyReplacement replaces background with an image
func (bp *BackgroundProcessor) applyReplacement(img *image.RGBA, mask *image.Gray, bg *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	// Resize background if needed (simple scaling)
	bgResized := bp.resizeImage(bg, bounds.Dx(), bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			maskValue := mask.GrayAt(x, y).Y

			if maskValue == 255 {
				result.Set(x, y, img.At(x, y))
			} else if maskValue == 0 {
				result.Set(x, y, bgResized.At(x, y))
			} else {
				// Blend
				origColor := img.RGBAAt(x, y)
				bgColor := bgResized.RGBAAt(x, y)

				alpha := float64(maskValue) / 255.0
				r := uint8(float64(origColor.R)*alpha + float64(bgColor.R)*(1-alpha))
				g := uint8(float64(origColor.G)*alpha + float64(bgColor.G)*(1-alpha))
				b := uint8(float64(origColor.B)*alpha + float64(bgColor.B)*(1-alpha))

				result.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}

	return result
}

// applyColor replaces background with solid color
func (bp *BackgroundProcessor) applyColor(img *image.RGBA, mask *image.Gray, bgColor color.RGBA) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			maskValue := mask.GrayAt(x, y).Y

			if maskValue == 255 {
				result.Set(x, y, img.At(x, y))
			} else if maskValue == 0 {
				result.SetRGBA(x, y, bgColor)
			} else {
				// Blend
				origColor := img.RGBAAt(x, y)

				alpha := float64(maskValue) / 255.0
				r := uint8(float64(origColor.R)*alpha + float64(bgColor.R)*(1-alpha))
				g := uint8(float64(origColor.G)*alpha + float64(bgColor.G)*(1-alpha))
				b := uint8(float64(origColor.B)*alpha + float64(bgColor.B)*(1-alpha))

				result.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}

	return result
}

// resizeImage resizes an image (nearest neighbor)
func (bp *BackgroundProcessor) resizeImage(img *image.RGBA, width, height int) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, width, height))

	srcBounds := img.Bounds()
	scaleX := float64(srcBounds.Dx()) / float64(width)
	scaleY := float64(srcBounds.Dy()) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			result.Set(x, y, img.At(srcX, srcY))
		}
	}

	return result
}

// GetStatistics returns processing statistics
func (bp *BackgroundProcessor) GetStatistics() Statistics {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return Statistics{
		Mode:            bp.mode,
		FramesProcessed: bp.framesProcessed,
		BlurRadius:      bp.blurRadius,
	}
}

// Statistics represents background processing statistics
type Statistics struct {
	Mode            BackgroundMode
	FramesProcessed uint64
	BlurRadius      int
}

// Reset resets statistics
func (bp *BackgroundProcessor) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.framesProcessed = 0
}
