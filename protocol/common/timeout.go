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
	DefaultTimeoutSeconds               = 30 // default timeout in seconds, can be overridden by flag
	CacheTimeout                        = 50 * time.Millisecond
	// On subscriptions we must use context.Background(),
	// we cant have a context.WithTimeout() context, meaning we can hang for ever.
	// to avoid that we introduced a first reply timeout using a routine.
	// if the first reply doesn't return after the specified timeout a timeout error will occur
	SubscriptionFirstReplyTimeout = 10 * time.Second
)

// DefaultTimeout is the configurable default timeout for relay processing.
// It can be overridden via the --default-timeout flag on consumer and smart router commands.
var DefaultTimeout = time.Duration(DefaultTimeoutSeconds) * time.Second

func LocalNodeTimePerCu(cu uint64) time.Duration {
	return BaseTimePerCU(cu)
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
	if timeoutInfo.CU >= 50 {
		ctxTimeout = DefaultTimeout * 2
	}
	if timeoutInfo.Hanging || timeoutInfo.CU >= 100 || timeoutInfo.Stateful == CONSISTENCY_SELECT_ALL_PROVIDERS {
		ctxTimeout = DefaultTimeout * 6
	}
	if relayTimeout > ctxTimeout {
		ctxTimeout = relayTimeout
	}
	return ctxTimeout
}
