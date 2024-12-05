package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderReputationDetails(goCtx context.Context, req *types.QueryProviderReputationDetailsRequest) (*types.QueryProviderReputationDetailsResponse, error) {
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

	// get all the reputations and reputation scores of the requested provider
	res := []types.ReputationDevData{}
	for _, chainID := range chains {
		for _, cluster := range clusters {
			chainClusterRes := types.ReputationDevData{ChainID: chainID, Cluster: cluster}
			score, foundPairingScore := k.GetReputationScore(ctx, chainID, cluster, req.Address)
			reputation, foundReputation := k.GetReputation(ctx, chainID, cluster, req.Address)
			if !foundPairingScore || !foundReputation {
				continue
			}
			chainClusterRes.Reputation = reputation
			chainClusterRes.ReputationPairingScore = types.ReputationPairingScore{Score: score}
			res = append(res, chainClusterRes)
		}
	}

	return &types.QueryProviderReputationDetailsResponse{Data: res}, nil
}
