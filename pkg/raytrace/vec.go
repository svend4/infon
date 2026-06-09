package raytrace

import "math"

// Vec3 is a 3-D vector (also reused as an RGB colour with channels in 0..1).
// Right-handed, y up. All methods are value receivers and allocation-free.
type Vec3 struct{ X, Y, Z float64 }

// Add returns a+b.
func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

// Sub returns a-b.
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

// Scale returns a*s.
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

// Mul is the component-wise (Hadamard) product, used for colour modulation.
func (a Vec3) Mul(b Vec3) Vec3 { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }

// Neg returns -a.
func (a Vec3) Neg() Vec3 { return Vec3{-a.X, -a.Y, -a.Z} }

// Dot returns the dot product a·b.
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// Cross returns the cross product a×b.
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

// LenSq returns the squared length (cheaper than Len when only comparing).
func (a Vec3) LenSq() float64 { return a.Dot(a) }

// Len returns the Euclidean length.
func (a Vec3) Len() float64 { return math.Sqrt(a.LenSq()) }

// Norm returns a unit vector in the direction of a (or a unchanged if zero).
func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l == 0 {
		return a
	}
	return a.Scale(1 / l)
}

// Reflect reflects a about the unit normal n (a - 2(a·n)n).
func (a Vec3) Reflect(n Vec3) Vec3 { return a.Sub(n.Scale(2 * a.Dot(n))) }
