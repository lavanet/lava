package keeper

// Delegation allows securing funds for a specific provider to effectively increase
// its stake so it will be paired with consumers more often. The delegators do not
// transfer the funds to the provider but only bestow the funds with it. In return
// to locking the funds there, delegators get some of the provider’s profit (after
// commission deduction).
//
// The delegated funds are stored in the module's BondedPoolName account. On request
// to terminate the delegation, they are then moved to the modules NotBondedPoolName
// account, and remain locked there for staking.UnbondingTime() witholding period
// before finally released back to the delegator. The timers for bonded funds are
// tracked are indexed by the delegator, provider, and chainID.
//
// The delegation state is stores with fixation using two maps: one for delegations
// indexed by the combination <provider,chainD,delegator>, used to track delegations
// and find/access delegations by provider (and chainID); and another for delegators
// tracking the list of providers for a delegator, indexed by the delegator.

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

const monthHours = 720 // 30 days * 24 hours

// calculate the delegation credit based on the timestamps, and the amounts of delegations
// amounts and credits represent daily value, rounded down
// can be used to calculate the credit for distribution or update the credit fields in the delegation
func (k Keeper) CalculateCredit(ctx sdk.Context, delegation types.Delegation) (credit sdk.Coin, creditTimestampRet int64) {
	// Calculate the credit for the delegation
	currentAmount := delegation.Amount
	creditAmount := delegation.Credit
	// handle uninitialized amounts
	if creditAmount.IsNil() {
		creditAmount = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	}
	if currentAmount.IsNil() {
		// this should never happen, but we handle it just in case
		currentAmount = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	}
	currentTimestamp := ctx.BlockTime().UTC()
	delegationTimestamp := time.Unix(delegation.Timestamp, 0)
	creditTimestamp := time.Unix(delegation.CreditTimestamp, 0)
	// we normalize dates before we start the calculation
	// maximum scope is 30 days, we start with the delegation truncation then the credit
	monthAgo := currentTimestamp.AddDate(0, 0, -30)
	if monthAgo.Unix() >= delegationTimestamp.Unix() {
		// in the case the delegation wasn't changed for 30 days or more we truncate the timestamp to 30 days ago
		// and disable the credit for older dates since they are irrelevant
		delegationTimestamp = monthAgo
		creditTimestamp = delegationTimestamp
		creditAmount = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	} else if monthAgo.Unix() > creditTimestamp.Unix() {
		// delegation is less than 30 days, but credit might be older, so truncate it to 30 days
		creditTimestamp = monthAgo
	}

	creditDelta := int64(0) // hours
	if delegation.CreditTimestamp == 0 || creditAmount.IsZero() {
		// in case credit was never set, we set it to the delegation timestamp
		creditTimestamp = delegationTimestamp
	} else if creditTimestamp.Before(delegationTimestamp) {
		// calculate the credit delta in hours
		creditDelta = int64(delegationTimestamp.Sub(creditTimestamp).Hours())
	}

	amountDelta := int64(0) // hours
	if !currentAmount.IsZero() && delegationTimestamp.Before(currentTimestamp) {
		amountDelta = int64(currentTimestamp.Sub(delegationTimestamp).Hours())
	}

	// creditDelta is the weight of the history and amountDelta is the weight of the current amount
	// we need to average them and store it in the credit
	totalDelta := creditDelta + amountDelta
	if totalDelta == 0 {
		return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), currentTimestamp.Unix()
	}
	credit = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), currentAmount.Amount.MulRaw(amountDelta).Add(creditAmount.Amount.MulRaw(creditDelta)).QuoRaw(totalDelta))
	return credit, creditTimestamp.Unix()
}

// this function takes the delegation and returns it's credit within the last 30 days
func (k Keeper) CalculateMonthlyCredit(ctx sdk.Context, delegation types.Delegation) (credit sdk.Coin) {
	credit, creditTimeEpoch := k.CalculateCredit(ctx, delegation)
	if credit.IsNil() || credit.IsZero() || creditTimeEpoch <= 0 {
		return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	}
	creditTimestamp := time.Unix(creditTimeEpoch, 0)
	timeStampDiff := ctx.BlockTime().UTC().Sub(creditTimestamp).Hours()
	if timeStampDiff <= 0 {
		// no positive credit
		return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	}
	// make sure we never increase the credit
	if timeStampDiff > monthHours {
		timeStampDiff = monthHours
	}
	// normalize credit to 30 days
	credit.Amount = credit.Amount.MulRaw(int64(timeStampDiff)).QuoRaw(monthHours)
	return credit
}
