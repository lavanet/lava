package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MinStakeProvider(ctx),
		k.MinStakeClient(ctx),
		k.MintCoinsPerCU(ctx),
		k.BurnCoinsPerCU(ctx),
		k.FraudStakeSlashingFactor(ctx),
		k.FraudSlashingAmount(ctx),
		k.ServicersToPairCount(ctx),
		k.EpochBlocksOverlap(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MinStakeProvider returns the MinStakeProvider param
func (k Keeper) MinStakeProvider(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyMinStakeProvider, &res)
	return
}

// MinStakeClient returns the MinStakeClient param
func (k Keeper) MinStakeClient(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyMinStakeClient, &res)
	return
}

// MintCoinsPerCU returns the MintCoinsPerCU param
func (k Keeper) MintCoinsPerCU(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyMintCoinsPerCU, &res)
	return
}

// BurnCoinsPerCU returns the BurnCoinsPerCU param
func (k Keeper) BurnCoinsPerCU(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyBurnCoinsPerCU, &res)
	return
}

// FraudStakeSlashingFactor returns the FraudStakeSlashingFactor param
func (k Keeper) FraudStakeSlashingFactor(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyFraudStakeSlashingFactor, &res)
	return
}

// FraudSlashingAmount returns the FraudSlashingAmount param
func (k Keeper) FraudSlashingAmount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyFraudSlashingAmount, &res)
	return
}

// ServicersToPairCount returns the ServicersToPairCount param
func (k Keeper) ServicersToPairCount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyServicersToPairCount, &res)
	return
}

// EpochBlocksOverlap returns the EpochBlocksOverlap param
func (k Keeper) EpochBlocksOverlap(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocksOverlap, &res)
	return
}
