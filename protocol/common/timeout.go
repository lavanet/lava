package common

import "time"

const (
	TimePerCU                      = uint64(100 * time.Millisecond)
	MinimumTimePerRelayDelay       = time.Second
	DataReliabilityTimeoutIncrease = 5 * time.Second
	AverageWorldLatency            = 300 * time.Millisecond
)

func LocalNodeTimePerCu(cu uint64) time.Duration {
	return BaseTimePerCU(cu) + AverageWorldLatency // TODO: remove average world latency once our providers run locally, or allow a flag that says local to make it tight, tighter timeouts are better
}

func BaseTimePerCU(cu uint64) time.Duration {
	return time.Duration(cu * TimePerCU)
}

func GetTimePerCu(cu uint64) time.Duration {
	return LocalNodeTimePerCu(cu) + MinimumTimePerRelayDelay
}
