package types

import (
	"strings"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	MAX_SUBSCRIPTION_DURATION = 12 // max duration of subscription in months
)

func (sub Subscription) IsMonthExpired(date time.Time) bool {
	expiry := time.Unix(int64(sub.MonthExpiryTime), 0).UTC()
	return expiry.Before(date)
}

func (sub Subscription) IsStale(block uint64) bool {
	return sub.DurationLeft == 0 && sub.PrevExpiryBlock < block
}

// ValidateSubscription validates a subscription object fields
func (sub Subscription) ValidateSubscription() error {
	// PlanIndex may not be blank
	if len(strings.TrimSpace(sub.PlanIndex)) == 0 {
		return sdkerrors.Wrap(ErrBlankParameter, "subscription plan cannot be blank")
	}

	// Creator may not be blank
	if len(strings.TrimSpace(sub.Creator)) == 0 {
		return sdkerrors.Wrap(ErrBlankParameter, "subscription creator cannot be blank")
	}

	// Consumer may not be blank
	if len(strings.TrimSpace(sub.Consumer)) == 0 {
		return sdkerrors.Wrap(ErrInvalidParameter, "subscription consumer cannot be blank")
	}

	// DurationTotal must be between 1 and MAX_SUBSCRIPTION_DURATION
	if sub.DurationTotal > MAX_SUBSCRIPTION_DURATION {
		return sdkerrors.Wrap(ErrInvalidParameter, "subscription duration (total) is out of range")
	}
	// DurationLeft must be between 1 and MAX_SUBSCRIPTION_DURATION+1
	if sub.DurationLeft > MAX_SUBSCRIPTION_DURATION+1 {
		return sdkerrors.Wrap(ErrInvalidParameter, "subscription duration (left) is out of range")
	}

	return nil
}
