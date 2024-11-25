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
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

// calculate the delegation credit based on the timestamps, and the amounts of delegations
// amounts and credits represent daily value, rounded down
// can be used to calculate the credit for distribution or update the credit fields in the delegation
func (k Keeper) CalculateCredit(ctx sdk.Context, delegation types.Delegation) (credit sdk.Coin, creditTimestampRet int64) {
	// Calculate the credit for the delegation
	currentAmount := delegation.Amount
	creditAmount := delegation.Credit
	currentTimestamp := ctx.BlockTime().UTC()
	delegationTimestap := time.Unix(delegation.Timestamp, 0)
	creditTimestamp := time.Unix(delegation.CreditTimestamp, 0)
	// we normalize dates before we start the calculation
	// maximum scope is 30 days, we start with the delegation truncation then the credit
	monthAgo := currentTimestamp.AddDate(0, 0, -30)
	if monthAgo.Unix() > delegationTimestap.Unix() {
		delegationTimestap = monthAgo
		// set no credit if the delegation is older than 30 days
		creditAmount = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	} else if monthAgo.Unix() > creditTimestamp.Unix() {
		creditTimestamp = monthAgo
	}
	creditDelta := int64(0) // hours
	if !creditAmount.IsNil() && !creditAmount.IsZero() {
		if creditTimestamp.Before(delegationTimestap) {
			creditDelta = int64(delegationTimestap.Sub(creditTimestamp).Hours())
		}
	} else {
		// handle uninitialized credit
		creditAmount = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	}
	amountDelta := int64(0) // hours
	if !currentAmount.IsZero() {
		if delegationTimestap.Before(currentTimestamp) {
			amountDelta = int64(currentTimestamp.Sub(delegationTimestap).Hours())
		}
	}

	// creditDelta is the weight of the history and amountDelta is the weight of the current amount
	// we need to average them and store it in the credit
	totalDelta := creditDelta + amountDelta
	if totalDelta == 0 {
		return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), currentTimestamp.Unix()
	}
	fmt.Println("creditDelta", creditDelta, "amountDelta", amountDelta, "totalDelta", totalDelta, "creditAmount", creditAmount.Amount, "currentAmount", currentAmount.Amount)
	credit = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), currentAmount.Amount.MulRaw(amountDelta).Add(creditAmount.Amount.MulRaw(creditDelta)).QuoRaw(totalDelta))
	return credit, creditTimestamp.Unix()
}
