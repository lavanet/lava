package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MinStake(ctx),
		k.CoinsPerCU(ctx),
		k.UnstakeHoldBlocks(ctx),
		k.FraudStakeSlashingFactor(ctx),
		k.FraudSlashingAmount(ctx),
		k.SessionBlocks(ctx),
		k.SessionsToSave(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MinStake returns the MinStake param
func (k Keeper) MinStake(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyMinStake, &res)
	return
}

func (k Keeper) GetMinStake(ctx sdk.Context) (res sdk.Coin) {
	var val uint64
	k.paramstore.Get(ctx, types.KeyMinStake, &val)
	res = sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(val)}
	return
}

// CoinsPerCU returns the CoinsPerCU param
func (k Keeper) CoinsPerCU(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyCoinsPerCU, &res)
	return
}

func (k Keeper) GetCoinsPerCU(ctx sdk.Context) (res float64) {
	var val uint64
	k.paramstore.Get(ctx, types.KeyCoinsPerCU, &val)
	res = float64(val) / float64(types.PrecisionForCoinsPerCU)
	return
}

// UnstakeHoldBlocks returns the UnstakeHoldBlocks param
func (k Keeper) UnstakeHoldBlocks(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyUnstakeHoldBlocks, &res)
	return
}

// FraudStakeSlashingFactor returns the FraudStakeSlashingFactor param
func (k Keeper) FraudStakeSlashingFactor(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyFraudStakeSlashingFactor, &res)
	return
}

// FraudSlashingAmount returns the FraudSlashingAmount param
func (k Keeper) FraudSlashingAmount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyFraudSlashingAmount, &res)
	return
}

// SessionBlocks returns the SessionBlocks param
func (k Keeper) SessionBlocks(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeySessionBlocks, &res)
	return
}

// return the next session start
func (k Keeper) IsSessionStart(ctx sdk.Context) (res bool) {
	blocksCycle := k.SessionBlocks(ctx)
	currentBlock := uint64(ctx.BlockHeight())
	//current block modulu blocks cycle returns how many block in the current session we are, if its 0 we are at session start
	return (currentBlock % blocksCycle) == 0
}

// SessionsToSave returns the SessionsToSave param
func (k Keeper) SessionsToSave(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeySessionsToSave, &res)
	return
}

func (k Keeper) BlocksToSave(ctx sdk.Context) (res uint64) {
	blocksToSave := k.SessionsToSave(ctx) * k.SessionBlocks(ctx)
	return blocksToSave
}
