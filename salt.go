package zrx

import (
	"crypto/rand"
	"math/big"
)

// GeneratePseudoRandomSalt generates a pseudo-random integer bound by (2^256 - 1)
func GeneratePseudoRandomSalt() (*big.Int, error) {
	// set the max to 2^256 - 1 (the maximum unsigned 256-bit integer)
	u265p1 := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	max := new(big.Int).Sub(u265p1, big.NewInt(1))

	return rand.Int(rand.Reader, max)
}
