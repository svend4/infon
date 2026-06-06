// Package glyphqr encodes data into a "glyph-QR": a grid of glyph-symbols
// carrying Reed-Solomon parity, so the code decodes correctly even when several
// cells are wrong (a model misplaced a piece, or the picture is damaged). This
// file is the Reed-Solomon engine over GF(256) (primitive polynomial 0x11d,
// generator 2) — a faithful port of the standard "Reed-Solomon for coders"
// algorithm: it corrects up to ecc/2 byte errors per codeword.
package glyphqr

import "errors"

var (
	gfExp [512]byte
	gfLog [256]byte
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(x)
		gfLog[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11d
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[int(gfLog[a])+int(gfLog[b])]
}

func gfDiv(a, b byte) byte {
	if a == 0 {
		return 0
	}
	return gfExp[(int(gfLog[a])-int(gfLog[b])+255)%255]
}

func gfPow(a byte, n int) byte {
	if a == 0 {
		if n == 0 {
			return 1
		}
		return 0
	}
	e := (int(gfLog[a]) * n) % 255
	if e < 0 {
		e += 255
	}
	return gfExp[e]
}

func gfInverse(a byte) byte { return gfExp[(255-int(gfLog[a]))%255] }

func polyScale(p []byte, x byte) []byte {
	r := make([]byte, len(p))
	for i := range p {
		r[i] = gfMul(p[i], x)
	}
	return r
}

func polyAdd(a, b []byte) []byte {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	r := make([]byte, n)
	for i := range a {
		r[i+n-len(a)] = a[i]
	}
	for i := range b {
		r[i+n-len(b)] ^= b[i]
	}
	return r
}

func polyMul(a, b []byte) []byte {
	r := make([]byte, len(a)+len(b)-1)
	for i := range a {
		for j := range b {
			r[i+j] ^= gfMul(a[i], b[j])
		}
	}
	return r
}

func polyEval(p []byte, x byte) byte {
	y := p[0]
	for i := 1; i < len(p); i++ {
		y = gfMul(y, x) ^ p[i]
	}
	return y
}

func polyDiv(dividend, divisor []byte) (quot, rem []byte) {
	out := append([]byte(nil), dividend...)
	for i := 0; i < len(dividend)-(len(divisor)-1); i++ {
		coef := out[i]
		if coef != 0 {
			for j := 1; j < len(divisor); j++ {
				if divisor[j] != 0 {
					out[i+j] ^= gfMul(divisor[j], coef)
				}
			}
		}
	}
	sep := len(out) - (len(divisor) - 1)
	return out[:sep], out[sep:]
}

func rsGenPoly(nsym int) []byte {
	g := []byte{1}
	for i := 0; i < nsym; i++ {
		g = polyMul(g, []byte{1, gfPow(2, i)})
	}
	return g
}

// rsEncode returns the full codeword: message bytes followed by nsym parity.
func rsEncode(msg []byte, nsym int) []byte {
	gen := rsGenPoly(nsym)
	out := make([]byte, len(msg)+nsym)
	copy(out, msg)
	for i := 0; i < len(msg); i++ {
		coef := out[i]
		if coef != 0 {
			for j := 1; j < len(gen); j++ {
				out[i+j] ^= gfMul(gen[j], coef)
			}
		}
	}
	copy(out, msg)
	return out
}

func rsSyndromes(msg []byte, nsym int) []byte {
	s := make([]byte, nsym+1)
	for i := 0; i < nsym; i++ {
		s[i+1] = polyEval(msg, gfPow(2, i))
	}
	return s
}

func rsErrataLocator(ePos []int) []byte {
	eLoc := []byte{1}
	for _, i := range ePos {
		eLoc = polyMul(eLoc, polyAdd([]byte{1}, []byte{gfPow(2, i), 0}))
	}
	return eLoc
}

func rsErrorEvaluator(synd, errLoc []byte, nsym int) []byte {
	div := make([]byte, nsym+2)
	div[0] = 1
	_, rem := polyDiv(polyMul(synd, errLoc), div)
	return rem
}

func rsErrorLocator(synd []byte, nsym int) ([]byte, error) {
	errLoc := []byte{1}
	oldLoc := []byte{1}
	syndShift := len(synd) - nsym
	for i := 0; i < nsym; i++ {
		k := i + syndShift
		delta := synd[k]
		for j := 1; j < len(errLoc); j++ {
			delta ^= gfMul(errLoc[len(errLoc)-1-j], synd[k-j])
		}
		oldLoc = append(oldLoc, 0)
		if delta != 0 {
			if len(oldLoc) > len(errLoc) {
				newLoc := polyScale(oldLoc, delta)
				oldLoc = polyScale(errLoc, gfInverse(delta))
				errLoc = newLoc
			}
			errLoc = polyAdd(errLoc, polyScale(oldLoc, delta))
		}
	}
	for len(errLoc) > 0 && errLoc[0] == 0 {
		errLoc = errLoc[1:]
	}
	if (len(errLoc)-1)*2 > nsym {
		return nil, errors.New("glyphqr: too many errors to correct")
	}
	return errLoc, nil
}

func rsFindErrors(errLocRev []byte, nmess int) ([]int, error) {
	errs := len(errLocRev) - 1
	var pos []int
	for i := 0; i < nmess; i++ {
		if polyEval(errLocRev, gfPow(2, i)) == 0 {
			pos = append(pos, nmess-1-i)
		}
	}
	if len(pos) != errs {
		return nil, errors.New("glyphqr: could not locate errors")
	}
	return pos, nil
}

func reversed(p []byte) []byte {
	r := make([]byte, len(p))
	for i := range p {
		r[len(p)-1-i] = p[i]
	}
	return r
}

func rsCorrectErrata(msg, synd []byte, errPos []int) []byte {
	coefPos := make([]int, len(errPos))
	for i, p := range errPos {
		coefPos[i] = len(msg) - 1 - p
	}
	errLoc := rsErrataLocator(coefPos)
	errEval := reversed(rsErrorEvaluator(reversed(synd), errLoc, len(errLoc)-1))

	X := make([]byte, len(coefPos))
	for i, cp := range coefPos {
		X[i] = gfPow(2, cp-255)
	}
	E := make([]byte, len(msg))
	for i, Xi := range X {
		XiInv := gfInverse(Xi)
		errLocPrime := byte(1)
		for j := range X {
			if j != i {
				errLocPrime = gfMul(errLocPrime, byte(1)^gfMul(XiInv, X[j]))
			}
		}
		y := polyEval(reversed(errEval), XiInv)
		y = gfMul(Xi, y)
		if errLocPrime != 0 {
			E[errPos[i]] = gfDiv(y, errLocPrime)
		}
	}
	return polyAdd(msg, E)
}

// rsDecode corrects up to nsym/2 byte errors in codeword cw (message+parity),
// returning the corrected message (without parity).
func rsDecode(cw []byte, nsym int) ([]byte, error) {
	msg := append([]byte(nil), cw...)
	synd := rsSyndromes(msg, nsym)
	clean := true
	for _, s := range synd {
		if s != 0 {
			clean = false
			break
		}
	}
	if clean {
		return msg[:len(msg)-nsym], nil
	}
	errLoc, err := rsErrorLocator(synd, nsym)
	if err != nil {
		return nil, err
	}
	pos, err := rsFindErrors(reversed(errLoc), len(msg))
	if err != nil {
		return nil, err
	}
	msg = rsCorrectErrata(msg, synd, pos)
	synd = rsSyndromes(msg, nsym)
	for _, s := range synd {
		if s != 0 {
			return nil, errors.New("glyphqr: decode failed (too many errors)")
		}
	}
	return msg[:len(msg)-nsym], nil
}
