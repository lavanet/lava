package types

import (
	"strings"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (sub Subscription) IsExpired(date time.Time) bool {
	expiry := time.Unix(int64(sub.ExpiryTime), 0).UTC()
	return expiry.Before(date)
}

// ValidateSubscription validates a subscription object fields
func (sub Subscription) ValidateSubscription() error {
	// PlanIndex may not be blank
	if len(strings.TrimSpace(sub.PlanIndex)) == 0 {
		return sdkerrors.Wrap(ErrBlankParameter, "subscription package cannot be blank")
	}

	// Creator may not be blank
	if len(strings.TrimSpace(sub.Creator)) == 0 {
		return sdkerrors.Wrap(ErrBlankParameter, "subscription creator cannot be blank")
	}

	// Consumer may not be blank
	if len(strings.TrimSpace(sub.Consumer)) == 0 {
		return sdkerrors.Wrap(ErrBlankParameter, "subscription consumer cannot be blank")
	}

	return nil
}
