package common

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/gogo/status"
	"github.com/lavanet/lava/v5/utils"
	"google.golang.org/grpc/codes"
)

const (
	TimePerCU                           = uint64(100 * time.Millisecond)
	CacheWriteTimeout                   = 5 * time.Second
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
// It can be overridden via the --default-processing-timeout flag on consumer and smart router commands.
var DefaultTimeout = time.Duration(DefaultTimeoutSeconds) * time.Second

// MinimumTimePerRelayDelay is the minimum relay timeout floor used by GetTimePerCu.
// It can be overridden via the --min-relay-timeout flag on consumer and smart router commands.
var MinimumTimePerRelayDelay = time.Second

// ValidateAndCapMinRelayTimeout ensures both DefaultTimeout and MinimumTimePerRelayDelay
// are positive and that MinimumTimePerRelayDelay < DefaultTimeout. Called once at startup
// after flags are parsed.
func ValidateAndCapMinRelayTimeout() {
	// Guard DefaultTimeout < 1s: GetTimeoutForProcessing feeds into
	// CapContextTimeout — values below 1s cause immediate DeadlineExceeded on every relay.
	reset := time.Duration(DefaultTimeoutSeconds) * time.Second
	if DefaultTimeout < time.Second {
		utils.LavaFormatWarning("default-processing-timeout is unreasonably small, resetting to default",
			nil,
			utils.LogAttr("invalid_value", DefaultTimeout),
			utils.LogAttr("reset_to", reset),
		)
		DefaultTimeout = reset
	}

	if MinimumTimePerRelayDelay >= DefaultTimeout {
		capped := DefaultTimeout / 2
		// Integer division of a very small DefaultTimeout (< 2ns) rounds to 0.
		// Clamp to at least 1ms so the floor never becomes zero.
		if capped <= 0 {
			capped = time.Millisecond
		}
		utils.LavaFormatWarning("min-relay-timeout >= default-processing-timeout, capping to 50% of processing timeout",
			nil,
			utils.LogAttr("min_relay_timeout", MinimumTimePerRelayDelay),
			utils.LogAttr("default_processing_timeout", DefaultTimeout),
			utils.LogAttr("capped_to", capped),
		)
		MinimumTimePerRelayDelay = capped
	}

	// Guard MinimumTimePerRelayDelay <= 0: GetTimePerCu returns 0 for low-CU methods,
	// which feeds into CapContextTimeout and causes immediate timeouts.
	if MinimumTimePerRelayDelay <= 0 {
		utils.LavaFormatWarning("min-relay-timeout is zero or negative, resetting to 1s",
			nil,
			utils.LogAttr("invalid_value", MinimumTimePerRelayDelay),
		)
		MinimumTimePerRelayDelay = time.Second
	}
}

func LocalNodeTimePerCu(cu uint64) time.Duration {
	return BaseTimePerCU(cu)
}

func BaseTimePerCU(cu uint64) time.Duration {
	return time.Duration(cu * TimePerCU)
}

func GetTimePerCu(cu uint64) time.Duration {
	base := LocalNodeTimePerCu(cu)
	if base < MinimumTimePerRelayDelay {
		return MinimumTimePerRelayDelay
	}
	return base
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
