package rand

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	mathrand "math/rand"
	"sync"
)

func generateSeed(data []byte) int64 {
	sum256 := sha256.Sum256(data)
	seed := int64(binary.LittleEndian.Uint64(sum256[0:8]))
	return seed
}

// New returns a new deterministic PRNG instance seeded from the given data.
// This is used for consensus-critical operations where deterministic randomness is required.
func New(data []byte) *mathrand.Rand {
	seed := generateSeed(data)
	source := mathrand.NewSource(seed)
	return mathrand.New(source)
}

// Seed re-seeds an existing deterministic PRNG instance with new data.
func Seed(rng *mathrand.Rand, data []byte) {
	seed := generateSeed(data)
	rng.Seed(seed)
}

// threadSafeRand wraps crypto/rand for thread-safe, cryptographically secure random number generation.
// This is used for the global protocolRand instance where security and uniformity are prioritized
// over determinism.
type threadSafeRand struct {
	lock sync.Mutex // we have no reads, just writes, so using a sync.Mutex.
}

func (t *threadSafeRand) Intn(n int) int {
	t.lock.Lock()
	defer t.lock.Unlock()
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	maxVal := big.NewInt(int64(n))
	result, err := cryptorand.Int(cryptorand.Reader, maxVal)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return int(result.Int64())
}

func (t *threadSafeRand) Float64() float64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	// Generate a random 53-bit integer (mantissa size for float64)
	// to ensure uniform distribution across [0, 1)
	maxVal := big.NewInt(1 << 53)
	n, err := cryptorand.Int(cryptorand.Reader, maxVal)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	// Convert to float64 in range [0, 1)
	return float64(n.Int64()) / float64(int64(1<<53))
}

func (t *threadSafeRand) Uint32() uint32 {
	t.lock.Lock()
	defer t.lock.Unlock()
	maxVal := big.NewInt(1 << 32)
	result, err := cryptorand.Int(cryptorand.Reader, maxVal)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return uint32(result.Uint64())
}

func (t *threadSafeRand) Uint64() uint64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	// Generate 8 random bytes and convert to uint64
	var b [8]byte
	_, err := cryptorand.Read(b[:])
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return binary.LittleEndian.Uint64(b[:])
}

func (t *threadSafeRand) Int63() int64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	// Int63 returns a non-negative int64, so max is 2^63
	maxVal := new(big.Int).SetUint64(1 << 63)
	result, err := cryptorand.Int(cryptorand.Reader, maxVal)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return result.Int64()
}

func (t *threadSafeRand) Int63n(n int64) int64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	if n <= 0 {
		panic("invalid argument to Int63n")
	}
	maxVal := big.NewInt(n)
	result, err := cryptorand.Int(cryptorand.Reader, maxVal)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return result.Int64()
}

var protocolRand *threadSafeRand

func Initialized() bool {
	return protocolRand != nil
}

func InitRandomSeed() {
	// Seed is no longer needed as crypto/rand is self-seeding
	// This function is kept for API compatibility
	protocolRand = &threadSafeRand{}
}

func SetSpecificSeed(seed int64) {
	// Seed is no longer used as crypto/rand is self-seeding and non-deterministic
	// This function is kept for API compatibility but has no effect
	// For deterministic randomness, use New(data) instead
	_ = seed
	protocolRand = &threadSafeRand{}
}

func PanicIfProtocolRandNotInitialized() {
	if protocolRand == nil {
		panic("rand.InitRandomSeed() must be called before using the rand package")
	}
}

func Intn(n int) int {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.Intn(n)
}

func Float64() float64 {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.Float64()
}

func Uint32() uint32 {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.Uint32()
}

func Uint64() uint64 {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.Uint64()
}

func Int63() int64 {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.Int63()
}

func Int63n(n int64) int64 {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.Int63n(n)
}
