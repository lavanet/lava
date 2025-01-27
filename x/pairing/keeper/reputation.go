package keeper

import (
	"fmt"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/x/pairing/types"
)

/*

Reputation is a provider performance metric calculated using QoS excellence reports that are retrieved from relay payments.
Higher reputation improves the provider's chance to be picked in the pairing mechanism.

The reputations are kept within a map called "reputations" in the keeper. The map's keys are a collection of the chain ID,
cluster, and the provider address.

The reputation's pairing score is kept in the reputations fixation store so pairing queries will be deterministic for
past blocks.

*/

// GetReputation gets a Reputation from the store
func (k Keeper) GetReputation(ctx sdk.Context, chainID string, cluster string, provider string) (types.Reputation, bool) {
	key := types.ReputationKey(chainID, cluster, provider)
	r, err := k.reputations.Get(ctx, key)
	if err != nil {
		utils.LavaFormatDebug("GetReputation: reputation not found",
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

// UpdateReputationEpochQosScore updates the epoch QoS score of the provider's reputation using the score from the relay
// payment's QoS excellence report
func (k Keeper) UpdateReputationEpochQosScore(ctx sdk.Context, chainID string, cluster string, provider string, score math.LegacyDec, weight int64, stake sdk.Coin) {
	// get current reputation and get parameters for the epoch score update
	r, found := k.GetReputation(ctx, chainID, cluster, provider)
	truncate := false
	if found {
		stabilizationPeriod := k.ReputationVarianceStabilizationPeriod(ctx)
		if r.ShouldTruncate(stabilizationPeriod, ctx.BlockTime().UTC().Unix()) {
			truncate = true
		}
	} else {
		// new reputation score is not truncated and its decay factor is equal to 1
		r = types.NewReputation(ctx)
	}

	// calculate the updated QoS epoch score
	r.EpochScore.Update(score, truncate, weight)

	// update the reputation's stake and set
	r.Stake = stake
	k.SetReputation(ctx, chainID, cluster, provider, r)
}

type ProviderQosScore struct {
	Provider string
	Score    types.QosScore
	Stake    sdk.Coin
}

type StakeProviderScores struct {
	ProviderScores []ProviderQosScore
	TotalStake     sdk.Coin
}

// UpdateAllReputationQosScore updates all the reputations on epoch start with the epoch score aggregated over the epoch
func (k Keeper) UpdateAllReputationQosScore(ctx sdk.Context) {
	// scores is a map of "chainID cluster" -> stakeProviderScores
	// it will be used to compare providers QoS scores within the same chain ID and cluster and determine
	// the providers' reputation pairing score.
	// note, the map is already sorted by QoS score in ascending order.
	scores, err := k.UpdateReputationsForEpochStart(ctx)
	if err != nil {
		utils.LavaFormatError("critical: UpdateReputationQosScore: could not update providers QoS scores", err)
	}

	// sort keys
	keys := []types.ReputationChainClusterKey{}
	for key := range scores {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].ChainID+" "+keys[i].Cluster < keys[j].ChainID+" "+keys[j].Cluster
	})

	// iterate over providers QoS scores with the same chain ID and cluster
	for _, chainCluster := range keys {
		stakeProvidersScore := scores[chainCluster]
		chainID, cluster := chainCluster.ChainID, chainCluster.Cluster

		// get benchmark score value. It's used as a percentile score to determine the reputation pairing score
		// of each provider in a scale of [MinReputationPairingScore, MaxReputationPairingScore] (currently 0.5-2).
		// to clarify, providers score are determined by their QoS score compared to the other providers that share the same chain ID and cluster.
		// the benchmark score is the score of the provider that has the lowest QoS score within the 90th percentile
		// of the providers in the chain ID and cluster.
		benchmark, err := k.GetBenchmarkReputationScore(stakeProvidersScore)
		if err != nil {
			utils.LavaFormatError("critical: UpdateReputationQosScore: could not get benchmark QoS score", err)
		}

		// set reputation pairing score by the benchmark
		err = k.setReputationPairingScoreByBenchmark(ctx, chainID, cluster, benchmark, stakeProvidersScore.ProviderScores)
		if err != nil {
			utils.LavaFormatError("critical: UpdateReputationQosScore: could not set repuatation pairing scores", err)
		}
	}
}

// UpdateReputationsForEpochStart does the following for each reputation:
// 1. applies time decay
// 2. resets the reputation epoch score
// 3. updates it last update time
// 4. add it to the scores map
func (k Keeper) UpdateReputationsForEpochStart(ctx sdk.Context) (map[types.ReputationChainClusterKey]StakeProviderScores, error) {
	halfLifeFactor := k.ReputationHalfLifeFactor(ctx)
	currentTime := ctx.BlockTime().UTC().Unix()

	scores := map[types.ReputationChainClusterKey]StakeProviderScores{}

	// iterate over all reputations
	iter, err := k.reputations.Iterate(ctx, nil)
	if err != nil {
		return nil, utils.LavaFormatError("updateReputationsScores: failed to create reputations iterator", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return nil, utils.LavaFormatError("updateReputationsScores: failed to get reputation key from iterator", err)
		}
		chainID := key.K1()
		cluster := key.K2()
		provider := key.K3()

		reputation, err := iter.Value()
		if err != nil {
			return nil, utils.LavaFormatError("updateReputationsScores: failed to get reputation from iterator", err,
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("cluster", cluster),
				utils.LogAttr("provider", provider),
			)
		}

		if reputation.EpochScore == types.ZeroQosScore {
			// if the epoch score is zero, we don't need to update the reputation
			utils.LavaFormatDebug("updateReputationsScores: epoch score is zero, skipping update",
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("cluster", cluster),
				utils.LogAttr("provider", provider),
			)
			continue
		}

		// apply time decay on current score and add the epoch score (which is reset right after)
		reputation, err = reputation.ApplyTimeDecayAndUpdateScore(utils.SafeUint64ToInt64Convert(halfLifeFactor), currentTime)
		if err != nil {
			return nil, utils.LavaFormatError("updateReputationsScores: apply time decay and update reputation", err,
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("cluster", cluster),
				utils.LogAttr("provider", provider),
			)
		}

		// reset epoch score, update last update time and set the reputation
		reputation.EpochScore = types.ZeroQosScore
		reputation.TimeLastUpdated = currentTime
		k.SetReputation(ctx, chainID, cluster, provider, reputation)

		// add entry to the scores map
		providerScores, ok := scores[types.ReputationChainClusterKey{ChainID: chainID, Cluster: cluster}]
		if !ok {
			providerScores.ProviderScores = []ProviderQosScore{{Provider: provider, Score: reputation.Score, Stake: reputation.Stake}}
			providerScores.TotalStake = reputation.Stake
		} else {
			providerScores.ProviderScores = append(providerScores.ProviderScores, ProviderQosScore{Provider: provider, Score: reputation.Score, Stake: reputation.Stake})
			providerScores.TotalStake = providerScores.TotalStake.Add(reputation.Stake)
		}
		scores[types.ReputationChainClusterKey{ChainID: chainID, Cluster: cluster}] = providerScores
	}

	// in the provider scoring process, each provider is scored by its QoS score compared to the other providers
	// that share the same chain ID and cluster.
	// sortProviderScores() sorts the providers by their QoS score in ascending order so that the best score is first
	// (low is better). This allows us to get the benchmark score more easily.
	sortProviderScores(scores)
	return scores, nil
}

// sortProviderScores sorts the stakeProviderScores map score slices in ascending order
func sortProviderScores(scores map[types.ReputationChainClusterKey]StakeProviderScores) {
	for chainCluster, stakeProviderScores := range scores {
		chainID, cluster := chainCluster.ChainID, chainCluster.Cluster

		sort.Slice(stakeProviderScores.ProviderScores, func(i, j int) bool {
			iScore, err := stakeProviderScores.ProviderScores[i].Score.Score.Resolve()
			if err != nil {
				panic(utils.LavaFormatError("UpdateReputationQosScore: cannot sort provider scores", err,
					utils.LogAttr("provider", stakeProviderScores.ProviderScores[i].Provider),
					utils.LogAttr("chain_id", chainID),
					utils.LogAttr("cluster", cluster),
				))
			}

			jScore, err := stakeProviderScores.ProviderScores[j].Score.Score.Resolve()
			if err != nil {
				panic(utils.LavaFormatError("UpdateReputationQosScore: cannot sort provider scores", err,
					utils.LogAttr("provider", stakeProviderScores.ProviderScores[j].Provider),
					utils.LogAttr("chain_id", chainID),
					utils.LogAttr("cluster", cluster),
				))
			}

			// if scores are equal, determine order by address
			if iScore.Equal(jScore) {
				iProvider := stakeProviderScores.ProviderScores[i].Provider
				jProvider := stakeProviderScores.ProviderScores[j].Provider
				return iProvider < jProvider
			}

			return iScore.LT(jScore)
		})
	}
}

// GetBenchmarkReputationScore gets the score that will be used as the normalization factor when converting
// the provider's QoS score to the reputation pairing score.
// To do that, we go over all the QoS scores of providers that share chain ID and cluster from the lowest
// score to the highest (that input stakeProviderScores are sorted). We aggregate the providers stake until
// we pass totalStake * ReputationPairingScoreBenchmarkStakeThreshold (currently equal to 10% of total stake).
// Then, we return the last provider's score as the benchmark
func (k Keeper) GetBenchmarkReputationScore(stakeProviderScores StakeProviderScores) (math.LegacyDec, error) {
	threshold := types.ReputationPairingScoreBenchmarkStakeThreshold.MulInt(stakeProviderScores.TotalStake.Amount)
	aggregatedStake := sdk.ZeroDec()
	scoreBenchmarkIndex := 0
	for i, providerScore := range stakeProviderScores.ProviderScores {
		aggregatedStake = aggregatedStake.Add(providerScore.Stake.Amount.ToLegacyDec())
		if aggregatedStake.GTE(threshold) {
			scoreBenchmarkIndex = i
			break
		}
	}

	benchmark, err := stakeProviderScores.ProviderScores[scoreBenchmarkIndex].Score.Score.Resolve()
	if err != nil {
		return sdk.ZeroDec(), utils.LavaFormatError("getBenchmarkReputationScore: could not resolve benchmark score", err)
	}

	return benchmark, nil
}

// setReputationPairingScoreByBenchmark sets the reputation pairing score using a benchmark score for all providers
// with the same chain ID and cluster.
// The reputation pairing scores are determined as follows: if the provider's QoS score is smaller than the benchmark,
// it gets the max reputation pairing score. If not, it's normalized by the benchmark and scaled to fit the range
// [MinReputationPairingScore, MaxReputationPairingScore]. Note, smaller scores are better.
// To scale, we use the following formula: scaled_score = min_score + (max_score - min_score) * (benchmark / score)
// We divide (benchmark / score) and not the other way around since we expect that benchmark < score.
func (k Keeper) setReputationPairingScoreByBenchmark(ctx sdk.Context, chainID string, cluster string, benchmark math.LegacyDec, scores []ProviderQosScore) error {
	if benchmark.IsNegative() {
		return utils.LavaFormatError("setReputationPairingScoreByBenchmark: cannot set reputation pairing score with benchmark", fmt.Errorf("benchmark is negative"),
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("cluster", cluster),
			utils.LogAttr("benchmark", benchmark.String()),
		)
	}

	scale := types.MaxReputationPairingScore.Sub(types.MinReputationPairingScore)
	for _, providerScore := range scores {
		score, err := providerScore.Score.Score.Resolve()
		if err != nil {
			return utils.LavaFormatError("setReputationPairingScoreByBenchmark: cannot resolve provider score", err,
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("cluster", cluster),
				utils.LogAttr("provider", providerScore.Provider),
			)
		}

		if score.IsNegative() {
			return utils.LavaFormatError("setReputationPairingScoreByBenchmark: invalid provider score", fmt.Errorf("score is negative"),
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("cluster", cluster),
				utils.LogAttr("provider", providerScore.Provider),
				utils.LogAttr("score", score.String()),
			)
		}

		scaledScore := types.MinReputationPairingScore
		if score.IsZero() || score.LTE(benchmark) {
			// since the benchmark is a very high percentile score, we set the scaled score to the max
			// for providers that have a score lower than the benchmark.
			scaledScore = types.MaxReputationPairingScore
		} else if score.GT(benchmark) {
			scaledScore = types.MinReputationPairingScore.Add((benchmark.Quo(score)).Mul(scale))
		}

		err = k.SetReputationScore(ctx, chainID, cluster, providerScore.Provider, scaledScore)
		if err != nil {
			return utils.LavaFormatError("setReputationPairingScoreByBenchmark: set reputation pairing score failed", err,
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("cluster", cluster),
				utils.LogAttr("provider", providerScore.Provider),
			)
		}
	}

	return nil
}

// GetReputationScore returns the current reputation pairing score
func (k Keeper) GetReputationScore(ctx sdk.Context, chainID string, cluster string, provider string) (val math.LegacyDec, found bool) {
	block := uint64(ctx.BlockHeight())
	key := types.ReputationScoreKey(chainID, cluster, provider)

	var score types.ReputationPairingScore
	found = k.reputationsFS.FindEntry(ctx, key, block, &score)

	return score.Score, found
}

// GetReputationScore returns a reputation pairing score in a specific block
func (k Keeper) GetReputationScoreForBlock(ctx sdk.Context, chainID string, cluster string, provider string, block uint64) (val math.LegacyDec, entryBlock uint64, found bool) {
	var score types.ReputationPairingScore
	key := types.ReputationScoreKey(chainID, cluster, provider)

	entryBlock, _, _, found = k.reputationsFS.FindEntryDetailed(ctx, key, block, &score)
	return score.Score, entryBlock, found
}

// SetReputationScore sets a reputation pairing score
func (k Keeper) SetReputationScore(ctx sdk.Context, chainID string, cluster string, provider string, score math.LegacyDec) error {
	key := types.ReputationScoreKey(chainID, cluster, provider)
	reputationScore := types.ReputationPairingScore{Score: score}
	err := k.reputationsFS.AppendEntry(ctx, key, uint64(ctx.BlockHeight()), &reputationScore)
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
