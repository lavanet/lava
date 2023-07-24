package rand

import (
	gorand "math/rand"
)

var rand *gorand.Rand

func Seed(seed int64) {
	rand = gorand.New(gorand.NewSource(seed))
}

func Intn(n int) int {
	return rand.Intn(n)
}

func Float64() float64 {
	return rand.Float64()
}

func Uint32() uint32 {
	return rand.Uint32()
}

func Int63() int64 {
	return rand.Int63()
}

func NormFloat64() float64 {
	return rand.NormFloat64()
}
