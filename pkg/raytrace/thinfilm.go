// thinfilm.go adds thin-film interference — the shifting rainbow sheen of soap
// bubbles, oil slicks and anodised metal. A thin transparent film makes light
// reflected off its front and back surfaces interfere; the optical path
// difference 2*n*d*cos(theta_t) is a fixed length, so each wavelength is
// reinforced or cancelled depending on how many times it fits, and the colour
// shifts with both film thickness and viewing angle. This is a (partial) spectral
// effect evaluated at representative R/G/B wavelengths — the most visible piece of
// spectral rendering without converting the whole pipeline off RGB. Clean-room.
package raytrace

import "math"

// rgbWavelengths are representative wavelengths (nm) for the R, G, B channels.
var rgbWavelengths = [3]float64{630, 532, 465}

// thinFilmTint returns the per-channel interference factor (0..1) for a film of
// the given thickness (nm) and index nFilm, viewed at incidence cosine cosI in
// air. The half-wave (pi) phase shift at the denser front interface is included,
// so a vanishingly thin film cancels (dark) as it should. Total internal
// reflection returns white (no transmitted beam to interfere).
func thinFilmTint(cosI, thickness, nFilm float64) Vec3 {
	if nFilm <= 1 {
		nFilm = 1.33
	}
	cosI = math.Min(1, math.Max(0, cosI))
	sinT2 := (1 - cosI*cosI) / (nFilm * nFilm) // Snell: sinT = sinI / nFilm
	if sinT2 >= 1 {
		return Vec3{X: 1, Y: 1, Z: 1}
	}
	cosT := math.Sqrt(1 - sinT2)
	opd := 2 * nFilm * thickness * cosT // optical path difference (nm)
	factor := func(lambda float64) float64 {
		// two-beam interference with a pi shift at the front surface.
		return 0.5 * (1 + math.Cos(2*math.Pi*opd/lambda+math.Pi))
	}
	return Vec3{
		X: factor(rgbWavelengths[0]),
		Y: factor(rgbWavelengths[1]),
		Z: factor(rgbWavelengths[2]),
	}
}
