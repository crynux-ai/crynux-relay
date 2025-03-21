package utils

import (
	"math/bits"
)

func HammingDistance(a, b []byte) int {
	distance := 0

	for i := 0; i < len(a); i++ {
		xor := a[i] ^ b[i]

		distance += bits.OnesCount8(xor)
	}

	return distance
}
