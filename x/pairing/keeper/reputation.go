package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

/*

Reputation is a provider performance metric calculated using QoS excellence reports that are retrieved from relay payments.
Higher reputation improves the provider's chance to be picked in the pairing mechanism.

The reputations are kept within an indexed map called "reputations" in the keeper. An indexed map allows accessing map
entries using two type of keys: primary keys and reference keys. The primary keys are the regular map keys, each point
to a single entry. The reference keys can be of various types: unique, multi, and more. In this case, the reputations
indexed map holds "multi" type reference keys. This means that a single reference key returns a group of entries that
fit the reference key.

The map's primary keys are a collection of the chain ID, cluster, and the provider address.
The map's reference keys are a collection of the chain ID and cluster. Using a reference key, we can get a group of entries
that share the same chain ID and cluster.

Since the collections package doesn't support getting the full list of reference keys from an indexed map, we save a KeySet
of the reference keys in the keeper in the "reputationRefKeys" field.

The reputation's pairing score is kept in the reputations fixation store so pairing queries will be deterministic for
past blocks.

*/

// TODO: remove and reimplement in future work
func (k Keeper) GetQos(ctx sdk.Context, chainID string, cluster string, provider string) (qos types.QualityOfServiceReport, err error) {
	return qos, nil
}

// GetReputation gets a Reputation from the store
func (k Keeper) GetReputation(ctx sdk.Context, chainID string, cluster string, provider string) (types.Reputation, bool) {
	key := types.ReputationKey(chainID, cluster, provider)
	r, err := k.reputations.Get(ctx, key)
	if err != nil {
		utils.LavaFormatWarning("GetReputation: reputation not found", err,
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("cluster", cluster),
			utils.LogAttr("provider", provider),
		)
		return types.Reputation{}, false
	}

	return r, true
}

// SetReputation sets a Reputation in the store
func (k Keeper) SetReputation(ctx sdk.Context, chainID string, cluster string, provider string, r types.Reputation) {
	key := types.ReputationKey(chainID, cluster, provider)
	err := k.reputations.Set(ctx, key, r)
	if err != nil {
		panic(fmt.Errorf("SetReputation: failed to set entry with key %v, error: %w", key, err))
	}
	chainClusterKey := collections.Join(chainID, cluster)
	err = k.reputationRefKeys.Set(ctx, chainClusterKey)
	if err != nil {
		panic(err)
	}
}

// RemoveReputation removes a Reputation from the store
func (k Keeper) RemoveReputation(ctx sdk.Context, chainID string, cluster string, provider string) {
	key := types.ReputationKey(chainID, cluster, provider)
	err := k.reputations.Remove(ctx, key)
	if err != nil {
		panic(fmt.Errorf("RemoveReputation: failed to remove entry with key %v, error: %w", key, err))
	}
}

// GetAllReputation gets all the reputation entries from the store for genesis
func (k Keeper) GetAllReputation(ctx sdk.Context) []types.ReputationGenesis {
	iter, err := k.reputations.Iterate(ctx, nil)
	if err != nil {
		panic(fmt.Errorf("GetAllReputation: Failed to create iterator, error: %w", err))
	}
	defer iter.Close()

	entries := []types.ReputationGenesis{}
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			panic(fmt.Errorf("GetAllReputation: Failed to get key from iterator, error: %w", err))
		}

		entry, err := k.reputations.Get(ctx, key)
		if err != nil {
			panic(fmt.Errorf("GetAllReputation: Failed to get entry with key %v, error: %w", key, err))
		}

		entries = append(entries, types.ReputationGenesis{
			ChainId:    key.K1(),
			Cluster:    key.K2(),
			Provider:   key.K3(),
			Reputation: entry,
		})
	}

	return entries
}

// GetReputationScore returns the current reputation pairing score
func (k Keeper) GetReputationScore(ctx sdk.Context, chainID string, cluster string, provider string) (val types.ReputationPairingScore, found bool) {
	block := uint64(ctx.BlockHeight())
	key := types.ReputationScoreKey(chainID, cluster, provider)

	var score types.ReputationPairingScore
	found = k.reputationsFS.FindEntry(ctx, key, block, &score)

	return score, found
}

// GetReputationScore returns a reputation pairing score in a specific block
func (k Keeper) GetReputationScoreForBlock(ctx sdk.Context, chainID string, cluster string, provider string, block uint64) (val types.ReputationPairingScore, entryBlock uint64, found bool) {
	var score types.ReputationPairingScore
	key := types.ReputationScoreKey(chainID, cluster, provider)

	entryBlock, _, _, found = k.reputationsFS.FindEntryDetailed(ctx, key, block, &score)
	return score, entryBlock, found
}

// SetReputationScore sets a reputation pairing score
func (k Keeper) SetReputationScore(ctx sdk.Context, chainID string, cluster string, provider string, score types.ReputationPairingScore) error {
	key := types.ReputationScoreKey(chainID, cluster, provider)
	err := k.reputationsFS.AppendEntry(ctx, key, uint64(ctx.BlockHeight()), &score)
	if err != nil {
		return utils.LavaFormatError("SetReputationScore: set reputation pairing score failed", err,
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("cluster", cluster),
			utils.LogAttr("provider", provider),
			utils.LogAttr("score", score.String()),
		)
	}

	return nil
}

// RemoveReputationScore removes a reputation pairing score
func (k Keeper) RemoveReputationScore(ctx sdk.Context, chainID string, cluster string, provider string) error {
	block := uint64(ctx.BlockHeight())
	nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, block)
	if err != nil {
		return utils.LavaFormatError("RemoveReputationScore: get next epoch failed", err,
			utils.LogAttr("block", block),
		)
	}
	key := types.ReputationScoreKey(chainID, cluster, provider)

	err = k.reputationsFS.DelEntry(ctx, key, nextEpoch)
	if err != nil {
		return utils.LavaFormatError("RemoveReputationScore: delete score failed", err,
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("cluster", cluster),
			utils.LogAttr("provider", provider),
		)
	}
	return nil
}
