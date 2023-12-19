package keeper

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

// SetUniquePaymentStorageClientProvider set a specific uniquePaymentStorageClientProvider in the store from its index
func (k Keeper) SetAdjustment(ctx sdk.Context, adjustment types.Adjustment) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AdjustmentKeyPrefix))
	b := k.cdc.MustMarshal(&adjustment)
	store.Set([]byte(adjustment.Index), b)
}

// GetUniquePaymentStorageClientProvider returns a uniquePaymentStorageClientProvider from its index
func (k Keeper) GetAdjustment(
	ctx sdk.Context,
	index string,
) (val types.Adjustment, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AdjustmentKeyPrefix))
	b := store.Get([]byte(index))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveUniquePaymentStorageClientProvider removes a uniquePaymentStorageClientProvider from the store
func (k Keeper) RemoveAdjustment(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AdjustmentKeyPrefix))
	store.Delete([]byte(index))
}

// GetAllUniquePaymentStorageClientProvider returns all uniquePaymentStorageClientProvider
func (k Keeper) GetAllAdjustment(ctx sdk.Context) (list []types.Adjustment) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AdjustmentKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Adjustment
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetProviderFromAdjustment(adjustment *types.Adjustment) (string, error) {
	index := adjustment.Index
	// index consists of chain_epoch_providerAddress
	lastIndex := strings.LastIndex(index, "_")
	if lastIndex != -1 {
		return index[lastIndex+1:], nil
	}
	return "", fmt.Errorf("invalid adjustment key %s", index)
}

func (k Keeper) AdjustmentIndex(consumer string, provider string) string {
	// consumer must come first for the deleteConsumer iteration
	return consumer + "_" + provider
}

func (k Keeper) AppendAdjustment(ctx sdk.Context, consumer string, provider string, totalConsumerUsage uint64, usageWithThisProvider uint64) {
	index := k.AdjustmentIndex(consumer, provider)
	adjustment, _ := k.GetAdjustment(ctx, index)
	// adjustment = weighted average(adjustment/epoch)
	// this epoch adjustment = usageWithThisProvider / totalConsumerUsage
	// adjustment = sum(epoch_adjustment * cu_used_this_epoch) / total_used_cu
	// so we need to save:
	// 1. sum(epoch_adjustment * total_cu_used_this_epoch)
	// 2. total_used_cu = sum(totalConsumerUsage)

	maxRewardsBoost := int64(5) // TODO: yarom, this needs to be read from rewards module

	// check for adjustment limits: adjustment = min(1,1/rewardsMaxBoost * epoch_sum_cu/cu_with_provider)
	if totalConsumerUsage >= uint64(maxRewardsBoost)*usageWithThisProvider {
		// epoch adjustment is 1
		adjustment.TotalUsage += totalConsumerUsage
		adjustment.AdjustedUsage += totalConsumerUsage
	} else {
		// totalConsumerUsage < uint64(maxRewardsBoost)*usageWithThisProvider
		adjustment.TotalUsage += totalConsumerUsage
		// epoch adjustment is (1/maxRewardsBoost * totalConsumerUsage/usageWithThisProvider) * totalConsumerUsage
		adjustment.AdjustedUsage += (totalConsumerUsage / uint64(maxRewardsBoost)) * (totalConsumerUsage / usageWithThisProvider)
	}
	// we need to append, in both cases of existing adjustment or a not found one
	adjustment.Index = index
	k.SetAdjustment(ctx, adjustment)
}

func (k Keeper) GetConsumerAdjustments(ctx sdk.Context, consumer string) (list []types.Adjustment) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AdjustmentKeyPrefix))
	// set consumer prefix
	iterator := sdk.KVStorePrefixIterator(store, []byte(consumer))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Adjustment
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}
	return
}

// assumes consumer comes first in the key, when querying by subscription it will catch all
func (k Keeper) RemoveConsumerAdjustments(ctx sdk.Context, consumer string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AdjustmentKeyPrefix))
	// set consumer prefix
	iterator := sdk.KVStorePrefixIterator(store, []byte(consumer))
	defer iterator.Close()

	keysToDelete := []string{}
	for ; iterator.Valid(); iterator.Next() {
		keysToDelete = append(keysToDelete, string(iterator.Key()))
	}
	for _, key := range keysToDelete {
		k.RemoveAdjustment(ctx, key)
	}
}

func (k Keeper) GetAdjustmentFactorProvider(ctx sdk.Context, adjustments []types.Adjustment) map[string]sdk.Dec {
	type usage struct {
		total    int64
		adjusted int64
	}
	providers := []string{}
	providerUsage := map[string]usage{}
	for _, adjustment := range adjustments {
		provider, err := k.GetProviderFromAdjustment(&adjustment)
		if err != nil {
			utils.LavaFormatError("could not get provider from adjustment", err)
			continue
		}
		usage := providerUsage[provider]
		usage.adjusted += int64(adjustment.AdjustedUsage)
		usage.total += int64(adjustment.TotalUsage)
		providerUsage[provider] = usage
		providers = append(providers, provider)
	}

	providerAdjustment := map[string]sdk.Dec{}
	// we use providers list to iterate deterministically
	for _, provider := range providers {
		if _, ok := providerAdjustment[provider]; !ok {
			totalUsage := providerUsage[provider].total
			totalAdjustedUsage := providerUsage[provider].adjusted
			// indexes may repeat but we only need to handle each provider once
			providerAdjustment[provider] = sdk.OneDec().MulInt64(int64(totalAdjustedUsage)).QuoInt64(int64(totalUsage))
		}
	}
	return providerAdjustment
}
