// textures.go adds surface textures and image-based environment maps. A Texture
// maps a UV coordinate (and the world point, for solid/3-D textures) to a colour
// in 0..1. SolidTex is a constant; CheckerTex is a 3-D checkerboard that works on
// any surface without UVs; ImageTex samples an image by UV and doubles as an
// equirectangular sky when assigned to Scene.SkyTex.
package raytrace

import (
	"image"
	"math"
)

// Texture maps a surface point to a colour (channels 0..1).
type Texture interface {
	At(u, v float64, p Vec3) Vec3
}

// SolidTex is a constant colour (useful for env-map sky tests / flat fills).
type SolidTex struct{ C Vec3 }

// At implements Texture.
func (t SolidTex) At(_, _ float64, _ Vec3) Vec3 { return t.C }

// CheckerTex is a 3-D checkerboard between two colours; Scale is the cell size.
type CheckerTex struct {
	A, B  Vec3
	Scale float64
}

// At implements Texture (checker by world position, so no UVs are required).
func (t CheckerTex) At(_, _ float64, p Vec3) Vec3 {
	s := t.Scale
	if s <= 0 {
		s = 1
	}
	cx := int(math.Floor(p.X / s))
	cy := int(math.Floor(p.Y / s))
	cz := int(math.Floor(p.Z / s))
	if (cx+cy+cz)&1 == 0 {
		return t.A
	}
	return t.B
}

// ImageTex samples an image by UV (0..1, v top-down). With Wrap it tiles,
// otherwise it clamps. sRGB is decoded toward linear so it composes with the
// renderer's output gamma. Used both for surfaces and as an equirectangular
// environment map.
type ImageTex struct {
	Img  image.Image
	Wrap bool
}

// At implements Texture.
func (t ImageTex) At(u, v float64, _ Vec3) Vec3 {
	if t.Img == nil {
		return Vec3{}
	}
	b := t.Img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return Vec3{}
	}
	if t.Wrap {
		u -= math.Floor(u)
		v -= math.Floor(v)
	} else {
		u, v = clamp01(u), clamp01(v)
	}
	x := b.Min.X + int(u*float64(w-1)+0.5)
	y := b.Min.Y + int(v*float64(h-1)+0.5)
	r, g, bl, _ := t.Img.At(x, y).RGBA()
	f := func(c uint32) float64 { x := float64(c) / 65535; return x * x } // sRGB -> ~linear
	return Vec3{X: f(r), Y: f(g), Z: f(bl)}
}
