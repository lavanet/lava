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
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k Keeper) SetDelegationQue(ctx sdk.Context, que types.DelegationQue) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegationQueKeyPrefix))
	b := k.cdc.MustMarshal(&que)
	store.Set([]byte{0}, b)
}

func (k Keeper) GetDelegationQue(ctx sdk.Context) (val types.DelegationQue, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegationQueKeyPrefix))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) DelegationQuePop(ctx sdk.Context) bool {
	que, _ := k.GetDelegationQue(ctx)
	item := que.DeQue()
	k.SetDelegationQue(ctx, que)
	return item
}
