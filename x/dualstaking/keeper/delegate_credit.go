package keeper

// The credit mechanism is designed to fairly distribute rewards to delegators
// based on both the amount of tokens they delegate and the duration of their
// delegation. It ensures that rewards are proportional to the effective stake
// over time, rather than just the nominal amount of tokens delegated.
//
// Key Components:
// - Credit: Represents the effective delegation for a delegator, adjusted for
//   the time their tokens have been staked.
// - CreditTimestamp: Records the last time the credit was updated, allowing
//   for accurate calculation of rewards over time.
//
// How It Works:
// 1. When a delegation is made or modified, the credit is calculated based on
//    the current amount and the time elapsed since the last update.
// 2. The credit is normalized over a 30-day period to ensure consistent reward
//    distribution.
//
// Example 1:
// Consider two delegators, Alice and Bob, with a total delegators reward pool of 500 tokens.
// - Alice delegates 100 tokens for the full month, earning a credit of 100 tokens.
// - Bob delegates 200 tokens but only for half the month, earning a credit of 100 tokens.
//
// Total credit-adjusted delegations: Alice (100) + Bob (100) = 200 tokens.
// - Alice's reward: (500 * 100 / 200) = 250 tokens
// - Bob's reward: (500 * 100 / 200) = 250 tokens
//
// Example 2 (Mid-Month Delegation Change):
// Suppose Alice initially delegates 100 tokens, and halfway through the month,
// she increases her delegation to 150 tokens.
// - For the first half of the month, Alice's credit is calculated on 100 tokens.
// - When she increases her delegation, the credit is updated to reflect the rewards
//   earned so far (e.g., 50 tokens for 15 days).
// - The CreditTimestamp is updated to the current time.
// - For the remaining half of the month, her credit is calculated on 150 tokens.

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

const (
	monthHours  = 720 // 30 days * 24 hours
	hourSeconds = 3600
)

// CalculateCredit calculates the credit value for a delegation, which represents the
// average stake over time used for reward distribution.
// The credit is normalized according to the difference between credit timestamp and the latest delegation change (in hours)
// The credit is updated only when the delegation amount changes, but is also used to calculate rewards for the current delegation (without updating the entry).
func (k Keeper) CalculateCredit(ctx sdk.Context, delegation types.Delegation) (credit sdk.Coin, creditTimestampRet int64) {
	// Calculate the credit for the delegation
	currentAmount := delegation.Amount
	creditAmount := delegation.Credit

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get bond denom from staking keeper: %v", err))
	}
	// handle uninitialized amounts
	if creditAmount.IsNil() {
		creditAmount = sdk.NewCoin(bondDenom, math.ZeroInt())
	}
	if currentAmount.IsNil() {
		// this should never happen, but we handle it just in case
		currentAmount = sdk.NewCoin(bondDenom, math.ZeroInt())
	}
	currentTimestamp := ctx.BlockTime().UTC()
	delegationTimestamp := time.Unix(delegation.Timestamp, 0)
	creditTimestamp := time.Unix(delegation.CreditTimestamp, 0)
	// we normalize dates before we start the calculation
	// maximum scope is 30 days, we start with the delegation truncation then the credit
	monthAgo := currentTimestamp.AddDate(0, 0, -30) // we are doing 30 days not a month a month can be a different amount of days
	if monthAgo.After(delegationTimestamp) {
		// in the case the delegation wasn't changed for 30 days or more we truncate the timestamp to 30 days ago
		// and disable the credit for older dates since they are irrelevant
		delegationTimestamp = monthAgo
		creditTimestamp = delegationTimestamp
		creditAmount = sdk.NewCoin(bondDenom, math.ZeroInt())
	} else if monthAgo.After(creditTimestamp) {
		// delegation is less than 30 days, but credit might be older, so truncate it to 30 days
		creditTimestamp = monthAgo
	}

	creditDelta := int64(0) // hours
	if delegation.CreditTimestamp == 0 || creditAmount.IsZero() {
		// in case credit was never set, we set it to the delegation timestamp
		creditTimestamp = delegationTimestamp
	} else if creditTimestamp.Before(delegationTimestamp) {
		// calculate the credit delta in hours
		creditDelta = (delegationTimestamp.Unix() - creditTimestamp.Unix()) / hourSeconds
	}

	amountDelta := int64(0) // hours
	if !currentAmount.IsZero() && delegationTimestamp.Before(currentTimestamp) {
		amountDelta = (currentTimestamp.Unix() - delegationTimestamp.Unix()) / hourSeconds
	}

	// creditDelta is the weight of the history and amountDelta is the weight of the current amount
	// we need to average them and store it in the credit
	totalDelta := creditDelta + amountDelta
	if totalDelta == 0 {
		return sdk.NewCoin(bondDenom, math.ZeroInt()), currentTimestamp.Unix()
	}
	credit = sdk.NewCoin(bondDenom, currentAmount.Amount.MulRaw(amountDelta).Add(creditAmount.Amount.MulRaw(creditDelta)).QuoRaw(totalDelta))
	return credit, creditTimestamp.Unix()
}

// CalculateMonthlyCredit returns the total credit value for a delegation, normalized over a 30-day period (hours resolution).
// it does so by calculating the historical credit over the difference between the credit timestamp and the delegation timestamp (in hours) and normalizing it to 30 days
// it then adds the current delegation over the difference between the delegation timestamp and now (in hours) and normalizing it to 30 days
// the function does not modify the delegation amounts nor the timestamps, yet calculates the values for distribution with the current time
//
// For example:
// - If a delegator stakes 100 tokens for a full month, their credit will be 100
// - If they stake 100 tokens for half a month, their credit will be 50
// - If they stake 100 tokens for 15 days then increase to 200 tokens, their credit
// will be calculated as: (100 * 15 + 200 * 15) / 30 = 150
func (k Keeper) CalculateMonthlyCredit(ctx sdk.Context, delegation types.Delegation) (credit sdk.Coin) {
	credit, creditTimeEpoch := k.CalculateCredit(ctx, delegation)
	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get bond denom from staking keeper: %v", err))
	}
	if credit.IsNil() || credit.IsZero() || creditTimeEpoch <= 0 {
		return sdk.NewCoin(bondDenom, math.ZeroInt())
	}
	creditTimestamp := time.Unix(creditTimeEpoch, 0)
	timeStampDiff := (ctx.BlockTime().UTC().Unix() - creditTimestamp.Unix()) / hourSeconds
	if timeStampDiff <= 0 {
		// no positive credit
		return sdk.NewCoin(bondDenom, math.ZeroInt())
	}
	// make sure we never increase the credit
	if timeStampDiff > monthHours {
		timeStampDiff = monthHours
	}
	// normalize credit to 30 days
	credit.Amount = credit.Amount.MulRaw(timeStampDiff).QuoRaw(monthHours)
	return credit
}
