package common

import (
	"context"
	"math"
	"time"
)

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
