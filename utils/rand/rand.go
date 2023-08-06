package rand

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"time"

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

// rand wrapper for protocol structs which hosts the same seed for deterministic unified random distribution
// we set the seed once for the entire process.
var protocolRand *rand.Rand

func InitRandomSeed() {
	seed := time.Now().UnixNano()
	protocolRand = rand.New(rand.NewSource(seed))
}

func SetSpecificSeed(seed int64) {
	protocolRand = rand.New(rand.NewSource(seed))
}

func Intn(n int) int {
	return protocolRand.Intn(n)
}

func Float64() float64 {
	return protocolRand.Float64()
}

func Uint32() uint32 {
	return protocolRand.Uint32()
}

func Int63() int64 {
	return protocolRand.Int63()
}

func Int63n(n int64) int64 {
	return protocolRand.Int63n(n)
}

func NormFloat64() float64 {
	return protocolRand.NormFloat64()
}
