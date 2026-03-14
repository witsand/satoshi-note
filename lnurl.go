package main

import (
	"fmt"
	"math/big"
	"strings"
)

// lnurlEncode encodes a URL as a bech32 LNURL string (uppercase).
func lnurlEncode(url string) (string, error) {
	data := []byte(url)
	conv, err := bech32ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", err
	}
	encoded, err := bech32Encode("lnurl", conv)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(encoded), nil
}

// --- Minimal bech32 implementation ---

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

func bech32PolymodStep(pre uint32) uint32 {
	b := pre >> 25
	return ((pre & 0x1FFFFFF) << 5) ^
		(0x3b6a57b2 * ((b >> 0) & 1)) ^
		(0x26508e6d * ((b >> 1) & 1)) ^
		(0x1ea119fa * ((b >> 2) & 1)) ^
		(0x3d4233dd * ((b >> 3) & 1)) ^
		(0x2a1462b3 * ((b >> 4) & 1))
}

func bech32HrpExpand(hrp string) []byte {
	h := []byte(hrp)
	ret := make([]byte, len(h)*2+1)
	for i, c := range h {
		ret[i] = c >> 5
		ret[i+len(h)+1] = c & 31
	}
	ret[len(h)] = 0
	return ret
}

func bech32Checksum(hrp string, data []byte) []byte {
	values := append(bech32HrpExpand(hrp), data...)
	polymod := uint32(1)
	for _, v := range values {
		polymod = bech32PolymodStep(polymod) ^ uint32(v)
	}
	for i := 0; i < 6; i++ {
		polymod = bech32PolymodStep(polymod)
	}
	polymod ^= 1
	ret := make([]byte, 6)
	for i := range ret {
		ret[i] = byte((polymod >> (5 * (5 - i))) & 31)
	}
	return ret
}

func bech32Encode(hrp string, data []byte) (string, error) {
	combined := append(data, bech32Checksum(hrp, data)...)
	var sb strings.Builder
	sb.WriteString(hrp)
	sb.WriteByte('1')
	for _, b := range combined {
		if b >= 32 {
			return "", fmt.Errorf("bech32: invalid data byte %d", b)
		}
		sb.WriteByte(bech32Charset[b])
	}
	return sb.String(), nil
}

// bech32ConvertBits converts a byte slice from one bit-width to another.
func bech32ConvertBits(data []byte, fromBits, toBits int, pad bool) ([]byte, error) {
	acc := new(big.Int)
	bits := 0
	result := []byte{}
	maxv := (1 << toBits) - 1

	for _, value := range data {
		acc.Lsh(acc, uint(fromBits))
		acc.Or(acc, new(big.Int).SetUint64(uint64(value)))
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			b := new(big.Int).Rsh(acc, uint(bits))
			result = append(result, byte(b.Int64()&int64(maxv)))
		}
	}

	if pad {
		if bits > 0 {
			shifted := new(big.Int).Lsh(acc, uint(toBits-bits))
			result = append(result, byte(shifted.Int64()&int64(maxv)))
		}
	} else if bits >= fromBits || (new(big.Int).Lsh(acc, uint(toBits-bits))).Int64()&int64(maxv) != 0 {
		return nil, fmt.Errorf("bech32: invalid padding")
	}

	return result, nil
}
