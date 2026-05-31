package color

import "math"

// OKLab is a perceptual color space (by Björn Ottosson) in which Euclidean
// distance approximates perceived color difference far better than raw RGB.
// Using it for block color selection makes gradients and skin tones look right
// at terminal resolution, where *which* two colors we pick matters more than
// how many pixels we have.
//
// L is perceptual lightness (~0..1); A is green→red; B is blue→yellow.
type OKLab struct {
	L, A, B float64
}

// srgbToLinear converts one 0..255 sRGB channel to linear light (0..1).
func srgbToLinear(c uint8) float64 {
	x := float64(c) / 255.0
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

// linearToSrgb converts a linear-light value (0..1) back to a 0..255 sRGB channel.
func linearToSrgb(x float64) uint8 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 255
	}
	var s float64
	if x <= 0.0031308 {
		s = x * 12.92
	} else {
		s = 1.055*math.Pow(x, 1.0/2.4) - 0.055
	}
	return uint8(math.Round(s * 255.0))
}

// ToOKLab converts a 24-bit sRGB color to OKLab.
func (c RGB) ToOKLab() OKLab {
	r := srgbToLinear(c.R)
	g := srgbToLinear(c.G)
	b := srgbToLinear(c.B)

	// Linear sRGB -> LMS (cone response), then cube root, then -> OKLab.
	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b

	l_ := math.Cbrt(l)
	m_ := math.Cbrt(m)
	s_ := math.Cbrt(s)

	return OKLab{
		L: 0.2104542553*l_ + 0.7936177850*m_ - 0.0040720468*s_,
		A: 1.9779984951*l_ - 2.4285922050*m_ + 0.4505937099*s_,
		B: 0.0259040371*l_ + 0.7827717662*m_ - 0.8086757660*s_,
	}
}

// ToRGB converts an OKLab color back to 24-bit sRGB (clamped).
func (o OKLab) ToRGB() RGB {
	l_ := o.L + 0.3963377774*o.A + 0.2158037573*o.B
	m_ := o.L - 0.1055613458*o.A - 0.0638541728*o.B
	s_ := o.L - 0.0894841775*o.A - 1.2914855480*o.B

	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_

	r := +4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s

	return RGB{R: linearToSrgb(r), G: linearToSrgb(g), B: linearToSrgb(b)}
}

// PerceptualDistance returns the squared OKLab distance between two colors. Like
// Distance it omits the final sqrt (monotonic), so it is a drop-in comparison
// key that ranks colors by *perceived* difference instead of raw RGB difference.
func (c RGB) PerceptualDistance(other RGB) float64 {
	a := c.ToOKLab()
	b := other.ToOKLab()
	dl := a.L - b.L
	da := a.A - b.A
	db := a.B - b.B
	return dl*dl + da*da + db*db
}

// BlendOKLab interpolates two colors in OKLab space (perceptually even), unlike
// Blend which interpolates in gamma sRGB. alpha 0 = c, 1 = other.
func (c RGB) BlendOKLab(other RGB, alpha float64) RGB {
	if alpha <= 0 {
		return c
	}
	if alpha >= 1 {
		return other
	}
	a := c.ToOKLab()
	b := other.ToOKLab()
	return OKLab{
		L: a.L + (b.L-a.L)*alpha,
		A: a.A + (b.A-a.A)*alpha,
		B: a.B + (b.B-a.B)*alpha,
	}.ToRGB()
}
