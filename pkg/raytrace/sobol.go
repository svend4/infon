// sobol.go adds low-discrepancy ("quasi-Monte-Carlo") sampling. Pure random
// sampling clumps and leaves gaps, so noise falls only as 1/sqrt(spp); a
// stratified Sobol (0,2)-sequence spreads samples evenly and converges markedly
// faster on the smooth, low-dimensional parts of the integrand — pixel
// anti-aliasing and the lens (depth of field).
//
// The first 2^m points of a (0,2)-sequence fall exactly one per cell of every
// dyadic 2^a x 2^b grid (a+b=m): perfect 2-D stratification. To decorrelate
// pixels without losing that property we apply hash-based Owen scrambling
// (Laine-Karras / Burley 2020), which permutes points within nested dyadic
// intervals — so each pixel gets its own well-stratified, blue-noise-like set,
// and averaging the scrambled sets stays unbiased. Clean-room implementation.
package raytrace

import "math/bits"

// sobolDim0 is the van der Corput sequence (radical inverse base 2): reverse the
// bits of i into the high bits of a 32-bit fixed-point fraction.
func sobolDim0(i uint32) uint32 { return bits.Reverse32(i) }

// sobolDim1 is the second Sobol dimension; with sobolDim0 it forms the classic
// (0,2)-sequence. x_i = XOR of the direction numbers for the set bits of i.
func sobolDim1(i uint32) uint32 {
	var r uint32
	for v := uint32(1) << 31; i != 0; i >>= 1 {
		if i&1 != 0 {
			r ^= v
		}
		v ^= v >> 1
	}
	return r
}

// laineKarras is the bit-permutation hash at the core of hash-based Owen
// scrambling (Laine & Karras 2011; constants from Burley 2020).
func laineKarras(x, seed uint32) uint32 {
	x += seed
	x ^= x * 0x6c50b47c
	x ^= x * 0xb82f1e52
	x ^= x * 0xc7afe638
	x ^= x * 0x8d22f6e6
	return x
}

// owenScramble applies a nested-uniform (Owen) scramble: reverse so the most
// significant bit leads, permute with the hash, reverse back. It preserves the
// (0,2) stratification while decorrelating the sequence per seed.
func owenScramble(x, seed uint32) uint32 {
	x = bits.Reverse32(x)
	x = laineKarras(x, seed)
	return bits.Reverse32(x)
}

// hashU32 is a PCG-style finalizer used to derive independent per-group scramble
// seeds from a pixel's base seed.
func hashU32(x uint32) uint32 {
	x ^= x >> 16
	x *= 0x7feb352d
	x ^= x >> 15
	x *= 0x846ca68b
	x ^= x >> 16
	return x
}

const sobolInv = 1.0 / 4294967296.0 // 1 / 2^32
