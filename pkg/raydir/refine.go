package raydir

import (
	"image"

	"github.com/svend4/infon/pkg/raytrace"
)

// refine.go progressively sharpens a path-traced view while it holds still. A
// Refiner folds a small batch of samples into a running average each frame; when
// the camera changes it starts over. So a moving walk stays responsive at a few
// samples, and the moment you stop, the image converges to a clean render — no
// restart, no waiting.

// Refiner accumulates path-traced samples for a view, resetting when the camera
// moves. Call Frame each tick.
type Refiner struct {
	acc     *raytrace.Accumulator
	opt     raytrace.PathOptions
	batch   int
	maxSpp  int
	last    raytrace.Camera
	started bool
}

// NewRefiner makes a Refiner for a w x h image that adds `batch` samples/frame up
// to `maxSpp` total, using opt for the path-trace settings (its Samples is ignored
// in favour of batch).
func NewRefiner(w, h, batch, maxSpp int, opt raytrace.PathOptions) *Refiner {
	if batch < 1 {
		batch = 1
	}
	if maxSpp < batch {
		maxSpp = batch
	}
	return &Refiner{acc: raytrace.NewAccumulator(w, h), opt: opt, batch: batch, maxSpp: maxSpp}
}

// Reset forces the next Frame to start a fresh accumulation (e.g. the scene
// changed even though the camera did not).
func (r *Refiner) Reset() { r.started = false }

// Samples reports how many samples per pixel have accumulated for the current view.
func (r *Refiner) Samples() int { return r.acc.Samples() }

// Frame returns the current refined image: if the camera moved (or Reset was
// called) it restarts; otherwise it folds in another batch (until maxSpp) so a held
// view keeps sharpening.
func (r *Refiner) Frame(scene *raytrace.Scene, cam raytrace.Camera) image.Image {
	if !r.started || cam != r.last {
		r.acc.Reset()
		r.last = cam
		r.started = true
	}
	if r.acc.Samples() < r.maxSpp {
		o := r.opt
		o.Samples = r.batch
		r.acc.AddSamples(scene, cam, o)
	}
	return r.acc.Image()
}
