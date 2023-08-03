package rand

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"

	"github.com/vpxyz/xorshift/xoroshiro128starstar"
)

func generateSeed(data []byte) int64 {
	sum256 := sha256.Sum256(data)
	seed := int64(binary.LittleEndian.Uint64(sum256[0:8]))
	return seed
}

// New returns a new RPNG instance properly seeded
func New(data []byte) *rand.Rand {
	seed := generateSeed(data)
	source := xoroshiro128starstar.NewSource(seed)
	return rand.New(source)
}

// Seed re-seeds an existing RPNG instance
func Seed(rng *rand.Rand, data []byte) {
	seed := generateSeed(data)
	rng.Seed(seed)
}
