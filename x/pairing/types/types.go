package types

type ClientUsedCU struct {
	TotalOverused uint64
	Providers     map[string]uint64
}

type ClientProviderOverusedCUPercent struct {
	TotalOverusedPercent    float64
	OverusedPercentProvider float64
}
