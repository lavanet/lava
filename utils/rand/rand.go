package rand

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"
)

func generateSeed(data []byte) int64 {
	sum256 := sha256.Sum256(data)
	seed := int64(binary.LittleEndian.Uint64(sum256[0:8]))
	return seed
}

// New returns a new RPNG instance properly seeded
func New(data []byte) *rand.Rand {
	seed := generateSeed(data)
	source := rand.NewSource(seed)
	return rand.New(source)
}

// Seed re-seeds an existing RPNG instance
func Seed(rng *rand.Rand, data []byte) {
	seed := generateSeed(data)
	rng.Seed(seed)
}

// rand wrapper for protocol structs which hosts the same seed for deterministic unified random distribution
// we set the seed once for the entire process.
type threadSafeRand struct {
	lock sync.Mutex // we have no reads, just writes, so using a sync.Mutex.
	rand *rand.Rand
}

func (t *threadSafeRand) Intn(n int) int {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.Intn(n)
}

func (t *threadSafeRand) Float64() float64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.Float64()
}

func (t *threadSafeRand) Uint32() uint32 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.Uint32()
}

func (t *threadSafeRand) Uint64() uint64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.Uint64()
}

func (t *threadSafeRand) Int63() int64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.Int63()
}

func (t *threadSafeRand) Int63n(n int64) int64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.Int63n(n)
}

func (t *threadSafeRand) NormFloat64() float64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rand.NormFloat64()
}

var protocolRand *threadSafeRand

func Initialized() bool {
	return protocolRand != nil
}

func InitRandomSeed() {
	seed := time.Now().UnixNano()
	protocolRand = &threadSafeRand{rand: rand.New(rand.NewSource(seed))}
}

func SetSpecificSeed(seed int64) {
	protocolRand = &threadSafeRand{rand: rand.New(rand.NewSource(seed))}
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

func NormFloat64() float64 {
	PanicIfProtocolRandNotInitialized()
	return protocolRand.NormFloat64()
}
