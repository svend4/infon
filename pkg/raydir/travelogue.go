package raydir

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/svend4/infon/pkg/microfont"
)

// travelogue.go remembers a journey. As you walk, the world captures moments —
// where you were (the place's name), the time of day, and a small picture of the
// view — and at the end assembles them into one illustrated page: a postcard
// montage of the trip, each thumbnail captioned with its place and a little clock.
// It is the walk's memory turned into a keepsake.

// Moment is one captured view on the journey.
type Moment struct {
	Place string
	Time  float64 // time of day in [0,1)
	Thumb image.Image
}

// Travelogue collects the journey's moments and renders them as a page.
type Travelogue struct {
	Title   string
	Moments []Moment
	max     int
}

// NewTravelogue starts an empty travelogue (keeping at most the last 24 moments).
func NewTravelogue(title string) *Travelogue { return &Travelogue{Title: title, max: 24} }

// Capture records a moment (a thumbnail of the view at a named place and time).
func (tl *Travelogue) Capture(place string, timeOfDay float64, thumb image.Image) {
	tl.Moments = append(tl.Moments, Moment{Place: place, Time: timeOfDay, Thumb: thumb})
	if len(tl.Moments) > tl.max { // keep the most recent
		tl.Moments = tl.Moments[len(tl.Moments)-tl.max:]
	}
}

// Len is how many moments have been captured.
func (tl *Travelogue) Len() int { return len(tl.Moments) }

// clockStr formats a 0..1 time of day as HH:MM.
func clockStr(t float64) string {
	mins := int((t - float64(int(t))) * 24 * 60)
	if mins < 0 {
		mins += 24 * 60
	}
	return fmt.Sprintf("%02d:%02d", mins/60, mins%60)
}

// Thumbnail nearest-neighbour scales an image to w×h (for compact moment images).
func Thumbnail(src image.Image, w, h int) image.Image {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*b.Dy()/h
		for x := 0; x < w; x++ {
			sx := b.Min.X + x*b.Dx()/w
			out.Set(x, y, src.At(sx, sy))
		}
	}
	return out
}

// Render lays the moments out as a captioned grid `cols` wide, with the title
// across the top — the finished travelogue page.
func (tl *Travelogue) Render(cols int) image.Image {
	if cols < 1 {
		cols = 1
	}
	tw, th := 160, 120 // default cell picture size
	if len(tl.Moments) > 0 {
		b := tl.Moments[0].Thumb.Bounds()
		tw, th = b.Dx(), b.Dy()
	}
	const m = 8                              // margin / gap
	_, fh := microfont.Measure("Ag", 1)      // caption height
	capH := fh + 4                           // place + clock on one line
	_, tfh := microfont.Measure(tl.Title, 2) // title bar height
	titleH := tfh + 2*m
	cellW, cellH := tw, th+capH
	rows := (len(tl.Moments) + cols - 1) / cols
	if rows < 1 {
		rows = 1
	}
	W := cols*cellW + (cols+1)*m
	H := titleH + rows*cellH + (rows+1)*m

	page := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(page, page.Bounds(), &image.Uniform{C: color.RGBA{R: 18, G: 18, B: 22, A: 255}}, image.Point{}, draw.Src)
	microfont.Draw(page, m, m, 2, tl.Title, color.RGBA{R: 240, G: 230, B: 200, A: 255})

	for i, mo := range tl.Moments {
		cx := m + (i%cols)*(cellW+m)
		cy := titleH + m + (i/cols)*(cellH+m)
		draw.Draw(page, image.Rect(cx, cy, cx+tw, cy+th), mo.Thumb, mo.Thumb.Bounds().Min, draw.Src)
		caption := mo.Place + "  " + clockStr(mo.Time)
		microfont.Draw(page, cx+2, cy+th+2, 1, caption, color.RGBA{R: 235, G: 235, B: 240, A: 255})
	}
	return page
}
