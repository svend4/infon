package glyphqr

import "errors"

// fec.go adds a byte-stream Reed-Solomon layer for transports that lose or
// corrupt bytes (e.g. a semantic video call). The original length is prepended
// to the data and protected together with it, so there is NO vulnerable header:
// the payload is split into 255-ecc blocks, each given ecc parity, then the
// blocks are INTERLEAVED column-wise so a contiguous burst (a torn-out chunk)
// spreads thinly across blocks and is corrected per block. A burst up to about
// nblocks*(ecc/2) bytes — anywhere, including over the length — survives.

// RSEncodeBytes protects arbitrary-length data with interleaved Reed-Solomon.
func RSEncodeBytes(data []byte, ecc int) []byte {
	if ecc < 2 {
		ecc = 2
	}
	if ecc > 32 {
		ecc = 32
	}
	payload := append([]byte{byte(len(data) >> 16), byte(len(data) >> 8), byte(len(data))}, data...)
	K := 255 - ecc
	nb := (len(payload) + K - 1) / K
	if nb < 1 {
		nb = 1
	}
	blocks := make([][]byte, nb)
	for i := 0; i < nb; i++ {
		blk := make([]byte, K)
		s, e := i*K, i*K+K
		if e > len(payload) {
			e = len(payload)
		}
		if s < len(payload) {
			copy(blk, payload[s:e])
		}
		blocks[i] = rsEncode(blk, ecc) // 255 bytes
	}
	out := make([]byte, 0, nb*255)
	for col := 0; col < 255; col++ {
		for i := 0; i < nb; i++ {
			out = append(out, blocks[i][col])
		}
	}
	return out
}

// RSDecodeBytes inverts RSEncodeBytes, correcting up to ecc/2 byte errors per
// 255-byte block (so a spread or interleaved burst, anywhere, is recovered).
func RSDecodeBytes(w []byte, ecc int) ([]byte, error) {
	if ecc < 2 {
		ecc = 2
	}
	if ecc > 32 {
		ecc = 32
	}
	if len(w) == 0 || len(w)%255 != 0 {
		return nil, errors.New("glyphqr: bad stream length")
	}
	nb := len(w) / 255
	blocks := make([][]byte, nb)
	for i := range blocks {
		blocks[i] = make([]byte, 255)
	}
	k := 0
	for col := 0; col < 255; col++ {
		for i := 0; i < nb; i++ {
			blocks[i][col] = w[k]
			k++
		}
	}
	payload := make([]byte, 0, nb*(255-ecc))
	for i := 0; i < nb; i++ {
		msg, err := rsDecode(blocks[i], ecc)
		if err != nil {
			return nil, err
		}
		payload = append(payload, msg...)
	}
	if len(payload) < 3 {
		return nil, errors.New("glyphqr: truncated payload")
	}
	n := int(payload[0])<<16 | int(payload[1])<<8 | int(payload[2])
	if 3+n > len(payload) {
		return nil, errors.New("glyphqr: length overflow")
	}
	return payload[3 : 3+n], nil
}
