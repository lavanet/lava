package common

import (
	"context"
	"errors"
	"math"
	"time"
)

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

func ContextOutOfTime(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.DeadlineExceeded)
}

func GetRemainingTimeoutFromContext(ctx context.Context) (timeRemaining time.Duration) {
	deadline, ok := ctx.Deadline()
	if ok {
		return time.Until(deadline)
	}
	return time.Duration(math.MaxInt64)
}

func LowerContextTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if GetRemainingTimeoutFromContext(ctx) > timeout {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
