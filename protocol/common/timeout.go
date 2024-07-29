package common

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
)

const (
	TimePerCU                           = uint64(100 * time.Millisecond)
	MinimumTimePerRelayDelay            = time.Second
	DataReliabilityTimeoutIncrease      = 5 * time.Second
	AverageWorldLatency                 = 300 * time.Millisecond
	CommunicateWithLocalLavaNodeTimeout = (3 * time.Second) + AverageWorldLatency
	DefaultTimeout                      = 30 * time.Second
	DefaultTimeoutLongIsh               = 1 * time.Minute
	DefaultTimeoutLong                  = 3 * time.Minute
	CacheTimeout                        = 50 * time.Millisecond
	// On subscriptions we must use context.Background(),
	// we cant have a context.WithTimeout() context, meaning we can hang for ever.
	// to avoid that we introduced a first reply timeout using a routine.
	// if the first reply doesn't return after the specified timeout a timeout error will occur
	SubscriptionFirstReplyTimeout = 10 * time.Second
)

func LocalNodeTimePerCu(cu uint64) time.Duration {
	return BaseTimePerCU(cu) + AverageWorldLatency // TODO: remove average world latency once our providers run locally, or allow a flag that says local to make it tight, tighter timeouts are better
}

func BaseTimePerCU(cu uint64) time.Duration {
	return time.Duration(cu * TimePerCU)
}

func GetTimePerCu(cu uint64) time.Duration {
	if LocalNodeTimePerCu(cu) < MinimumTimePerRelayDelay {
		return MinimumTimePerRelayDelay
	}
	return LocalNodeTimePerCu(cu)
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

func CapContextTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if GetRemainingTimeoutFromContext(ctx) > timeout {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func IsTimeout(errArg error) bool {
	if statusCode, ok := status.FromError(errArg); ok {
		if statusCode.Code() == codes.DeadlineExceeded {
			return true
		}
	}
	return false
}

type TimeoutInfo struct {
	CU       uint64
	Hanging  bool
	Stateful uint32
}

func GetTimeoutForProcessing(relayTimeout time.Duration, timeoutInfo TimeoutInfo) time.Duration {
	ctxTimeout := DefaultTimeout
	if timeoutInfo.CU >= 50 { // for heavyish relays we set longish timeout :)
		ctxTimeout = DefaultTimeoutLongIsh
	}
	if timeoutInfo.Hanging || timeoutInfo.CU >= 100 || timeoutInfo.Stateful == CONSISTENCY_SELECT_ALL_PROVIDERS {
		ctxTimeout = DefaultTimeoutLong
	}
	if relayTimeout > ctxTimeout {
		ctxTimeout = relayTimeout
	}
	return ctxTimeout
}
