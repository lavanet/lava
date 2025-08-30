package keeper

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	"github.com/lavanet/lava/v5/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	varianceThreshold = float64(1)   // decides if the overall_performance field can be calculated
	percentileRank    = float64(0.8) // rank for percentile to decide whether the overall_performance is "good" or "bad"
	goodScore         = "good"
	badScore          = "bad"
	lowVariance       = "low_variance"
)

func (k Keeper) ProviderReputation(goCtx context.Context, req *types.QueryProviderReputationRequest) (*types.QueryProviderReputationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	chains := []string{req.ChainID}
	if req.ChainID == "*" {
		chains = k.specKeeper.GetAllChainIDs(ctx)
	}

	clusters := []string{req.Cluster}
	if req.Cluster == "*" {
		clusters = k.subscriptionKeeper.GetAllClusters(ctx)
	}

	// get all the reputation scores of the requested provider and gather valid chainID+cluster pairs
	type chainClusterScore struct {
		chainID string
		cluster string
		score   math.LegacyDec
	}
	requestedProviderData := []chainClusterScore{}
	for _, chainID := range chains {
		for _, cluster := range clusters {
			score, found := k.GetReputationScore(ctx, chainID, cluster, req.Provider)
			if !found {
				continue
			}
			requestedProviderData = append(requestedProviderData, chainClusterScore{chainID: chainID, cluster: cluster, score: score})
		}
	}

	// get scores from other providers for the relevant chains and clusters
	res := []types.ReputationData{}
	for _, data := range requestedProviderData {
		chainClusterRes := types.ReputationData{ChainID: data.chainID, Cluster: data.cluster}

		// get all reputation pairing score indices for a chainID+cluster pair
		inds := k.reputationsFS.GetAllEntryIndicesWithPrefix(ctx, types.ReputationScoreKey(data.chainID, data.cluster, ""))

		// collect all pairing scores with indices and sort in ascending order
		pairingScores := []float64{}
		for _, ind := range inds {
			var score types.ReputationPairingScore
			found := k.reputationsFS.FindEntry(ctx, ind, uint64(ctx.BlockHeight()), &score)
			if !found {
				return nil, utils.LavaFormatError("invalid reputationFS state", fmt.Errorf("reputation pairing score not found"),
					utils.LogAttr("index", ind),
					utils.LogAttr("block", ctx.BlockHeight()),
				)
			}
			pairingScores = append(pairingScores, score.Score.MustFloat64())
		}
		// Sort in descending order (highest scores first)
		sort.Slice(pairingScores, func(i, j int) bool {
			return pairingScores[i] > pairingScores[j]
		})

		// find the provider's rank (1 is best)
		rank := 1
		for i, score := range pairingScores {
			if data.score.MustFloat64() >= score {
				rank = i + 1
				break
			}
		}

		// calculate the pairing scores variance
		mean := lavaslices.Average(pairingScores)
		variance := lavaslices.Variance(pairingScores, mean)

		// Calculate the 80th percentile threshold
		percentileThreshold := lavaslices.Percentile(pairingScores, percentileRank, true)

		// create the reputation data and append
		chainClusterRes.Rank = uint64(rank)
		chainClusterRes.Providers = uint64(len(pairingScores))

		// Compare the provider's score against the threshold
		if data.score.MustFloat64() >= percentileThreshold {
			chainClusterRes.OverallPerformance = goodScore
			if variance < varianceThreshold {
				chainClusterRes.OverallPerformance += " (" + lowVariance + ")"
			}
		} else {
			chainClusterRes.OverallPerformance = badScore
			if variance < varianceThreshold {
				chainClusterRes.OverallPerformance += " (" + lowVariance + ")"
			}
		}

		res = append(res, chainClusterRes)
	}

	return &types.QueryProviderReputationResponse{Data: res}, nil
}
