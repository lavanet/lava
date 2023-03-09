package types

import (
	"strings"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	MAX_SUBSCRIPTION_DURATION = 12 // max duration of subscription in months
)

func (sub Subscription) IsExpired(date time.Time) bool {
	expiry := time.Unix(int64(sub.ExpiryTime), 0).UTC()
	return expiry.Before(date)
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

	// Duration must be between 1 and MAX_SUBSCRIPTION_DURATION
	if sub.Duration > MAX_SUBSCRIPTION_DURATION {
		return sdkerrors.Wrap(ErrInvalidParameter, "subscription duration is out of range")
	}

	return nil
}
