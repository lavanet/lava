package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) GetProviderQosMap(ctx sdk.Context, chainID string, cluster string) map[string]pairingtypes.QualityOfServiceReport

func (k Keeper) GetQos(ctx sdk.Context, chainID string, cluster string, provider string) pairingtypes.QualityOfServiceReport
