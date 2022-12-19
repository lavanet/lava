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
		k.ServicersToPairCountRaw(ctx),
		k.EpochBlocksOverlap(ctx),
		k.StakeToMaxCUListRaw(ctx),
		k.UnpayLimit(ctx),
		k.SlashLimit(ctx),
		k.DataReliabilityReward(ctx),
		k.QoSWeight(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MinStakeProvider returns the MinStakeProvider param
func (k Keeper) MinStakeProvider(ctx sdk.Context) (res sdk.Coin) {
	k.paramstore.Get(ctx, types.KeyMinStakeProvider, &res)
	return
}

// MinStakeClient returns the MinStakeClient param
func (k Keeper) MinStakeClient(ctx sdk.Context) (res sdk.Coin) {
	k.paramstore.Get(ctx, types.KeyMinStakeClient, &res)
	return
}

// MintCoinsPerCU returns the MintCoinsPerCU param
func (k Keeper) MintCoinsPerCU(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMintCoinsPerCU, &res)
	return
}

// BurnCoinsPerCU returns the BurnCoinsPerCU param
func (k Keeper) BurnCoinsPerCU(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyBurnCoinsPerCU, &res)
	return
}

// FraudStakeSlashingFactor returns the FraudStakeSlashingFactor param
func (k Keeper) FraudStakeSlashingFactor(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyFraudStakeSlashingFactor, &res)
	return
}

// FraudSlashingAmount returns the FraudSlashingAmount param
func (k Keeper) FraudSlashingAmount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyFraudSlashingAmount, &res)
	return
}

func (k Keeper) ServicersToPairCount(ctx sdk.Context, block uint64) (res uint64, err error) {
	err = k.epochStorageKeeper.GetParamForBlock(ctx, string(types.KeyServicersToPairCount), block, &res)
	return
}

// ServicersToPairCount returns the ServicersToPairCount param
func (k Keeper) ServicersToPairCountRaw(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyServicersToPairCount, &res)
	return
}

func (k Keeper) EpochBlocksOverlap(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocksOverlap, &res)
	return
}

func (k Keeper) StakeToMaxCUList(ctx sdk.Context, block uint64) (res types.StakeToMaxCUList, err error) {
	err = k.epochStorageKeeper.GetParamForBlock(ctx, string(types.KeyStakeToMaxCUList), block, &res)
	return
}

func (k Keeper) StakeToMaxCUListRaw(ctx sdk.Context) (res types.StakeToMaxCUList) {
	k.paramstore.Get(ctx, types.KeyStakeToMaxCUList, &res)
	return
}

func (k Keeper) UnpayLimit(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyUnpayLimit, &res)
	return
}

func (k Keeper) SlashLimit(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeySlashLimit, &res)
	return
}

func (k Keeper) DataReliabilityReward(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyDataReliabilityReward, &res)
	return
}

func (k Keeper) QoSWeight(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyQoSWeight, &res)
	return
}
