// sky.go adds a physically-based daytime sky: the Preetham analytic model. Given a
// sun direction and an atmospheric turbidity it gives the sky's colour for any view
// direction in closed form — deep blue overhead, brightening and warming toward the
// horizon and the sun, reddening as the sun sinks. It pairs with the day/night
// cycle (the sun direction comes from the time of day) and acts as an area light in
// the path tracer. Clean-room from the published Perez/Preetham coefficients.
package raytrace

import "math"

// PreethamSky is a Preetham analytic sky for a fixed sun direction and turbidity.
type PreethamSky struct {
	sun    Vec3 // unit sun direction
	thetaS float64
	// Perez coefficients for Y (luminance), x and y chromaticity.
	ay, by, cy, dy, ey      float64
	ax, bx, cx, dx, ex      float64
	axy, bxy, cxy, dxy, exy float64
	yz, xz, yzc             float64 // zenith luminance and chromaticities
	fyN, fxN, fyzN          float64 // normalisation Perez(0, thetaS) per band
	Intensity               float64 // output scale (default 1)
}

func clampUnit(x float64) float64 {
	if x < -1 {
		return -1
	}
	if x > 1 {
		return 1
	}
	return x
}

// NewPreethamSky builds the model for a sun direction and turbidity (~2 clear .. 10
// hazy; 2.5 is a typical clear day).
func NewPreethamSky(sunDir Vec3, turbidity float64) *PreethamSky {
	if turbidity < 1.7 {
		turbidity = 1.7
	}
	s := sunDir.Norm()
	t := turbidity
	thetaS := math.Acos(clampUnit(s.Y))
	p := &PreethamSky{sun: s, thetaS: thetaS, Intensity: 1}

	// Perez distribution coefficients (linear in turbidity).
	p.ay, p.by, p.cy, p.dy, p.ey = 0.1787*t-1.4630, -0.3554*t+0.4275, -0.0227*t+5.3251, 0.1206*t-2.5771, -0.0670*t+0.3703
	p.ax, p.bx, p.cx, p.dx, p.ex = -0.0193*t-0.2592, -0.0665*t+0.0008, -0.0004*t+0.2125, -0.0641*t-0.8989, -0.0033*t+0.0452
	p.axy, p.bxy, p.cxy, p.dxy, p.exy = -0.0167*t-0.2608, -0.0950*t+0.0092, -0.0079*t+0.2102, -0.0441*t-1.6537, -0.0109*t+0.0529

	ts, ts2, ts3 := thetaS, thetaS*thetaS, thetaS*thetaS*thetaS
	t2 := t * t
	p.xz = t2*(0.00166*ts3-0.00375*ts2+0.00209*ts) +
		t*(-0.02903*ts3+0.06377*ts2-0.03202*ts+0.00394) +
		(0.11693*ts3 - 0.21196*ts2 + 0.06052*ts + 0.25886)
	p.yzc = t2*(0.00275*ts3-0.00610*ts2+0.00317*ts) +
		t*(-0.04214*ts3+0.08970*ts2-0.04153*ts+0.00516) +
		(0.15346*ts3 - 0.26756*ts2 + 0.06670*ts + 0.26688)
	chi := (4.0/9.0 - t/120.0) * (math.Pi - 2*thetaS)
	p.yz = (4.0453*t-4.9710)*math.Tan(chi) - 0.2155*t + 2.4192 // kcd/m^2
	if p.yz < 0 {
		p.yz = 0
	}
	// normalisation: Perez at the zenith for this sun (theta=0).
	p.fyN = perez(0, thetaS, p.ay, p.by, p.cy, p.dy, p.ey)
	p.fxN = perez(0, thetaS, p.ax, p.bx, p.cx, p.dx, p.ex)
	p.fyzN = perez(0, thetaS, p.axy, p.bxy, p.cxy, p.dxy, p.exy)
	return p
}

// perez is the Perez sky distribution F(theta, gamma).
func perez(theta, gamma, a, b, c, d, e float64) float64 {
	cosTheta := math.Cos(theta)
	if cosTheta < 1e-3 {
		cosTheta = 1e-3 // clamp at the horizon so exp(b/cosTheta) stays bounded
	}
	cosG := math.Cos(gamma)
	return (1 + a*math.Exp(b/cosTheta)) * (1 + c*math.Exp(d*gamma) + e*cosG*cosG)
}

// At returns the sky radiance for a view direction (unit).
func (p *PreethamSky) At(dir Vec3) Vec3 {
	y := dir.Y
	if y < 1e-3 {
		y = 1e-3 // below the horizon: clamp to the horizon band
	}
	theta := math.Acos(clampUnit(y))
	gamma := math.Acos(clampUnit(dir.Norm().Dot(p.sun)))

	bigY := p.yz * perez(theta, gamma, p.ay, p.by, p.cy, p.dy, p.ey) / p.fyN
	cx := p.xz * perez(theta, gamma, p.ax, p.bx, p.cx, p.dx, p.ex) / p.fxN
	cy := p.yzc * perez(theta, gamma, p.axy, p.bxy, p.cxy, p.dxy, p.exy) / p.fyzN
	if cy < 1e-4 || bigY < 0 {
		return Vec3{}
	}
	// xyY -> XYZ -> linear sRGB.
	bigX := (cx / cy) * bigY
	bigZ := ((1 - cx - cy) / cy) * bigY
	r := 3.2406*bigX - 1.5372*bigY - 0.4986*bigZ
	g := -0.9689*bigX + 1.8758*bigY + 0.0415*bigZ
	b := 0.0557*bigX - 0.2040*bigY + 1.0570*bigZ
	scale := p.Intensity / 25.0 // bring kcd/m^2 into the renderer's range
	return Vec3{X: math.Max(0, r) * scale, Y: math.Max(0, g) * scale, Z: math.Max(0, b) * scale}
}
