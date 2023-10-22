package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	subsciptiontypes "github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MonthlyPayout(goCtx context.Context, req *types.QueryMonthlyPayoutRequest) (*types.QueryMonthlyPayoutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var amount uint64

	// get all tracked CU entries
	trackedCuInds := k.subscriptionKeeper.GetAllSubTrackedCuIndices(ctx, "")

	type totalUsedCuInfo struct {
		totalUsedCu uint64
		relevant    bool // sub is relevant if the provider provided service for it
		block       uint64
		providerCu  uint64
	}

	// get a map of sub+chainID to properties for reward calculation
	totalUsedCuMap := map[string]totalUsedCuInfo{}
	for _, ind := range trackedCuInds {
		sub, provider, chainID := subsciptiontypes.DecodeCuTrackerKey(ind)

		cu, block, found, _ := k.subscriptionKeeper.GetTrackedCu(ctx, sub, provider, chainID)
		if found {
			// check if sub got service from provider (mark relevant and keep the provider's CU)
			relevant := false
			var providerCu uint64
			if provider == req.Provider {
				relevant = true
				providerCu = cu
			}

			// count CU (we also count CU of sub that is not relevant because it might got service from many
			// providers, and one of them might be the requested provider)
			key := sub + " " + chainID
			if usedCuInfo, ok := totalUsedCuMap[key]; ok {
				usedCuInfo.totalUsedCu += cu
				usedCuInfo.providerCu += providerCu
				usedCuInfo.relevant = relevant
				usedCuInfo.block = block
				totalUsedCuMap[key] = usedCuInfo
			} else {
				totalUsedCuMap[key] = totalUsedCuInfo{
					totalUsedCu: cu,
					relevant:    relevant,
					block:       block,
					providerCu:  providerCu,
				}
			}
		}
	}

	for key, usedCuInfo := range totalUsedCuMap {
		if usedCuInfo.relevant {
			comps := strings.Split(key, " ")
			sub := comps[0]
			chainID := comps[1]
			totalMonthlyReward, _ := k.subscriptionKeeper.CalcTotalMonthlyReward(ctx, sub, usedCuInfo.block, usedCuInfo.providerCu, usedCuInfo.totalUsedCu)

			providerAddr, err := sdk.AccAddressFromBech32(req.Provider)
			if err != nil {
				return nil, utils.LavaFormatError("invalid provider address", err,
					utils.Attribute{Key: "provider", Value: req.Provider},
				)
			}

			// calculate the provider reward (smaller than totalMonthlyReward because it's shared with delegators)
			providerReward, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, chainID, totalMonthlyReward, subsciptiontypes.ModuleName, true)
			if err != nil {
				return nil, err
			}

			amount += providerReward.Uint64()
		}
	}

	return &types.QueryMonthlyPayoutResponse{Amount: amount}, nil
}
