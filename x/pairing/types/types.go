package types

type ClientUsedCU struct {
	TotalUsed uint64
	Providers map[string]uint64
}

type ClientProviderOverusedCUPrecent struct {
	TotalOverusedPrecent    float64
	OverusedPrecentProvider float64
}
