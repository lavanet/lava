package rand

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"

	"github.com/vpxyz/xorshift/xoroshiro128starstar"
)

func generateSeed(data []byte) (seed int64) {
	sum256 := sha256.Sum256(data)

	// fold the SHA-256 hash into a 64-bit seed using bitwise XOR
	for i := 0; i < len(sum256)/8; i++ {
		seed ^= int64(binary.BigEndian.Uint64(sum256[i*8 : (i+1)*8]))
	}

	return seed
}

// returns a new RPNG instance properly seeded
func New(data []byte) *rand.Rand {
	seed := generateSeed(data)
	source := xoroshiro128starstar.NewSource(seed)
	return rand.New(source)
}

// re-seeds an existing RPNG instance
func Seed(rng *rand.Rand, data []byte) {
	seed := generateSeed(data)
	rng.Seed(seed)
}
